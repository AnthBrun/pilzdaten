FROM golang:1.26-alpine AS builder

WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/pilzdaten ./cmd/pilzdaten

# Bildmodell für die Foto-Vorschläge laut model.lock (Release in pilz-ki),
# Prüfsummen werden geprüft. Ohne Modell: docker compose build --build-arg MODEL=none
FROM golang:1.26-alpine AS model
ARG MODEL=fetch
WORKDIR /src
COPY model.lock ./
COPY scripts/model.sh ./scripts/
RUN if [ "$MODEL" = none ]; then mkdir -p /out/model; else sh scripts/model.sh fetch /out/model; fi

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/pilzdaten /pilzdaten
# Aufbereitete Daten kommen mit ins Image, die Fotos werden als Volume eingebunden.
COPY data /data
COPY --from=model /out/model /model
ENV DATA_DIR=/data IMAGE_DIR=/images MODEL_DIR=/model ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["/pilzdaten"]
