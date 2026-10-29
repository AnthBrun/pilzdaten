FROM golang:1.26-alpine AS builder

WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/pilzdaten ./cmd/pilzdaten

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/pilzdaten /pilzdaten
# Aufbereitete Daten kommen mit ins Image, die Fotos werden als Volume eingebunden.
COPY data /data
ENV DATA_DIR=/data IMAGE_DIR=/images ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["/pilzdaten"]
