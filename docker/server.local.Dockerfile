# Locally-buildable variant of server.Dockerfile.
#
# IDENTICAL to server.Dockerfile except for the two base images. The official
# file builds FROM dhi.io/* (Docker Hardened Images), which require a paid
# Docker entitlement; an anonymous pull returns 401 Unauthorized. Use this file
# when you need to build the server image without that entitlement.
#
# NOT a drop-in replacement for production: DHI bases are distroless-style
# (no shell, no package manager) and carry their own CVE remediation stream.
# alpine:3.23 has a shell and apk, so the resulting image has a larger attack
# surface. For published production images, build server.Dockerfile in CI where
# the Docker Hub login provides the entitlement.
#
#   docker build -f docker/server.local.Dockerfile -t patchmon-server:local .
#
# Regenerate after editing server.Dockerfile so the two stay in sync.

# Development stage - run with go run, source can be volume-mounted for live reload
FROM golang:1.26-alpine AS development

RUN apk add --no-cache git ca-certificates tzdata curl nodejs npm

WORKDIR /app

# Copy agent binaries (run `make build-all-for-docker` in agent-source-code if agents-prebuilt is missing).
# The scripts are embedded in the server binary; there is no top-level agents/
# directory to copy since upstream removed the duplicate copies.
COPY --chmod=755 agents-prebuilt/patchmon-agent-* ./agents/

# Build frontend for embed
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm install --ignore-scripts --legacy-peer-deps
COPY frontend/ ./
RUN npm run build

WORKDIR /app/server
COPY server-source-code/ ./
RUN mkdir -p cmd/server/static/frontend && cp -r /app/frontend/dist cmd/server/static/frontend/

RUN go mod download

EXPOSE 3000

HEALTHCHECK --interval=10s --timeout=5s --start-period=60s --retries=5 \
  CMD curl -f http://localhost:${PORT:-3000}/health || exit 1

# Default: run server. Override CMD or use volume mount for live reload
ENV AGENTS_DIR=/app/agents
ENV PORT=3000
CMD ["go", "run", "./cmd/server"]

# Frontend builder stage for production.
#
# Pinned to $BUILDPLATFORM. The output is static JS/CSS/HTML (see the COPY of
# /app/frontend/dist below), which is architecture-independent, so there is
# nothing to gain from building it once per target platform — and a great deal
# to lose. Without this pin BuildKit instantiates this stage for every
# --platform in the build, so the linux/arm64 variant runs Node and npm under
# QEMU user-mode emulation on an amd64 runner. That crashed `npm ci` with
# "qemu: uncaught target signal 4 (Illegal instruction)" and exit code 132,
# while the native amd64 variant of the same step succeeded in seconds.
#
# Pinning also roughly halves this stage's wall-clock cost, since the install
# and the Vite build no longer run twice. The consumer of dist is the `builder`
# stage, which is itself $BUILDPLATFORM-pinned, so nothing downstream needs a
# target-architecture copy of these files.
FROM --platform=$BUILDPLATFORM node:22-slim AS frontend-builder

WORKDIR /app

# Install from the committed lockfile so the image resolves exactly the versions
# the host and CI resolve. The previous "rm package-lock.json && npm install
# --force" left the image free to pick any version matching the semver ranges,
# which silently broke the build when react-icons 5.7.0 dropped SiSlack.
#
# frontend is an npm workspace member and the lockfile lives at the repo root,
# so both manifests must be present before npm ci can run.
COPY package.json package-lock.json ./
COPY frontend/package.json ./frontend/

RUN npm ci --workspace=patchmon-frontend --include=dev --ignore-scripts --no-audit \
    && npm cache clean --force

COPY frontend/ ./frontend/

WORKDIR /app/frontend

RUN npm run build

# Build stage - server (runs on amd64, cross-compiles for target platform)
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy server source
COPY server-source-code/ ./server/
# Copy built frontend into embed directory
COPY --from=frontend-builder /app/frontend/dist ./server/cmd/server/static/frontend/dist

