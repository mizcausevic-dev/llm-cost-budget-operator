# Build the manager binary
FROM golang:1.26 AS build
WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY api/ api/
COPY internal/ internal/
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o manager ./cmd

# Minimal runtime
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=build /workspace/manager .
USER 65532:65532
ENTRYPOINT ["/manager"]
