FROM docker.io/golang:1.26-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY gateway ./gateway
COPY gen ./gen
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/ ./cmd/server ./cmd/gateway

FROM gcr.io/distroless/static-debian12:nonroot AS server

WORKDIR /app

COPY --from=build /out/server ./server
COPY assets ./assets

ENV GRPC_PORT=50051

EXPOSE 50051

ENTRYPOINT ["/app/server"]

FROM gcr.io/distroless/static-debian12:nonroot AS gateway

WORKDIR /app

COPY --from=build /out/gateway ./gateway

ENV HTTP_PORT=8080 \
    GIN_MODE=release

EXPOSE 8080

ENTRYPOINT ["/app/gateway"]
