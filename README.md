# Pilzdatenbank

Mobile Web-App zum Nachschlagen von Pilzen im Wald. Man tippt auf eine Pilzzeichnung, markiert, was man sieht (Hutfarbe, Lamellen, Ring, Untergrund …), und bekommt die passenden Arten. Die App funktioniert auch offline.

Grundlage ist ein lokaler Crawl von [123pilzsuche.de](https://123pilzsuche.de). Bei Einordnungen gilt im Zweifel die Quelle.

- Ein einzelner Go-Server, nur Standardbibliothek, keine Datenbank.
- Alle Daten werden beim Start in den Speicher geladen.
- Templates, CSS und JS sind in das Binary eingebettet.
- Die aufbereiteten Daten (~17 MB) liegen im Repo unter `data/`. Die Fotos (~900 MB) liegen außerhalb, ihr Ort wird über `IMAGE_DIR` angegeben.

## Schnellstart

Einmalig `.env` anlegen und `IMAGE_DIR` auf den Bilderordner setzen:

```sh
cp .env.example .env
```

```sh
docker compose up -d            # http://localhost:8080
docker compose up -d --build    # nach Code- oder Daten-Änderungen
```

Die Daten stecken im Image, deshalb ist nach Daten-Änderungen ein Neubau nötig. Die Bilder werden nur eingebunden.

Ohne Docker (Go ≥ 1.23), aus dem Repo-Ordner:

```sh
IMAGE_DIR=../pilzsuche/raw/images go run ./cmd/pilzdaten
```

Nach Änderungen an `internal/web/static` oder `internal/web/templates` muss der Server neu gebaut werden, weil die Dateien eingebettet sind.

## Ordnerstruktur der Daten

```
pilzdaten/                        dieses Repo
└── data/                         DATA_DIR (im Container /data)
    ├── species.jsonl             eine Art pro Zeile
    ├── index_arts.jsonl          Liste der echten Arten
    ├── filter_tags.jsonl         Filter-Tags und Hutdurchmesser je Art
    ├── facets.json               Filtergruppen mit Labels
    ├── best_images.jsonl         bevorzugtes Foto je Art
    └── pictogram_map.json        Tooltip-Texte der Merkmale

<beliebiger Ort>/                 IMAGE_DIR (im Container /images, read-only)
└── daten/details/
    └── <Slug>-Dateien/
        ├── image004.jpg          Fotos
        └── …
```

Alle sechs Datendateien sind Pflicht. Fehlt eine oder ist `IMAGE_DIR` nicht gesetzt, bricht der Server beim Start mit einer Meldung ab.

| Datei | Inhalt | Wofür |
|---|---|---|
| `species.jsonl` | `slug`, `name_trivial` (kommagetrennt, erster Name = Anzeigename), `name_wissenschaftlich`, `synonyme`, `merkmale` (Freitext-Felder wie `hut`, `vorkommen`, `priorität`), `bilddateien`, `verwechslungen` (Slugs) | Artseite, Suche, Saison- und Waldfilter (aus `vorkommen`), Sortierung |
| `index_arts.jsonl` | `slug`, `name`, `wissenschaftlich`, `gattung` | Nur Slugs aus dieser Liste erscheinen in der App. So bleiben Meta-Seiten der Quelle draußen. |
| `filter_tags.jsonl` | `slug`, `tags` (z. B. `rot`, `lr_gelb`, `ring`, `holz`), `hut_durchmesser_min_cm`, `hut_durchmesser_max_cm` | Alle Merkmalsfilter |
| `facets.json` | `min_diameter_cm`, `max_diameter_cm`, `groups[]` mit `group`, `title`, `options[]` (`tag`, `label`, `count`) | Filterleisten und Labels |
| `best_images.jsonl` | `slug`, `image` (Pfad wie in `bilddateien`) | Kartenbild und erstes Foto der Artseite |
| `pictogram_map.json` | `tag → { title, img, img_active }` | Nur `title`, als Tooltip der Chips |

### Bilder

Die Fotos stammen aus dem Crawl (`pilzsuche/raw/images`) und können überall liegen, z. B. auf einer externen Platte oder einem NAS. `IMAGE_DIR` zeigt auf den Ordner, der `daten/` enthält. Die Bildpfade in den Daten sind relativ dazu, zum Beispiel:

```
bilddateien:  daten/details/Fliegenpilz-Dateien/image004.jpg
Datei:        $IMAGE_DIR/daten/details/Fliegenpilz-Dateien/image004.jpg
URL:          /media/daten/details/Fliegenpilz-Dateien/image004.jpg
```

- Nur `.jpg` zählt als Foto. Die `.gif`-Einträge in `bilddateien` sind Marker der Quelle und werden ignoriert.
- Fehlende Bilder führen nicht zu Fehlern, sie fehlen dann nur in der Anzeige.

### Daten aktualisieren

Nach einem neuen Crawl die sechs Dateien aus `pilzsuche/data/` nach `data/` kopieren, committen und neu bauen:

```sh
cp ../pilzsuche/data/{species.jsonl,index_arts.jsonl,filter_tags.jsonl,facets.json,best_images.jsonl,pictogram_map.json} data/
docker compose up -d --build
```

## Aufbau des Codes

```
cmd/pilzdaten/main.go        Start: Umgebungsvariablen lesen, Katalog laden, Server starten
data/                        Aufbereitete Daten aus dem Crawl (siehe oben)
internal/catalog/            Daten laden und abfragen (ohne HTTP)
  load.go                    Dateien einlesen, Essbarkeit auflösen, Labels kürzen
  field.go                   Abgeleitete Tags: hutpilz, monat_1–12, wald_*/baum_*; Relevanz-Sortierung
  query.go                   Filter, Namenssuche, verfügbare Tags, A–Z
internal/web/                HTTP
  server.go                  Routen, gemeinsame Query-Parameter
  home.go                    Startseite (vorgerenderte Treffer, funktioniert ohne JS)
  detail.go                  Artseite
  api.go                     JSON-API
  offline.go                 Offline-Paket, Service Worker, Manifest, Offline-Hülle
  templatefuncs.go           Helfer für Templates (Bild-URLs, Namen)
  templates/
    index.html               Startseite
    mushroom.html            Pilzzeichnung (SVG) und Gruppensymbole
    detail.html              Artseite
    offline.html             Hülle für Artseiten ohne Netz
  static/
    app.css                  Alle Styles, gegliedert in "=== Abschnitte"
    sw.js                    Service Worker (Caching-Strategie)
    js/main.js               Einstieg: verbindet die Module
    js/config.js             Merkmals-Vokabular, Optionen je Pilzteil
    js/state.js              Suchzustand <-> URL
    js/mushroom.js           Zeichnung an Auswahl anpassen
    js/deck.js               Auswahl-Deck beim Antippen eines Pilzteils
    js/results.js            Trefferliste, Filter-Chips, Trefferleiste
    js/offline.js            Offline-Paket laden, offline suchen, Offline-Artseite
    js/detail.js             Lightbox, Zurück-Link
```

### Routen

| Route | Zweck |
|---|---|
| `GET /` | Startseite. Query: `q` (Name), `f` (Tags, kommagetrennt), `dmin`/`dmax` (cm), `page`, `letter` |
| `GET /art/{slug}` | Artseite |
| `GET /api/search` | Treffer als JSON, gleiche Parameter wie `/` |
| `GET /api/available-tags` | Tags, die mit der aktuellen Auswahl noch Treffer liefern. Die übrigen graut das Frontend aus. |
| `GET /api/facets` | Filtergruppen |
| `GET /api/offline` | Offline-Paket: alle Arten kompakt, gzip, mit ETag |
| `GET /media/…` | Fotos aus `IMAGE_DIR` |
| `GET /static/…` | CSS, JS, Icons (eingebettet) |
| `GET /sw.js`, `/manifest.webmanifest`, `/offline` | PWA |

### Tags

Ein Tag ist ein Merkmal, nach dem gefiltert wird. Gewählte Tags werden mit UND verknüpft.

- **Aus `filter_tags.jsonl`:** Die Farb-Tags tragen ein Präfix für den Pilzteil: `rot` (Hut), `lr_rot` (Lamellen/Röhren), `stiel_rot`, `fl_rot` (Fleisch), `sp_*` (Sporenpulver).
- **Abgeleitet in `internal/catalog/field.go`:**
  - `hutpilz`: hat Lamellen, Röhren, Stacheln oder Leisten
  - `monat_1` … `monat_12`: aus `merkmale.vorkommen`; ohne Angabe gilt jeder Monat
  - `wald_laub`, `wald_nadel`, `baum_*`: aus `merkmale.vorkommen`
- **Essbarkeit:** Pro Art bleibt genau ein Tag übrig, und zwar der vorsichtigste. Widersprüchliche Angaben der Quelle (z. B. „hervorragend“ und „giftig“) werden zu „giftig“.

### Sortierung

Ohne Namenssuche werden die Treffer in dieser Reihenfolge sortiert:

1. Priorität (`merkmale.priorität`, 1 = wichtig)
2. Häufigkeit (aus Wörtern wie „häufig“ oder „selten“ im Text)
3. Fotoanzahl, in Zehnerstufen bis 40
4. Name

Mit Namenssuche kommen zuerst exakte Treffer, dann solche, die mit dem Suchbegriff beginnen, dann solche, die ihn enthalten. Das Frontend wiederholt diese Logik für den Offline-Betrieb in `js/offline.js`. Wer `query.go` ändert, muss `js/offline.js` mitziehen.

### Offline

1. „Offline speichern“ lädt `/api/offline` (~3 MB gzip) und die Fotos der Priorität-1-Arten (~20 MB) in den Browser-Cache.
2. Ohne Netz sucht `js/offline.js` im Paket. Nicht besuchte Artseiten liefert der Service Worker als `/offline` aus, und `js/offline.js` füllt sie aus dem Paket.
3. Nach Daten-Updates muss man auf dem Handy erneut auf „Offline bereit“ tippen, um zu aktualisieren.

## Häufige Änderungen

- **Neue Option im Pilz-Deck:** Eintrag in `partChoices` in `js/config.js`. Soll sich die Zeichnung ändern, kommt ein Element in `templates/mushroom.html` dazu und ein Eintrag in `overlays` in `js/mushroom.js`.
- **Neue Pilzgruppe oder Silhouette:**
  1. In `templates/mushroom.html` eine `<g data-body="<tag>">` nach dem Muster der vorhandenen Silhouetten anlegen.
  2. Den Tag in `groupTags` in `js/config.js` eintragen.
  3. Die Schnittform in `fleshShapes` in `js/mushroom.js` ergänzen.
  4. Die Sichtbarkeitsregel in `app.css` im Abschnitt „Silhouetten“ ergänzen.
  5. In `addFieldFacets` in `field.go` eintragen.
- **Label eines Filters:** `sourceLabels` in `internal/catalog/load.go`.
- **Neue JS-Datei:** In `SHELL_URLS` in `static/sw.js` eintragen und dort `SHELL_CACHE` hochzählen, sonst fehlt sie offline.
- **Artseite:** `templates/detail.html` und `renderSpecies` in `js/offline.js` gemeinsam ändern.

## Tests

```sh
go test ./...
go vet ./...
```

Smoke-Test gegen eine laufende Instanz:

```sh
curl -fsS 'http://localhost:8080/api/search?q=Krempen'
curl -fsS 'http://localhost:8080/art/BittererKrempentrichterling' | grep -E 'Merkmale|Bitterer Krempentrichterling'
curl -fsS -o /dev/null -w 'HTTP %{http_code}\n' \
  'http://localhost:8080/media/daten/details/BittererKrempentrichterling-Dateien/image010.jpg'
```

## Konfiguration

| Variable | Standard | |
|---|---|---|
| `IMAGE_DIR` | – (Pflicht) | Ordner mit den Fotos (enthält `daten/details/…`). In Docker Compose: Host-Pfad aus `.env` |
| `DATA_DIR` | `data` (im Image `/data`) | Ordner mit den sechs Datendateien |
| `ADDR` | `:8080` | Listen-Adresse |
| `APP_PORT` | `8080` | Nur Docker Compose: Host-Port |
