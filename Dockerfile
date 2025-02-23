# Build stage
FROM ayankousky/go-base:1.2025-01-12 as build

ARG GIT_BRANCH
ARG GITHUB_SHA
ARG CI

WORKDIR /srv
COPY . .

RUN \
    if [ -z "$CI" ] ; then \
    echo "runs outside of CI" && version=$(git rev-parse --abbrev-ref HEAD)-$(git log -1 --format=%h)-$(date +%Y%m%dT%H:%M:%S); \
    else version=${GIT_BRANCH}-${GITHUB_SHA:0:7}-$(date +%Y%m%dT%H:%M:%S); fi && \
    echo "version=$version"
RUN go build -o /srv/exchange-importer -ldflags "-X main.revision=${version} -s -w" cmd/importer/main.go

# Development stage
FROM ayankousky/go-base:1.2025-01-12 as dev
WORKDIR /srv

# Create non-root user and setup directories
RUN adduser -s /bin/sh -D -u 1000 app && \
    mkdir -p /srv && \
    chown -R app:app /srv /go

USER app

CMD ["air", "-c", ".air.toml"]

# Release stage
FROM alpine:3.21.3 as release

# Create non-root user and setup directories
RUN adduser -s /bin/sh -D -u 1000 app && \
    mkdir -p /srv && \
    chown -R app:app /srv

COPY --from=build --chown=app:app /srv/exchange-importer /srv/exchange-importer

USER app
WORKDIR /srv
ENTRYPOINT ["/srv/exchange-importer"]