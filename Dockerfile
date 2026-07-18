# Dockerfile for snapshot-cli
FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.26@sha256:313faae491b410a35402c05d35e7518ae99103d957308e940e1ae2cfa0aac29b AS builder

ARG TARGETOS
ARG TARGETARCH
# Version metadata for bininfo ldflags; .git is excluded from the build
# context, so these must be passed in (CI does) or default to dev/unknown.
ARG VERSION=dev
ARG COMMIT=unknown
ENV CGO_ENABLED=0

WORKDIR /workspace
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
    make build CGO_ENABLED=${CGO_ENABLED} GOOS=${TARGETOS} GOARCH=${TARGETARCH} VERSION=${VERSION} COMMIT=${COMMIT}

FROM --platform=${BUILDPLATFORM:-linux/amd64} gcr.io/distroless/static:nonroot@sha256:963fa6c544fe5ce420f1f54fb88b6fb01479f054c8056d0f74cc2c6000df5240

WORKDIR /
COPY --from=builder /workspace/bin/snapshot-cli /snapshot-cli
USER 65532:65532

RUN ["/snapshot-cli", "--version"]
ENTRYPOINT ["/snapshot-cli"]
