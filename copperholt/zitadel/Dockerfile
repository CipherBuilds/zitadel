# syntax=docker/dockerfile:1.7
# Rebuild the exact audited Zitadel revision using the patched Go toolchain.
# Zitadel's Nx target generates the gRPC, OpenAPI, static, and Console assets
# before compiling, matching the upstream release pipeline.
FROM golang@sha256:db99415e2625ce0ee03eed09edf7f32ae8c58dc399f28cc602a87fe7238119e4 AS go-toolchain

# QA-TP15: the real regression test files never live in this Docker/ build context;
# the build invocation supplies them as the named context `zitadel-tests` (docker
# buildx --build-context zitadel-tests=<TestStudio checkout>), which always takes
# precedence over this empty fallback stage. Declaring the stage here only gives
# `COPY --from=zitadel-tests` a resolvable name; a build run without the named
# context fails closed at that COPY (no file), never by silently pulling an image.
FROM scratch AS zitadel-tests

FROM node@sha256:f70403e87646dc51b45295f4b8b70cdad0b63d2297c4c9899119b03f7af7a6b3 AS zitadel-builder
ARG ZITADEL_VERSION=v4.20.5
ARG ZITADEL_REVISION=59827a8ec6dd0c23acbba5d0ebdf41fdc857d66c
ENV PATH="/usr/local/go/bin:${PATH}" \
    ZITADEL_VERSION="${ZITADEL_VERSION}" \
    GOFLAGS="-buildvcs=false -tags=copperholt_zitadel_upstream" \
    NEXT_TELEMETRY_DISABLED="1" \
    NX_NO_CLOUD="true" \
    NX_DAEMON="false"
COPY --from=go-toolchain /usr/local/go /usr/local/go
WORKDIR /src
RUN apk add --no-cache bash=5.3.9-r1 build-base=0.5-r4 curl=8.23.0-r0 git=2.54.1-r0 gcompat=1.1.0-r4 make=4.4.1-r4
RUN git init \
    && git remote add origin https://github.com/zitadel/zitadel.git \
    && git fetch --depth 1 origin "${ZITADEL_REVISION}" \
    && git checkout --detach FETCH_HEAD
# Generated protobuf packages do not exist until the Nx pack target runs, so a
# pre-generation `go mod tidy` would incorrectly try to resolve Zitadel's own
# packages over the network. `go get` of named modules does not load the main
# package graph, so pin only the repaired modules (and the minimums x/net
# v0.62.0 requires) now; the upstream pack graph generates the packages before
# compiling them. x/net v0.62.0 fixes GO-2026-6603/6611/6612/6617/6610 (http2)
# and GO-2026-6866/6860; klauspost/compress v1.20.2 fixes GO-2026-6865;
# gorilla/csrf v1.7.3 fixes CVE-2025-24358; x/crypto at the GO-2026-5932 fix commit (the Traefik B33 pin, same
# requirements as v0.58.0) drops the unmaintained openpgp packages.
# ZITADEL-UP01/UP02 (v4.20.4, v4.20.5 has the same go.mod; 19 Nov 2026): upstream now pins chi v5.5.0 (GO-2026-5774/5775/5777, GHSA-j6qf-3m9c-vwj3),
# go-jose/v4 v4.1.7 (GHSA-mgcc-cww5-fp2q fixed in v4.1.6), grpc v1.84.0 (CVE-2026-84445 fixed in v1.83.2) and
# x/text v0.43.0, so those are no longer repaired here; the version checks below keep them as floors.
RUN go get \
      github.com/gorilla/csrf@v1.7.3 \
      github.com/klauspost/compress@v1.20.2 \
      golang.org/x/crypto@v0.58.1-0.20261009181718-2f1f834330b2 \
      golang.org/x/net@v0.62.0
RUN --mount=type=cache,id=zitadel-pnpm-store,target=/root/.local/share/pnpm/store \
    corepack enable && pnpm install --frozen-lockfile
