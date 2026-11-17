# Pilzdatenbank

Mobile Web-App zum Nachschlagen von Pilzen im Wald. Man tippt auf eine Pilzzeichnung, markiert, was man sieht (Hutfarbe, Lamellen, Ring, Untergrund …), und bekommt die passenden Arten. Die App funktioniert auch offline.

Grundlage ist ein lokaler Crawl von [123pilzsuche.de](https://123pilzsuche.de). Bei Einordnungen gilt im Zweifel die Quelle.

- Ein einzelner Go-Server, nur Standardbibliothek, keine Datenbank.
- Alle Daten werden beim Start in den Speicher geladen.
- Templates, CSS und JS sind in das Binary eingebettet.
- Die aufbereiteten Daten (~17 MB) liegen im Repo unter `data/`. Die Fotos (~900 MB Vorschauen, ~20 GB große Originale) liegen außerhalb, ihr Ort wird über `IMAGE_DIR` angegeben.

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
    ├── pictogram_map.json        Tooltip-Texte der Merkmale
    ├── originals.json            optional: Vorschaubild -> großes Original (Lightbox)
    └── i18n/                     optional: Übersetzungen (siehe "Übersetzungen")
        ├── <sprache>/ui.json         Oberfläche, Filter-Labels
        ├── <sprache>/content.json    Namen, Gattungen, Merkmale
        └── glossary/<sprache>.tsv    optional: feste Fachbegriffe für DeepL

<beliebiger Ort>/                 IMAGE_DIR (im Container /images, read-only)
└── daten/details/
    ├── <Slug>-Dateien/
    │   ├── image004.jpg          Fotos (Vorschauen, ~400 px)
    │   └── …
    └── <Name>.jpg                große Originale (bis 1200 px, aus originals.json)
```

Die sechs Datendateien außer `originals.json` sind Pflicht. Fehlt eine oder ist `IMAGE_DIR` nicht gesetzt, bricht der Server beim Start mit einer Meldung ab.

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
- Die Quelle bindet nur verkleinerte Vorschauen ein (~400 px, viele Galeriebilder nur ~150 px). Die großen Originale lädt `python3 crawler.py originals` im Crawl-Ordner; die Zuordnung steht in `originals.json`. Die Lightbox zeigt zuerst die Vorschau und tauscht sie gegen das Original unter `/original/…`, sobald es geladen ist. Ohne `originals.json` oder offline bleibt es bei der Vorschau. Originale landen bewusst nicht im Offline-Cache.

### Daten aktualisieren

Nach einem neuen Crawl die Dateien aus `pilzsuche/data/` nach `data/` kopieren, committen und neu bauen:

```sh
cp ../pilzsuche/data/{species.jsonl,index_arts.jsonl,filter_tags.jsonl,facets.json,best_images.jsonl,pictogram_map.json,originals.json} data/
docker compose up -d --build
```

## Daten-Bundle für die native App

Die Flutter-App (Repo `pilzkorb`) soll die PWA nach und nach ersetzen und
hängt nicht von den Seiten dieses Servers ab. Beim ersten Öffnen lädt sie ein
Daten-Bundle (ca. 30 MB): Artdaten mit allen Fotos und Originalen plus
Bildmodell. Fotos holt sie vorerst von `/media` und `/original`.

- **Bauen:** Das Docker-Image baut das Bundle selbst (Stage `appbundle`) aus
  `data/` und dem Modell laut `model.lock`. Lokal:
  `go run ./cmd/apppackage -data data -model data/model -bundle data/app-bundle`
- **Ausliefern:** `GET /app/bundle/manifest.json` und die Dateien darin, nur mit
  `Authorization: Bearer <Token>`. Tokens stehen kommagetrennt in `APP_TOKENS`
  (`.env`); ohne Tokens gibt es die Route nicht. Zum Wechseln alten und neuen
  Token eine Zeit lang parallel eintragen.
- **Format:** `internal/appbundle` (Manifest mit sha256 je Datei). Die App liest
  es in `pilzkorb/lib/data/bundle_store.dart` – beide gemeinsam ändern.
- Ein Token in der App lässt sich auslesen. Er hält beiläufiges Abgreifen ab,
  ersetzt aber keine Nutzerkonten.

## Übersetzungen

Deutsch ist die Quellsprache. Jede weitere Sprache ist ein Ordner `data/i18n/<code>/` mit zwei Wörterbüchern „deutscher Text → Übersetzung“. Sie werden mit DeepL erzeugt und committet, zur Laufzeit braucht die App kein DeepL.

```sh
# Schlüssel in .env eintragen: DEEPL_API_KEY=…  (Free-Schlüssel enden auf :fx)
go run ./cmd/translate -lang en -dry-run    # zählt Texte und Zeichen, sendet nichts
go run ./cmd/translate -lang en -limit 100  # Probelauf mit 100 Texten je Datei
go run ./cmd/translate -lang en,fr          # alles Fehlende übersetzen
docker compose up -d --build
```

- **Was übersetzt wird:** Oberfläche (~400 Texte, ~11.000 Zeichen) sowie Trivialnamen, Gattungen, Merkmalsnamen und -texte (~35.700 Texte, ~4,5 Mio. Zeichen je Sprache). Wissenschaftliche Namen und Synonyme bleiben.
- **Nur Neues wird gesendet:** Schlüssel ist der deutsche Text. Ändert er sich, fehlt er im Wörterbuch und wird beim nächsten Lauf neu übersetzt; Texte, die es nicht mehr gibt, werden entfernt. Nach jedem Paket wird gespeichert, ein abgebrochener Lauf (Strg+C, Kontingent) macht beim nächsten Start weiter.
- **Platzhalter** wie `{count}` gehen als XML-Tag an DeepL und bleiben erhalten.
- **Glossar:** `data/i18n/glossary/<code>.tsv` mit `deutsch<TAB>übersetzung` je Zeile legt Fachbegriffe fest (z. B. `Lamellen	gills`).
- **Sprachwahl:** `?lang=<code>` (wird als Cookie gemerkt), sonst Cookie, sonst `Accept-Language`, sonst Deutsch. Der Umschalter oben erscheint, sobald es mehr als eine Sprache gibt. Übersetzte Seiten zeigen unten einen Hinweis auf die maschinelle Übersetzung.
- **Suche:** findet Arten über übersetzte und deutsche Namen. Sortierung und A–Z richten sich weiter nach dem deutschen Namen.
- **Pseudo-Sprache zum Prüfen:** `go run ./cmd/translate -lang qa -pseudo` setzt jeden Text in `‹…›`. Was danach ohne Klammern erscheint, läuft nicht durch die Übersetzung. Der Ordner `data/i18n/qa` ist von Git und Docker ausgeschlossen.

**Neue Texte im Code:**

- Template: `{{t "Text"}}`, mit Platzhaltern `{{tf "Seite {page}" "page" .Page}}`. Artdaten: `{{name .}}`, `{{edibility .}}`, `{{t .Species.Gattung}}`.
- JS: `t("Text")` bzw. `t("{count} Arten", { count })` aus `js/util.js`. Immer ein String-Literal übergeben, sonst findet `cmd/translate` den Text nicht.
- Go: feste Texte als Paketvariable mit `i18n.N("Text")` anmelden und beim Ausgeben mit `s.t(lang, text)` übersetzen.

## Aufbau des Codes

```
cmd/pilzdaten/main.go        Start: Umgebungsvariablen lesen, Katalog laden, Server starten
cmd/translate/               Übersetzungen mit DeepL erzeugen (siehe "Übersetzungen")
internal/i18n/               Wörterbücher laden, Sprache wählen, Platzhalter
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
  photo.go                   Bildmodell unter /model/ ausliefern, /api/species
  templatefuncs.go           Helfer für Templates (Bild-URLs, Namen, t/tf je Sprache)
  texts.go                   Sammelt alle zu übersetzenden Texte für cmd/translate
  templates/
    index.html               Startseite
    mushroom.html            Pilzzeichnung (SVG) und Gruppensymbole
    detail.html              Artseite
    offline.html             Hülle für Artseiten ohne Netz
    partials.html            Sprachumschalter, Fußzeile
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
    js/photo.js              Foto-Vorschläge: Bildmodell im Browser, Vorschlagsregel
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
| `GET /app/bundle/…` | Daten-Bundle der nativen App, nur mit Token (siehe oben) |
| `GET /api/species` | Kartendaten bestimmter Arten, Query `slugs` (kommagetrennt, max. 20) |
| `GET /model/…` | Bildmodell und onnxruntime-web aus `MODEL_DIR` (nur wenn vorhanden) |
| `GET /media/…` | Fotos (Vorschauen) aus `IMAGE_DIR` |
| `GET /original/…` | Große Originale aus `IMAGE_DIR` (nur Lightbox, nicht im Offline-Cache) |
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
4. Das Paket ist in der gewählten Sprache. Nach einem Sprachwechsel zeigt der Knopf wieder „Offline speichern“.

## Foto-Vorschläge

Mit „Foto“ fotografiert man den Fund (oder wählt bis zu sechs Fotos aus), und die App schlägt die sichtbaren Merkmale vor: Pilzgruppe, Hutfarbe und -form, Oberfläche, Unterseite und ihre Farbe, Stielfarbe, Ring/Knolle/Netz, Untergrund, büscheliger Wuchs. Die Vorschläge erscheinen als gestrichelte Chips mit Kamera und lassen sich wie jeder Filter wegtippen. Darunter stehen bis zu drei ähnlich aussehende Arten zum Vergleichen.

- **Alles läuft im Browser** (onnxruntime-web, WASM). Fotos verlassen das Gerät nie.
- **Fotos ohne Pilz** (Menschen, Tiere, Essen, Mikroskopbilder, leere Flächen) erkennt ein Gate im Modell; dann gibt es keine Vorschläge, sondern einen Hinweis.
- **Vorsichtig:** Ein Merkmal wird nur vorgeschlagen, wenn es auf Testfotos zu ≥ 99,5 % stimmte. Bei mehreren Fotos zählt der Mittelwert. Eine selbst gewählte Hutfarbe o. Ä. wird nie überschrieben.
- **Laden:** Modell und Laufzeit (~27 MB mit gzip) kommen erst beim ersten Antippen von „Foto“ und bleiben dann im Service Worker, auch offline.

### Modell einbinden

Das Modell entsteht im Repo [pilz-ki](https://github.com/AnthBrun/pilz-ki) (Training, Bewertung, Export) und wird dort als GitHub-Release `model-<version>` veröffentlicht. Dieses Repo pinnt die Version in `model.lock` (URL und SHA-256 von `meta.json` und `model.int8.onnx`, dazu Version und SHA-256 von onnxruntime-web).

```
pilz-ki:    finetune → train-gate → export → pilzki release
                                               ├─ GitHub-Release model-<version> (+ Kennzahlen)
                                               └─ PR hier: model.lock aktualisieren
pilzdaten:  PR mergen → docker compose up -d --build
            (Stage "model" im Dockerfile lädt die Dateien laut model.lock und prüft die Prüfsummen)
```

- **Neues Modell:** den PR „Bildmodell <version>“ mergen und neu bauen. Handys holen den neuen Stand von selbst, weil die URLs die Version tragen.
- **Zurück:** den Lock-Commit reverten und neu bauen.
- **Ohne Modell bauen:** `docker compose build --build-arg MODEL=none`. Der Foto-Knopf fehlt dann.
- **Lokal ohne Docker:** `scripts/model.sh fetch` legt das Modell laut Lock nach `data/model/` (dort sucht `go run` standardmäßig, `MODEL_DIR` überschreibt das). Einen noch nicht veröffentlichten Export aus pilz-ki testet man mit `scripts/model.sh local ../pilz-ki/work/export/<run>`.

Im Image liegt das Modell unter `/model` (~65 MB inklusive `.gz`-Varianten). `data/model/` ist nicht im Git.

Code: `internal/web/photo.go` (Auslieferung unter `/model/`, `/api/species`), `static/js/photo.js` (Vorverarbeitung, Inferenz, Vorschlagsregel). Die Regeln spiegeln `pilzki/predict.py` in pilz-ki; wer eine Seite ändert, ändert die andere mit. Auf 262 fremden Pilzfotos liefert der Browser dieselbe Trefferqualität wie die Python-Referenz (4,3 Merkmale pro Foto, in 98 % der Fälle ist jedes vorgeschlagene Merkmal richtig).

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
| `MODEL_DIR` | `DATA_DIR/model` (im Image `/model`) | Bildmodell für die Foto-Vorschläge (siehe oben). Fehlt es, ist die Funktion aus |
| `DATA_DIR` | `data` (im Image `/data`) | Ordner mit den sechs Datendateien |
| `ADDR` | `:8080` | Listen-Adresse |
| `APP_PORT` | `8080` | Nur Docker Compose: Host-Port |
| `APP_BUNDLE_DIR` | `data/app-bundle` (im Image `/app-bundle`) | Daten-Bundle der nativen App |
| `APP_TOKENS` | – | Tokens der App für `/app/bundle/`, kommagetrennt; leer = Route aus |
| `DEEPL_API_KEY` | – | Nur `cmd/translate`: DeepL-Schlüssel, aus der Umgebung oder `.env` |
