# Pilzdatenbank

Leichtgewichtige, deutschsprachige Web-App für den lokalen Crawl der 123pilzsuche.de mit Live-Suche, A–Z-Verzeichnis und Artenporträts.

## Stack

Die Anwendung ist ein einzelner Go-Webserver mit HTML-Templates und Standardbibliothek, ohne zusätzliche Runtime-Dependencies. Das hält Build und Image klein; die strukturierten JSONL-Daten werden beim Start in den Speicher geladen und benötigen keinen separaten Datenbankdienst.

## Voraussetzungen und Start

Docker mit Compose sowie die Daten unter `/home/hermes/projects/pilzsuche` müssen verfügbar sein. Compose bindet `species.jsonl`, `index_arts.jsonl` und den Bilderordner read-only vom benachbarten `pilzsuche`-Verzeichnis ein; die 900 MB Bilder landen nicht im Image.

```sh
docker compose up -d
```

Die App ist anschließend unter <http://localhost:8080> erreichbar. Der erste Start baut das Multi-Stage-Image. Für spätere Änderungen am Image:

```sh
docker compose up -d --build
docker compose logs -f app
docker compose down
```

Wenn Host-Port 8080 bereits belegt ist, lässt sich der veröffentlichte Port für diesen Lauf z. B. mit `APP_PORT=18080 docker compose up -d` überschreiben.

Die Daten werden beim Start geladen. Nach Änderungen an den JSONL-Dateien den Container neu starten:

```sh
docker compose restart app
```

## Smoke-Tests

Mit laufender App prüfen diese drei Requests Suche, Artenseite und eine echte Bilddatei:

```sh
# Such-Endpoint muss den Bitteren Krempentrichterling enthalten
curl -fsS 'http://localhost:8080/api/search?q=Krempen'

# Artenseite muss erfolgreich gerendert werden
curl -fsS 'http://localhost:8080/art/BittererKrempentrichterling' | grep -E 'Merkmale|Bilder|Bitterer Krempentrichterling'

# Bilddatei muss mit HTTP 200 und einem nichtleeren Body laden
curl -fsS -o /dev/null -w 'HTTP %{http_code}, %{size_download} Bytes\n' \
  'http://localhost:8080/media/daten/details/BittererKrempentrichterling-Dateien/image002.gif'
```

## Entwicklung ohne Docker

Go 1.23 oder neuer genügt; die App verwendet keine externen Go-Module.

```sh
go test ./...
SPECIES_FILE=/home/hermes/projects/pilzsuche/data/species.jsonl \
INDEX_FILE=/home/hermes/projects/pilzsuche/data/index_arts.jsonl \
IMAGE_DIR=/home/hermes/projects/pilzsuche/raw/images \
go run ./cmd/pilzdaten
```

Weitere Konfiguration ist über `SPECIES_FILE`, `INDEX_FILE`, `IMAGE_DIR` und `ADDR` möglich. Die Such-API ist `GET /api/search?q=...`; Detailseiten liegen unter `/art/{slug}`.