# AGPL-3.0 section 13 source offer (LIC-FIX01, Compliance FINAL wording of 14 Nov 2026) on every page this image
# serves: the Management Console footer (on every console page) and the login v1 footer (every login v1 page renders
# it through main-bottom), each with "Source code" linking to the published fork. Applied before generation, which
# compiles the login v1 stylesheet and templates into statik and builds the console that the binary embeds; the
# regression below reads both as built.
COPY core-source-offer.patch core-source-offer.before.sha256 core-source-offer.after.sha256 /opt/copperholt-core-source-offer/
COPY --from=zitadel-tests source_offer_test.go /src/internal/api/ui/console/copperholt_source_offer_test.go
RUN sha256sum -c /opt/copperholt-core-source-offer/core-source-offer.before.sha256 \
    && git apply --check --whitespace=error-all /opt/copperholt-core-source-offer/core-source-offer.patch \
    && git apply --whitespace=error-all /opt/copperholt-core-source-offer/core-source-offer.patch \
    && sha256sum -c /opt/copperholt-core-source-offer/core-source-offer.after.sha256 \
    && rm -r /opt/copperholt-core-source-offer
# The pinned Console protobuf plugins are glibc-linked executables. Alpine's
# gcompat loader above is required; without it exec reports a misleading ENOENT.
# The upstream task's cache metadata does not declare every generated Go tool.
# A remote cache hit can therefore omit statik and make a later generator fail.
# Execute generation and console assembly first, reconcile the now-complete Go
# package graph, then run only the final packaging target.
RUN --mount=type=cache,id=zitadel-go-mod,target=/go/pkg/mod \
    --mount=type=cache,id=zitadel-go-build,target=/root/.cache/go-build \
    pnpm nx run --nxBail --skip-nx-cache --parallel=1 @zitadel/api:generate \
    && pnpm nx run --nxBail --skip-nx-cache --parallel=1 @zitadel/api:build-console
# Only genuine no-row results establish absence. Query/scan errors and corrupt
# counts must not authorize downstream erasure. Recheck the exact audited source
# before the narrow patch, and require the real upstream regression before pack.
# QA-TP15: the two regression test files are maintained only under
# TestStudio/tests/shared/go/Zitadel/ (migrations.json, module "Docker/Zitadel");
# the build reads them from that additional context, never a Docker/ copy.
COPY user_scan_errors.patch /opt/copperholt-build/user_scan_errors.patch
COPY --from=zitadel-tests user_scan_errors_test.go /src/internal/query/copperholt_user_scan_errors_test.go
COPY legacy_token_errors.patch /opt/copperholt-build/legacy_token_errors.patch
COPY --from=zitadel-tests legacy_token_errors_test.go /src/internal/auth/repository/eventsourcing/eventstore/copperholt_legacy_token_errors_test.go
# QA-INS-11 log scan (9 Nov 2026): a missing message translation logged the whole template data (names, email,
# phone, login names and the one-time code) at warning level. Log only the message ID and error.
COPY translator_log_args.patch /opt/copperholt-build/translator_log_args.patch
SHELL ["/bin/bash", "-o", "pipefail", "-c"]
RUN --mount=type=cache,id=zitadel-go-mod,target=/go/pkg/mod \
    --mount=type=cache,id=zitadel-go-build,target=/root/.cache/go-build \
    printf '%s  %s\n' \
        '24b1f7d7b9b445a8553bcbca98615af9e205ed0b2819a9c266ef4367ae1ed9fd' \
        'internal/query/user.go' | sha256sum -c - \
    && git apply --check --whitespace=error-all /opt/copperholt-build/user_scan_errors.patch \
    && git apply --whitespace=error-all /opt/copperholt-build/user_scan_errors.patch \
    && printf '%s  %s\n' \
        '9bee9dd44f8e4c53191fba05a996316ced6240fbf91a5f259c54b8398267b3c6' \
        'internal/auth/repository/eventsourcing/eventstore/token.go' | sha256sum -c - \
    && git apply --check --whitespace=error-all /opt/copperholt-build/legacy_token_errors.patch \
    && git apply --whitespace=error-all /opt/copperholt-build/legacy_token_errors.patch \
    && printf '%s  %s\n' \
        'ab69dceffab3b3c52d5dc098b8a9355099c62439fb501a898ca7cd2904574c74' \
        'internal/i18n/translator.go' | sha256sum -c - \
    && git apply --check --whitespace=error-all /opt/copperholt-build/translator_log_args.patch \
    && git apply --whitespace=error-all /opt/copperholt-build/translator_log_args.patch \
    && ! grep -n '"args", args' internal/i18n/translator.go \
    && go mod tidy \
    && GOMAXPROCS=2 go vet -p 2 ./internal/auth/repository/eventsourcing/eventstore \
    && GOMAXPROCS=2 go test -p 2 -race -count=1 -timeout=180s -v \
        -run '^TestCopperholtLegacyTokenErrorsStayUnconfirmed$' \
        ./internal/auth/repository/eventsourcing/eventstore | tee /tmp/copperholt-legacy-token.log \
    && grep -q -- '^--- PASS: TestCopperholtLegacyTokenErrorsStayUnconfirmed ' /tmp/copperholt-legacy-token.log \
    && test "$(go list -m -f '{{.Version}}' google.golang.org/grpc)" = v1.84.0 \
    && test "$(go list -m -f '{{.Version}}' golang.org/x/net)" = v0.62.0 \
    && test "$(go list -m -f '{{.Version}}' github.com/klauspost/compress)" = v1.20.2 \
    && test "$(go list -m -f '{{.Version}}' github.com/go-jose/go-jose/v4)" = v4.1.7 \
    && test "$(go list -m -f '{{.Version}}' github.com/gorilla/csrf)" = v1.7.3 \
    && test "$(go list -m -f '{{.Version}}' github.com/go-chi/chi/v5)" = v5.5.0 \
    && test "$(go list -m -f '{{.Version}}' golang.org/x/crypto)" = v0.58.1-0.20261009181718-2f1f834330b2 \
    && test "$(go list -m -f '{{.Version}}' golang.org/x/text)" = v0.43.0 \
    && GOMAXPROCS=2 go vet -p 2 ./internal/query \
    && GOMAXPROCS=2 go test -p 2 -race -count=1 -timeout=180s -v \
        -run '^TestCopperholtUserScanErrorsStayUnconfirmed$' ./internal/query | tee /tmp/copperholt-user-scan.log \
    && grep -q -- '^--- PASS: TestCopperholtUserScanErrorsStayUnconfirmed ' /tmp/copperholt-user-scan.log \
    && GOMAXPROCS=2 go vet -p 2 ./internal/api/ui/console \
    && GOMAXPROCS=2 go test -p 2 -race -count=1 -timeout=180s -v \
        -run '^TestCopperholtSourceOfferOnEveryCorePage$' ./internal/api/ui/console | tee /tmp/copperholt-source-offer.log \
    && grep -q -- '^--- PASS: TestCopperholtSourceOfferOnEveryCorePage ' /tmp/copperholt-source-offer.log \
    && GOMAXPROCS=2 go test -p 2 -race -count=1 -timeout=180s -v \
      -run '^Test/(HeadersHTTPStatusGRPCStatus|RouteAndProcess_MissingAuthority|MissingAuthorityAndHostHeader|AuthorityHeader)$' \
      google.golang.org/grpc/internal/transport google.golang.org/grpc/internal/xds/server google.golang.org/grpc/test \
    && pnpm nx run --nxBail --skip-nx-cache --parallel=1 \
    @zitadel/api:pack-linux-amd64 --excludeTaskDependencies

