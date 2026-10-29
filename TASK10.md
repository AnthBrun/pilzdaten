# TASK10: Kombinationen ohne Daten ausgrauen (available-tags)

Ziel: Chips und Deck-Optionen, die mit der aktuellen Auswahl keine Pilze ergeben, werden **visuell ausgraviert** (ausgegraut), aber bleiben anklickbar. Ein einzelner Endpoint, kein N-Requests.

## 1) Backend — `internal/catalog/query.go` + `internal/web/server.go`

**`catalog/query.go`**: Neue Methode neben `Filter`:
```go
// AvailableTags returns, for every tag that exists in the catalog, how many
// species match the given selection plus that tag. Tags with count 0 are
// combinations with no data. Query and diameter use the same rules as Filter.
func (c *Catalog) AvailableTags(query string, tags []string, minCM, maxCM int) map[string]int
```
Implementierung (Logik von `Filter` wiederverwenden, keine Duplikation):
- Alle bekannten Tags sammeln: Union über `c.filterData[slug].Tags` (einmal bei jedem Call ist OK, oder aus `c.all`/facets — was am einfachsten ist; es sind nur ~100 Tags).
- Für jedes Kandidaten-Tag `t`: count = Anzahl Arten, die `query` (falls gesetzt) + `minCM/maxCM`-Intervall + **alle** `tags` + `t` erfüllen (gleiche Regeln wie in `Filter`: Tag-Match via `filterData`, Durchmesser-Regel identisch).
- Um O(species × tags) zu vermeiden: **ein** Durchlauf über `c.all` — für jede Art die Base-Prüfung (query, diameter) wie in `Filter`; wenn sie besteht, wird ihre Tag-Menge geprüft: für jedes `t` in ihren Tags, das alle `tags` außer sich selbst hat … einfacher und korrekt: für jede base-passende Art `s` mit Tag-Menge `P`: wenn `P ⊇ selected` gilt, dann incremented `count[t]` für jedes `t ∈ P` zusätzlich nur, wenn `t ∉ selected` UND `t ≠` … — korrekt: ein Tag `t` zählt zu "verfügbar", wenn mindestens eine Art alle selected-Tags **plus** `t` hat. Also: für jede Art mit `P ⊇ selected ∪ {t}`. Effizient: für jede base-passende Art `s` mit `P`: wenn `P` alle `selected` enthält → für jedes `t ∈ P \ selected`: `count[t]++`. Das ist exakt die gewünschte Semantik (selected-Tags selbst erscheinen nicht in der Antwort oder mit ihrem normalen Count — irrelevant, Frontend braucht nur die nicht-selektierten).
- Return: `map[string]int` (nur Tags mit count ≥ 1). ~4200 Arten × kleine Tag-Sets → microseconds.

**`server.go`**: Route `GET /api/available-tags` (neben `/api/search`):
- Parameter: `f` (kommaseparierte Tags), `dmin`, `dmax`, `q` — exakt wie bei `/api/search` parsen (gleiche Helper wiederverwenden).
- Antwort JSON: `{"tags": ["hervorragend", "gut", ...]}` — die Tags mit count ≥ 1, alphabetisch sortiert. (Nur Liste statt Map reicht dem Frontend.)
- Cache-Control wie die anderen API-Endpoints (`withCacheControl`/No-Store — wie `/api/search` es macht, konsistent halten).

## 2) Frontend — `app.js` + `app.css`

- Im selben Update-Zyklus wie `/api/search` (gleicher Debounce/`requestSequence`-Guard, damit kein altes Request die neue Auswahl überlagert): `fetch('/api/available-tags?f=…[&dmin&dmax&q=…]')`.
- Antwort-Cache in `state.availableTags` (Set) halten.
- **Visuell ausgrauen**: Alle `[data-trait-tag]`-Chips UND alle `#option-deck [data-choice-tag]`-Karten, deren Tag **nicht** in `state.availableTags` ist (und auch nicht selbst selektiert ist — selektierte Tags behalten ihren aktiven Look), bekommen die Klasse `is-exhausted`. Bei jedem Re-Render (`renderDeck()`, Chip-Zeile) und bei jedem API-Antwort-Ankommen neu anwenden (Funktion `applyExhausted()` zentral aufrufen: einmal global für Chips, einmal beim Deck-Render).
- **Aber**: Tags ohne Daten-Grundlage (z. B. alle Chips im Ausgangszustand ohne Auswahl) → `availableTags` enthält dann alle Tags mit ≥1 Art im Gesamtdatensatz; das ist korrekt und graut nichts aus, solange Kombinationen noch Daten haben.
- **Wichtig**: Nicht ausgrauen, wenn die Anfrage noch läuft bzw. bei leerem `state.availableTags`-Set nach Fehler (Fallback: alles normal anzeigen).
- Deck-Optionen bleiben bei `is-exhausted` **anklickbar** (User soll sehen, dass es leer ist — Ergebnis-Zähler zeigt dann 0).

**`app.css`**: `.is-exhausted` — `opacity: .38; filter: grayscale(.6);` mit `transition: opacity .15s`. Für `.deck-option.is-exhausted` zusätzlich z. B. `text-decoration: line-through` auf dem Label (subtil). `:focus-visible`-Outline trotzdem behalten (A11y).

## Abnahme
1. `go build ./...` (im Docker-Build) + `node --check internal/web/static/app.js` → 0.
2. `GET /api/available-tags` (leer) → enthält u. a. `hervorragend`, `giftig`, `rot`, `lamellen`.
3. `GET /api/available-tags?f=hervorragend` → enthält `rot`? (je nach Daten) — und: ein Tag, der zusammen mit `hervorragend` keine Art hat, fehlt in der Liste. Vergleichs-Check: `GET /api/search?count=1&f=hervorragend,<tag>` liefert 0 genau dann, wenn `<tag>` nicht in der Liste ist (mindestens 5 Stichproben).
4. Browser: 2 Chips setzen, die eine seltene Kombination ergeben (z. B. über Deck: Hutfarbe "blau" + Untergrund "holz") → in allen Gruppen werden alle Optionen/Chips, die 0 ergeben, ausgegraut, andere normal. Zurück auf "braun" → Ausgrauung hebt sich (keine `is-exhausted` mehr bei den früher-grauen, die wieder Daten haben).
5. Regression: Filter `grau+lamellen+lr_weiß+stiel_weiß` → 368 (UI=API), Deck-Pfeile/Multi, URL-Roundtrip.
6. Timing: `/api/available-tags` mit 4 Tags antwortet < 50 ms (curl `-w '%{time_total}'`).

## Stil
Bestehender Code-Stil, nur stdlib, keine neuen Dateien, keine JS-Bibliothek.
