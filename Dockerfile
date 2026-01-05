# syntax=docker/dockerfile:experimental
# ---
FROM golang:1.25 AS build

ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED=0

WORKDIR /workspace

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build admission-webhook
RUN --mount=type=cache,target=/root/.cache/go-build,sharing=private \
  go build -o admission-webhook . && \
  ls -lh /workspace/admission-webhook

# ---
FROM gcr.io/distroless/static AS run
WORKDIR /
COPY --from=build /workspace/admission-webhook /admission-webhook
USER nonroot:nonroot

ENTRYPOINT ["/admission-webhook"]