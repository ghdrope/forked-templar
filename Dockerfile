# Binary name, usually the GitHub repository name
ARG PROJECT_NAME="templar"

FROM golang:1.27 AS builder

ARG TARGETOS
ARG TARGETARCH
ARG PROJECT_NAME
# Empty for single component projects, otherwise builds cmd/<COMPONENT>
ARG COMPONENT=""
ARG VERSION="development"
ARG GIT_COMMIT="none"
ARG BUILD_DATE="unknown"

ENV CGO_ENABLED=0

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" go build -trimpath \
      -ldflags "-s -w \
        -X github.com/ghdrope/go-version.Version=${VERSION} \
        -X github.com/ghdrope/go-version.GitCommit=${GIT_COMMIT} \
        -X github.com/ghdrope/go-version.BuildDate=${BUILD_DATE}" \
      -o "/out/${PROJECT_NAME}${COMPONENT:+-${COMPONENT}}" \
      "./cmd/${COMPONENT}"


FROM debian:trixie-slim

ARG VERSION="development"
ENV VERSION=${VERSION}

# upgrade picks up Debian security fixes released after the base image was built
RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get upgrade -y \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Binary: /usr/local/bin/<PROJECT_NAME> or /usr/local/bin/<PROJECT_NAME>-<COMPONENT>
COPY --from=builder /out/ /usr/local/bin/
