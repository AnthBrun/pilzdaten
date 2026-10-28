FROM golang:1.26-alpine AS builder

WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/pilzdaten ./cmd/pilzdaten

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/pilzdaten /pilzdaten
EXPOSE 8080
ENTRYPOINT ["/pilzdaten"]