WORKDIR /app/server

ARG TARGETOS
ARG TARGETARCH
RUN go mod download && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -buildvcs=false -ldflags="-s -w" -o /app/patchmon-server ./cmd/server

# SSG content stage — download ComplianceAsCode datastream files at build time.
# Pass --build-arg SSG_VERSION=0.1.80 to pin a specific version; otherwise
# the latest GitHub release is resolved automatically.
#
# Pinned to $BUILDPLATFORM for the same reason as frontend-builder: the payload
# is ssg-*-ds.xml datastream files, which are architecture-independent. Left
# unpinned, this stage downloaded and unpacked the same ~30s archive once per
# target platform, and did the unpacking under QEMU for the non-native one.
FROM --platform=$BUILDPLATFORM alpine:3.23 AS ssg-content
ARG SSG_VERSION=""
# Use shell variable VER to avoid Docker ARG substitution in the wget URL.
# Docker substitutes ${SSG_VERSION} at parse time; when empty, the URL would be
# .../v/scap-security-guide-.zip. VER is set once from ARG, then expanded by the shell.
RUN apk add --no-cache wget unzip jq \
    && VER="${SSG_VERSION}" \
    && if [ -z "${VER}" ]; then \
         VER=$(wget -qO- https://api.github.com/repos/ComplianceAsCode/content/releases/latest | jq -r '.tag_name' | sed 's/^v//'); \
         echo "Resolved latest SSG version from GitHub API: ${VER}"; \
       else \
         echo "Using pinned SSG version: ${VER}"; \
       fi \
    && if [ -z "${VER}" ] || [ "${VER}" = "null" ]; then \
         echo "ERROR: Could not resolve SSG version (GitHub API may be rate-limited). Pass --build-arg SSG_VERSION=x.y.z to pin." >&2; exit 1; \
       fi \
    && wget -q "https://github.com/ComplianceAsCode/content/releases/download/v${VER}/scap-security-guide-${VER}.zip" -O /tmp/ssg.zip \
    && mkdir -p /tmp/ssg-extract /ssg-content \
    && unzip -q /tmp/ssg.zip -d /tmp/ssg-extract \
    && find /tmp/ssg-extract -name 'ssg-*-ds.xml' -exec cp {} /ssg-content/ \; \
    && echo "${VER}" > /ssg-content/.ssg-version \
    && rm -rf /tmp/ssg.zip /tmp/ssg-extract

# Production stage — stock Alpine runtime standing in for dhi.io/alpine-base.
FROM alpine:3.23

# The hardened base ships ca-certificates and tzdata; stock alpine does not.
# Both are required: TLS verification for outbound calls (SMTP, OIDC, GitHub
# release checks) and the timezone database for scheduled reports.
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binary (migrations and frontend are embedded in the binary)
COPY --from=builder /app/patchmon-server ./

# Copy SSG content (SCAP datastream files for compliance scanning)
COPY --from=ssg-content /ssg-content ./ssg-content/

# Copy agent binaries to /app/agents (in-image, read-only; no volume).
COPY --chmod=755 agents-prebuilt/patchmon-agent-* ./agents/

# Entrypoint starts server (no volume copy; agents served from image)
COPY --chmod=755 docker/backend.docker-entrypoint.sh ./entrypoint.sh

ENV PORT=3000
ENV AGENTS_DIR=/app/agents
ENV SSG_CONTENT_DIR=/app/ssg-content
# Cap Go heap to reduce RAM (override at runtime if needed, e.g. GOMEMLIMIT=128MiB)
ENV GOMEMLIMIT=256MiB

EXPOSE 3000

HEALTHCHECK --interval=10s --timeout=5s --start-period=30s --retries=5 \
  CMD wget -q -O /dev/null http://localhost:${PORT:-3000}/health || exit 1

ENTRYPOINT ["./entrypoint.sh"]
