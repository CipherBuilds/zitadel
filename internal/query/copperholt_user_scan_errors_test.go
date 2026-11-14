//go:build copperholt_zitadel_upstream

// This regression test is copied into the pinned upstream query package during
// the Copperholt image build. It invokes its actual database/sql row scanners.
package query

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/zitadel/zitadel/internal/zerrors"
)

func TestCopperholtUserScanErrorsStayUnconfirmed(t *testing.T) {
	for _, scanner := range []string{"user", "notify_user"} {
		t.Run(scanner, func(t *testing.T) {
			for _, mode := range []string{"present", "absent", "query_error", "conversion_error", "zero_count", "multiple_count"} {
				t.Run(mode, func(t *testing.T) { copperholtCheckUserScan(t, scanner, mode) })
			}
		})
	}
}

func copperholtCheckUserScan(t *testing.T, scanner, mode string) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal("synthetic SQL row fixture unavailable")
	}
	t.Cleanup(func() {
		mock.ExpectClose()
		if db.Close() != nil || mock.ExpectationsWereMet() != nil {
			t.Error("synthetic SQL row fixture did not settle and close")
		}
	})
	prepareCopperholtUserScan(t, mock, scanner, mode)
	row := db.QueryRowContext(t.Context(), "SELECT synthetic_user")
	id, err := copperholtScanUserRow(scanner, row)
	if mode == "present" {
		if err != nil || id != "synthetic-user" {
			t.Fatal("ordinary single-row user stopped working")
		}
		return
	}
	classified, ok := zerrors.AsZitadelError(err)
	if !ok || id != "" {
		t.Fatal("failed user scan returned usable identity or unclassified error")
	}
	if mode == "absent" {
		if classified.Kind != zerrors.KindNotFound {
			t.Fatal("genuine absent row lost its absence classification")
		}
	} else if classified.Kind != zerrors.KindInternal {
		t.Fatal("failed or inconsistent user row became absence")
	}
}

func copperholtScanUserRow(scanner string, row *sql.Row) (string, error) {
	if scanner == "user" {
		user, err := scanUser(row)
		if err != nil {
			return "", err
		}
		return user.ID, nil
	}
	user, err := scanNotifyUser(row)
	if err != nil {
		return "", err
	}
	return user.ID, nil
}

func copperholtUserScanValues(scanner string) ([]string, []driver.Value) {
	count := 31
	if scanner == "notify_user" {
		count = 25
	}
	columns := make([]string, count)
	for index := range columns {
		columns[index] = "synthetic_field"
	}
	values := make([]driver.Value, count)
	values[0], values[1], values[2], values[3] = "synthetic-user", time.Unix(1700000000, 0), time.Unix(1700000000, 0), "synthetic-org"
	values[4], values[5], values[6], values[7] = int64(1), int64(1), int64(1), "synthetic-user"
	values[8], values[9] = "{}", "synthetic-user"
	if scanner == "notify_user" {
		values[18] = "synthetic-user"
	}
	values[count-1] = int64(1)
	return columns, values
}

func prepareCopperholtUserScan(t *testing.T, mock sqlmock.Sqlmock, scanner, mode string) {
	t.Helper()
	columns, values := copperholtUserScanValues(scanner)
	rows := sqlmock.NewRows(columns)
	expected := mock.ExpectQuery("SELECT synthetic_user")
	if mode == "query_error" {
		expected.WillReturnError(errors.New("synthetic query unavailable"))
	} else {
		switch mode {
		case "conversion_error":
			values[1] = "synthetic-invalid-time"
		case "zero_count":
			values[len(values)-1] = int64(0)
		case "multiple_count":
			values[len(values)-1] = int64(2)
		}
		if mode != "absent" {
			rows.AddRow(values...)
		}
		expected.WillReturnRows(rows)
	}
}