FROM alpine@sha256:fd791d74b68913cbb027c6546007b3f0d3bc45125f797758156952bc2d6daf40 AS runtime-files
RUN apk add --no-cache ca-certificates=20261020-r0 \
    && addgroup -S -g 1000 zitadel \
    && adduser -S -D -H -u 1000 -G zitadel zitadel

FROM scratch
COPY --from=runtime-files /etc/passwd /etc/passwd
COPY --from=runtime-files /etc/group /etc/group
COPY --from=runtime-files /etc/ssl/certs /etc/ssl/certs
COPY --from=zitadel-builder /src/.artifacts/bin/linux/amd64/zitadel /app/zitadel
COPY zitadel-config.yaml /zitadel-config.yaml
USER zitadel
HEALTHCHECK --interval=30s --timeout=30s --start-period=20s --retries=3 \
  CMD ["/app/zitadel", "ready"]
# ZITADEL-INIT-SPLIT01 (Compliance, LIC-FIX01: none of our code goes into the Zitadel image): the entrypoint is
# the upstream binary. The passwords arrive as upstream configuration in the zitadel_runtime_config secret
# (rendered in memory from Docker secrets; Docker/Zitadel/zitadel-runtime-config.yaml.tmpl), read as both
# --config and --steps. One file for both means a missing or unreadable file fails the database connection
# before any first-instance step, never a default administrator password.
ENTRYPOINT ["/app/zitadel"]
CMD ["start-from-init", "--config", "/zitadel-config.yaml", "--config", "/run/secrets/zitadel_runtime_config", \
     "--steps", "/run/secrets/zitadel_runtime_config", "--masterkeyFile", "/run/secrets/zitadel_masterkey"]
