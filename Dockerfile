# Dockerfile for snapshot-cli
FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.26@sha256:e2ddb153f786ee6210bf8c40f7f35490b3ff7d38be70d1a0d358ba64225f6428 AS builder

ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=0

WORKDIR /workspace
COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
    make build CGO_ENABLED=${CGO_ENABLED} GOOS=${TARGETOS} GOARCH=${TARGETARCH}

FROM --platform=${BUILDPLATFORM:-linux/amd64} gcr.io/distroless/static:nonroot@sha256:e3f945647ffb95b5839c07038d64f9811adf17308b9121d8a2b87b6a22a80a39

WORKDIR /
COPY --from=builder /workspace/bin/* .
USER 65532:65532

RUN ["/snapshot-cli", "--version"]
ENTRYPOINT ["/snapshot-cli"]
