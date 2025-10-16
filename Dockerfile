# Dockerfile for snapshot-cli
FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.25 AS builder

ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=0

WORKDIR /workspace
COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
    go build -o snapshot-cli ./cmd/main.go

FROM --platform=${BUILDPLATFORM:-linux/amd64} gcr.io/distroless/static:nonroot

WORKDIR /
COPY --from=builder /workspace/bin/* .
USER 65532:65532

RUN ["/snapshot-cli", "--version"]
ENTRYPOINT ["/snapshot-cli"]
