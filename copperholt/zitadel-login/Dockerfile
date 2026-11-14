# syntax=docker/dockerfile:1.7
# Rebuild the exact deployed Zitadel Login revision with current fixed runtime
# dependencies. Remove this wrapper once the official image includes the same
# or newer patched dependency floors.
FROM node@sha256:a34ab6d7f6f4813f4d0c5469d74685604b962fbdbf5793a7bc7b4998ecb867c8 AS builder

ARG ZITADEL_REVISION=917ade35af9c51e4b57ca8e57488ae95928182ad
ENV NEXT_TELEMETRY_DISABLED="1" \
    NX_NO_CLOUD="true" \
    NX_DAEMON="false"
WORKDIR /src

RUN apk add --no-cache git=2.54.1-r0
RUN git clone --filter=blob:none https://github.com/zitadel/zitadel.git . \
    && git checkout "${ZITADEL_REVISION}" \
    && sed -i \
      's/"tar@<7.5.16": "[^"]*"/"tar@<7.5.19": "7.5.19"/' \
      pnpm-workspace.yaml \
    && sed -i \
      's/"brace-expansion@>=5.0.0 <5.0.6": "[^"]*"/'\
'"brace-expansion@>=5.0.0 <5.0.8": "5.0.8"/' \
      pnpm-workspace.yaml \
    && sed -i \
      '/^overrides:/a\  "@opentelemetry/propagator-jaeger@<2.9.0": "2.9.0"'\
'\n  "@grpc/grpc-js@<1.14.5": "1.14.5"'\
'\n  "next@<16.4.1": "16.4.1"'\
'\n  "protobufjs@<7.6.5": "7.6.5"'\
'\n  "sharp@<0.35.5": "0.35.5"' \
      pnpm-workspace.yaml
RUN --mount=type=cache,id=zitadel-login-pnpm-store,target=/root/.local/share/pnpm/store \
    corepack enable \
    && pnpm install --no-frozen-lockfile
# Repair upstream test contracts without weakening production type checking.
COPY login-test-contracts.patch login-test-contracts.before.sha256 login-test-contracts.after.sha256 /opt/copperholt-login-tests/
RUN chmod 0700 /opt/copperholt-login-tests \
    && sha256sum -c /opt/copperholt-login-tests/login-test-contracts.before.sha256 \
    && git apply --check /opt/copperholt-login-tests/login-test-contracts.patch \
    && git apply /opt/copperholt-login-tests/login-test-contracts.patch \
    && sha256sum -c /opt/copperholt-login-tests/login-test-contracts.after.sha256 \
    && rm /opt/copperholt-login-tests/login-test-contracts.patch /opt/copperholt-login-tests/login-test-contracts.before.sha256 /opt/copperholt-login-tests/login-test-contracts.after.sha256
# Every sign-in host cookie the login sets is Secure, HttpOnly and SameSite=Lax
# (QA-AUTH-STORAGE found fingerprintId and NEXT_LOCALE without Secure), and
# fingerprintId lasts 30 days instead of upstream's one year (Compliance, 8 Nov
# 2026, data minimisation). Upstream has no setting for these flags or for the
# fingerprintId lifetime, so the exact audited files are patched and the
# regression scan below pins them.
COPY login-cookie-flags.patch login-cookie-flags.before.sha256 login-cookie-flags.after.sha256 /opt/copperholt-login-cookies/
COPY copperholt-cookie-flags.test.ts /src/apps/login/src/lib/copperholt-cookie-flags.test.ts
RUN sha256sum -c /opt/copperholt-login-cookies/login-cookie-flags.before.sha256 \
    && git apply --check --whitespace=error-all /opt/copperholt-login-cookies/login-cookie-flags.patch \
    && git apply --whitespace=error-all /opt/copperholt-login-cookies/login-cookie-flags.patch \
    && sha256sum -c /opt/copperholt-login-cookies/login-cookie-flags.after.sha256 \
    && rm -r /opt/copperholt-login-cookies
