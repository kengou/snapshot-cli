# Dockerfile for snapshot-cli
FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.26@sha256:b900de91b15b2e2953d930ece1d0ecff0a1590ab2006088d20dcf0f56f1e979f AS builder

ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=0

WORKDIR /workspace
COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
    make build CGO_ENABLED=${CGO_ENABLED} GOOS=${TARGETOS} GOARCH=${TARGETARCH}

FROM --platform=${BUILDPLATFORM:-linux/amd64} gcr.io/distroless/static:nonroot@sha256:963fa6c544fe5ce420f1f54fb88b6fb01479f054c8056d0f74cc2c6000df5240

WORKDIR /
COPY --from=builder /workspace/bin/* .
USER 65532:65532

RUN ["/snapshot-cli", "--version"]
ENTRYPOINT ["/snapshot-cli"]
