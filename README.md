# Pilzdatenbank

Leichtgewichtige Web-App für den lokalen Crawl der 123pilzsuche.de.

## Stack

Die Anwendung ist ein einzelner Go-Webserver mit HTML-Templates und Standardbibliothek, ohne zusätzliche Runtime-Dependencies. Das hält Build und Image klein; die strukturierten JSONL-Daten werden beim Start in den Speicher geladen und benötigen keinen separaten Datenbankdienst.

## Entwicklung

```sh
go test ./...
SPECIES_FILE=/home/hermes/projects/pilzsuche/data/species.jsonl \
INDEX_FILE=/home/hermes/projects/pilzsuche/data/index_arts.jsonl \
go run ./cmd/pilzdaten
```