# AGPL-3.0 section 13 source offer (LIC-FIX01, Compliance FINAL wording of 14 Nov
# 2026): every page the login serves (the (login) layout, its loading fallback and
# the global error page) shows the footer line with the "Source code" link to the
# published fork. The regression test below pins the wording and that every root
# layout renders it.
COPY login-source-offer.patch login-source-offer.before.sha256 login-source-offer.after.sha256 /opt/copperholt-login-source-offer/
COPY copperholt-source-offer.test.tsx /src/apps/login/src/components/copperholt-source-offer.test.tsx
RUN sha256sum -c /opt/copperholt-login-source-offer/login-source-offer.before.sha256 \
    && git apply --check --whitespace=error-all /opt/copperholt-login-source-offer/login-source-offer.patch \
    && git apply --whitespace=error-all /opt/copperholt-login-source-offer/login-source-offer.patch \
    && sha256sum -c /opt/copperholt-login-source-offer/login-source-offer.after.sha256 \
    && rm -r /opt/copperholt-login-source-offer
RUN pnpm nx run --nxBail @zitadel/login:build
COPY summarize-login-tests.cjs /opt/copperholt-login-tests/summarize-login-tests.cjs
RUN --network=none pnpm --dir apps/login exec vitest run --maxWorkers=2 \
      src/app/ready/route.test.ts \
      src/components/change-password-form.test.tsx \
      src/components/copperholt-source-offer.test.tsx \
      src/components/set-password-form.test.tsx \
      src/components/set-register-password-form.test.tsx \
      src/lib/api.test.ts \
      src/lib/cookies.test.ts \
      src/lib/copperholt-cookie-flags.test.ts \
      src/lib/fingerprint.test.ts \
      src/lib/oidc.test.ts \
      src/lib/server/loginname.test.ts \
      src/lib/server/password.test.ts \
      src/lib/server/verify.test.ts \
      src/lib/session.test.ts \
      src/lib/verify-helper.test.ts \
      --reporter=json --outputFile=/opt/copperholt-login-tests/login-tests.json > /opt/copperholt-login-tests/login-tests.log 2>&1; \
    test_status=$?; \
    node /opt/copperholt-login-tests/summarize-login-tests.cjs && test "$test_status" -eq 0 \
    && rm /opt/copperholt-login-tests/login-tests.json /opt/copperholt-login-tests/login-tests.log /opt/copperholt-login-tests/summarize-login-tests.cjs

FROM node@sha256:a34ab6d7f6f4813f4d0c5469d74685604b962fbdbf5793a7bc7b4998ecb867c8
WORKDIR /app
# The base's only pending security upgrade is zlib (named, not a blanket apk upgrade).
RUN apk add --no-cache --upgrade zlib=1.4-r0 \
    && addgroup --system --gid 1001 nodejs \
    && adduser --system --uid 1001 --ingroup nodejs nextjs \
    && mkdir -p /.env-file \
    && touch /.env-file/.env \
    && chown -R nextjs:nodejs /.env-file \
    && rm -rf /usr/local/lib/node_modules/npm /usr/local/lib/node_modules/corepack \
    && rm -f /usr/local/bin/npm /usr/local/bin/npx /usr/local/bin/corepack \
             /usr/local/bin/pnpm /usr/local/bin/pnpx
COPY --from=builder --chown=nextjs:nodejs /src/apps/login/.next/standalone ./
USER nextjs
ENV HOSTNAME="::" \
    PORT="3000" \
    NODE_ENV="production" \
    NEXT_TELEMETRY_DISABLED="1" \
    NODE_OPTIONS="--use-openssl-ca --require /app/load-ssl-cert-dir.cjs" \
    SSL_CERT_FILE="/etc/ssl/certs/ca-certificates.crt" \
    ZITADEL_TLS_ENABLED="false" \
    OTEL_SERVICE_NAME="zitadel-login" \
    OTEL_EXPORTER_OTLP_PROTOCOL="http/protobuf"
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ["/usr/local/bin/node", "/app/healthcheck.mjs", "/ui/v2/login/ready"]
ENTRYPOINT ["/app/entrypoint.sh", "node", "apps/login/server.js"]
