# syntax=docker/dockerfile:1.7
# Rebuild the exact audited Zitadel revision using the patched Go toolchain.
# Zitadel's Nx target generates the gRPC, OpenAPI, static, and Console assets
# before compiling, matching the upstream release pipeline.
FROM golang@sha256:db99415e2625ce0ee03eed09edf7f32ae8c58dc399f28cc602a87fe7238119e4 AS go-toolchain

FROM node@sha256:f70403e87646dc51b45295f4b8b70cdad0b63d2297c4c9899119b03f7af7a6b3 AS zitadel-builder
ARG ZITADEL_VERSION=v4.16.2
ARG ZITADEL_REVISION=917ade35af9c51e4b57ca8e57488ae95928182ad
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
# go-jose/v4 v4.1.6 fixes GHSA-mgcc-cww5-fp2q (JWK RSA parameter DoS).
RUN go get \
      github.com/go-jose/go-jose/v4@v4.1.6 \
      github.com/gorilla/csrf@v1.7.3 \
      github.com/klauspost/compress@v1.20.2 \
      golang.org/x/crypto@v0.58.0 \
      golang.org/x/net@v0.62.0 \
      golang.org/x/text@v0.43.0 \
      google.golang.org/grpc@v1.83.2
RUN --mount=type=cache,id=zitadel-pnpm-store,target=/root/.local/share/pnpm/store \
    corepack enable && pnpm install --frozen-lockfile
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
        '73a6450abd3f4cdc5ec5f305d69d6f4bf4ecb63f57c0f51c74b71657493eb81d' \
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
    && test "$(go list -m -f '{{.Version}}' google.golang.org/grpc)" = v1.83.2 \
    && test "$(go list -m -f '{{.Version}}' golang.org/x/net)" = v0.62.0 \
    && test "$(go list -m -f '{{.Version}}' github.com/klauspost/compress)" = v1.20.2 \
    && test "$(go list -m -f '{{.Version}}' github.com/go-jose/go-jose/v4)" = v4.1.6 \
    && test "$(go list -m -f '{{.Version}}' github.com/gorilla/csrf)" = v1.7.3 \
    && GOMAXPROCS=2 go vet -p 2 ./internal/query \
    && GOMAXPROCS=2 go test -p 2 -race -count=1 -timeout=180s -v \
        -run '^TestCopperholtUserScanErrorsStayUnconfirmed$' ./internal/query | tee /tmp/copperholt-user-scan.log \
    && grep -q -- '^--- PASS: TestCopperholtUserScanErrorsStayUnconfirmed ' /tmp/copperholt-user-scan.log \
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
