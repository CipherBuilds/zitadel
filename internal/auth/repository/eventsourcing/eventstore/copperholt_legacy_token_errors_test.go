//go:build copperholt_zitadel_upstream

// Copied into the pinned upstream package during the guarded provider build.
// Tests call the real TokenRepo method, eventstore.Filter and token reducer.
package eventstore

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/zitadel/zitadel/internal/api/authz"
	store "github.com/zitadel/zitadel/internal/eventstore"
	"github.com/zitadel/zitadel/internal/query"
	user "github.com/zitadel/zitadel/internal/repository/user"
	"github.com/zitadel/zitadel/internal/user/repository/view/model"
	"github.com/zitadel/zitadel/internal/zerrors"
)

type copperholtTokenProjection struct {
	t             *testing.T
	token         *model.TokenView
	sequenceError error
	viewError     error
	queries       int
}

func (p *copperholtTokenProjection) GetLatestTokenSequence(_ context.Context, instanceID string) (*query.CurrentState, error) {
	p.queries++
	if instanceID != "fixture-instance" {
		p.t.Error("sequence query lost instance binding")
	}
	return nil, p.sequenceError
}

func (p *copperholtTokenProjection) TokenByIDs(tokenID, userID, instanceID string) (*model.TokenView, error) {
	p.queries++
	if tokenID != "fixture-token" || userID != "fixture-user" || instanceID != "fixture-instance" {
		p.t.Error("projection query lost token/user/instance binding")
	}
	if p.token == nil {
		return nil, p.viewError
	}
	copy := *p.token
	return &copy, p.viewError
}

type copperholtTokenQuerier struct {
	store.Querier
	events []store.Event
	err    error
	cancel context.CancelFunc
	calls  int
}

func (q *copperholtTokenQuerier) FilterToReducer(_ context.Context, _ *store.SearchQueryBuilder, reduce store.Reducer) error {
	q.calls++
	for _, event := range q.events {
		if err := reduce(event); err != nil {
			return err
		}
	}
	if q.cancel != nil {
		q.cancel()
	}
	return q.err
}

func copperholtTokenEvent(kind store.EventType, body string) store.Event {
	return &store.BaseEvent{EventType: kind, Creation: time.Now().UTC(), Agg: &store.Aggregate{ID: "fixture-user", InstanceID: "fixture-instance", ResourceOwner: "fixture-org", Type: user.AggregateType}, Data: []byte(body)}
}

func TestCopperholtLegacyTokenErrorsStayUnconfirmed(t *testing.T) {
	for _, mode := range []string{
		"current", "absent", "revoked", "refresh-revoked", "user-deactivated", "expired",
		"sequence-error", "view-error", "event-error", "partial-event-error",
		"expired-event-error", "malformed-event", "revoked-then-malformed",
		"absent-event-error", "nil-projection", "cancelled-before", "cancelled-after-events",
	} {
		t.Run(mode, func(t *testing.T) { copperholtCheckLegacyToken(t, mode) })
	}
}

func copperholtCheckLegacyToken(t *testing.T, mode string) {
	t.Helper()
	ctx, cancel := context.WithCancel(authz.WithInstanceID(t.Context(), "fixture-instance"))
	defer cancel()
	projection := &copperholtTokenProjection{t: t, token: &model.TokenView{ID: "fixture-token", UserID: "fixture-user", InstanceID: "fixture-instance", RefreshTokenID: "fixture-refresh", Expiration: time.Now().Add(time.Hour)}}
	querier := &copperholtTokenQuerier{}
	privateError := errors.New("synthetic-private-provider-detail")
	missing := zerrors.ThrowNotFound(nil, "fixture-absent", "Errors.Token.NotFound")
	removed := copperholtTokenEvent(user.UserTokenRemovedType, `{"tokenId":"fixture-token"}`)
	malformed := copperholtTokenEvent(user.UserTokenRemovedType, `{`)
	switch mode {
	case "absent":
		projection.token, projection.viewError = nil, missing
	case "revoked":
		querier.events = []store.Event{removed}
	case "refresh-revoked":
		querier.events = []store.Event{copperholtTokenEvent(user.HumanRefreshTokenRemovedType, `{"tokenId":"fixture-refresh"}`)}
	case "user-deactivated":
		querier.events = []store.Event{copperholtTokenEvent(user.UserDeactivatedType, `{}`)}
	case "expired":
		projection.token.Expiration = time.Now().Add(-time.Hour)
	case "sequence-error":
		projection.sequenceError = privateError
	case "view-error":
		projection.viewError = privateError
	case "event-error":
		querier.err = privateError
	case "partial-event-error":
		querier.events, querier.err = []store.Event{removed}, privateError
	case "expired-event-error":
		projection.token.Expiration, querier.err = time.Now().Add(-time.Hour), privateError
	case "malformed-event":
		querier.events = []store.Event{malformed}
	case "revoked-then-malformed":
		querier.events = []store.Event{removed, malformed}
	case "absent-event-error":
		projection.token, projection.viewError, querier.err = nil, missing, privateError
	case "nil-projection":
		projection.token = nil
	case "cancelled-before":
		cancel()
	case "cancelled-after-events":
		querier.cancel = cancel
	}
	repo := &TokenRepo{View: projection, Eventstore: store.NewEventstore(&store.Config{Querier: querier})}
	defer func() {
		if recover() != nil {
			t.Error("uncertain token state panicked instead of denying authority")
		}
	}()
	result, err := repo.TokenByIDs(ctx, "fixture-user", "fixture-token")
	if mode == "current" {
		if err != nil || result == nil || result.ID != "fixture-token" || result.UserID != "fixture-user" || projection.queries != 2 || querier.calls != 1 {
			t.Fatal("current token no longer reconciles successfully")
		}
		return
	}
	if err == nil || result != nil {
		t.Fatal("unconfirmed token state returned usable identity")
	}
	assertLegacyTokenFailure(t, mode, err, privateError, projection, querier)
}

func assertLegacyTokenFailure(t *testing.T, mode string, err, privateError error, projection *copperholtTokenProjection, querier *copperholtTokenQuerier) {
	t.Helper()
	classified, ok := zerrors.AsZitadelError(err)
	if !ok {
		t.Fatal("token rejection has no bounded provider classification")
	}
	switch mode {
	case "absent", "revoked", "refresh-revoked", "user-deactivated", "expired":
		if classified.Kind != zerrors.KindNotFound {
			t.Error("confirmed inactive token lost its denial classification")
		}
	default:
		if classified.Kind != zerrors.KindInternal || strings.Contains(err.Error(), privateError.Error()) {
			t.Error("uncertain state became absence or exposed provider details")
		}
	}
	if mode == "cancelled-before" && (projection.queries != 0 || querier.calls != 0) {
		t.Error("cancelled lookup still dispatched provider reads")
	}
	if mode == "sequence-error" && (projection.queries != 1 || querier.calls != 0) {
		t.Error("failed sequence lookup still used a cached projection")
	}
}
