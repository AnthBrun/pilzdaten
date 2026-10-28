# TASK3: Pilzdatenbank als Such-Hilfe neu ausrichten

Die aktuelle Version ist ein Lexikon mit schmucklosem Filter-Panel. Sie soll eine
**praktische Bestimmungs-Hilfe** werden: Man steht im Wald, kennt den Pilz nicht,
markiert die Merkmale, die man sieht, und bekommt wenige plausible Kandidaten.

**WICHTIG — Port 18080 bleibt unverändert.** Hinter 18080 läuft ein Reverse-Proxy,
der den Stand mitverfolgt. `APP_PORT=18080` setzen beim `docker compose up`, nie ändern.

## 1. Piktogramm-Filter (wie im Original)

Das Original (123pilzsuche.de) zeigt pro Merkmal ein kleines **Piktogramm-Bild** statt
einer langweiligen Checkbox + Textzeile. Das sieht der User im Wald auf dem Handy sofort.

**Alle 209 Piktogramm-Dateien sind bereits vorhanden** (aus dem Original gecrawlt):
`internal/web/static/piktogramme/*.jpg` — sie liegen schon im Embed-Pfad
(`static/piktogramme/...` wird via `//go:embed static/*` eingebunden und über
`/static/piktogramme/<name>.jpg` serviert). Pro Merkmal gibt es zwei Varianten:
`<tag>.jpg` (inaktiv) und `<tag>_ja.jpg` (aktiv/geprüft).

**Das Mapping Tag → Piktogramm liegt in `/data/pictogram_map.json`** (im Container gemountet,
siehe docker-compose.yml):
```json
{ "braun": {"title":"Hut ist braun.","img":"braun","img_active":"braun_ja"}, ... }
```
Alle 109 Facetten-Optionen aus `facets.json` sind abgedeckt. Die `title`-Felder sind die
Tooltips (HTML-Entities wie `&szlig;` korrekt rendern).

Aufgabe: Das Filter-Panel so umbauen, dass jede Option ein **Piktogramm zeigt** (ca. 36–48px),
deutlich größerer Klickbereich, Tooltip mit der `title`-Beschreibung, aktivem Zustand
farbig hervorgehoben (Piktogramm-Datei mit `_ja`-Suffix verwenden oder CSS-Highlight).
Text-Label optional klein daneben/unter. Gruppen bleiben (9 Gruppen aus facets.json).
Layout: kompakt, auf dem Handy nutzbar — z. B. Grid von Piktogrammen pro Gruppe.

## 2. Filter muss IMMER sichtbar und sofort reaktiv sein

Verifizierter Ist-Zustand: Der Filter funktioniert serverseitig (`?f=braun` → 2875,
Kombinationen + dmin/dmax funktionieren), ABER das Panel ist unter dem Hero + A-Z-Leiste
versteckt — wer oben auf der Seite ist, sieht das Ergebnis-Label nicht und denkt,
der Filter reagiert nicht.

Anforderungen:
- Filter-Panel und Ergebnis-Zeile müssen **zusammen im Viewport** sein (Sticky-Toolbar oder
  zweispaltiges Layout: Filter links sticky, Ergebnisse rechts — wie im Original).
- Ergebnis-Label + Trefferzahl **sticky** über der Ergebnisliste, ändert sich sofort sichtbar.
- Aktive Filter als **baldige Chips** anzeigen ("Hut braun ✕"), ein Klick entfernt sie.
- Mobile: Filter-Panel als einklappbare Leiste oben, Ergebnisse darunter.

## 3. Essbarkeit sofort sichtbar

Im Wald ist die wichtigste Frage: **Essbar, giftig oder tödlich?** — sichtbar, nicht
vergraben in der Detailseite.

Die Daten liegen in `filter_tags.jsonl` (Tags: `hervorragend`, `gut`, `schlecht`, `giftig`,
`tot`). Aufgabe:
- **Jede Ergebniskarte** bekommt ein essbarkeits-farbiges Badge/Band:
  grün = hervorragend/gut essbar, grau/gelb = essbar aber ungenießbar/schlecht,
  orange = giftig, rot = tödlich. (Kodierung ist frei, aber konsistent und mit Legende.)
- Die **Essbarkeits-Gruppe im Filter prominent nach oben** (sie ist die häufigste Wald-Frage:
  "welche essbaren braunen Pilze gibt es mit Lamellen?").
- Detailseite: Essbarkeit als großes, klares Banner am Seitenanfang (mit Warnhinweis bei
  giftig/tot: "Nur als Nachschlagehilfe — Pilze niemals ungeprüft essen!").
- Arten ohne Essbarkeits-Tag: neutral ("Essbarkeit nicht erfasst"), nicht fälschlich grün.

## 4. Bessere Karten-Bilder

Verifiziert: Die Bilddatenlisten beginnen pro Art oft mit **15×15-GIF-Markern**; `firstImage`
wählt jetzt das erste JPG, aber das ist nicht das beste. Die Quelle liefert 1–36 kB JPGs
(kein High-Res — das ist eine Daten-Begrenzung, nicht fixierbar durch Zoomen).

**Neue Datenquelle ist bereits bereit: `/data/best_images.jsonl`** (gemountet, Env
`BEST_IMAGES_FILE`): `{"slug":"...","image":"daten/details/.../imageNNN.jpg"}` — 4200 Arten,
das **größte JPG je Art** (Median 32 kB).
Aufgabe: Karten-Thumbnail aus `best_images.jsonl` nehmen (nicht erst-JPG).
- Karte: Bild groß, quadratisch (object-fit: cover), Name + essbarkeits-Badge deutlich.
- Detailseite: größtes Bild prominent als Hero-Bild, Galerie darunter (GIF-Marker bleiben
  gefiltert). Lightbox bleibt.
- Qualität: Da die Quellen-Bilder klein sind: schärfebewusstes Rendering (keine
  Upscaling-Verzerrungen), optionales `image-rendering`-Tuning. Größer darstehen = besser,
  aber nicht über den tatsächlichen Pixelinhalt hinaus schummeln.

## 5. Wald-Use-Case: "Ich stehe im Wald"

Der Kern: Der User tippt NICHT einen Namen, er **markiert Merkmale**. Also:
- Startseite = die Merkmalsuche VORAN, nicht "Durchsuche 4179 Arten" + A-Z-Lexikon.
  A-Z-Verzeichnis bleibt als zweiter Bereich, aber die Filter-UI ist der Hauptakteur.
- Hinweis-Texte im Filter-Panel im Wald-Duktus: "Was siehst du? Markiere alle Merkmale,
  die passen — je mehr, desto genauer."
- Wenn Filter aktiv sind, aber >50 Treffer: Vorschlag "Noch 2–3 Merkmale markieren für
  genauere Treffer" (sanft, nicht aufdringlich).
- Ergebnis-Listen-Header: aktiv gewählte Merkmale als Chips, Reset-Button prominent.

## Technische Fakten (verifiziert, nicht erneut recherchieren)

- Port: **18080** (`APP_PORT=18080` beim Compose-Start).
- Go-stdlib, SSR, keine DB. Docker-Build + `go test ./...` müssen grün bleiben.
- Daten (alle read-only gemountet, docker-compose.yml zeigt die Pfade):
  - `/data/species.jsonl` — 4200 Arten (Felder: slug, name_trivial,
    name_wissenschaftlich, synonyme, bilddateien, vermechslungen, …)
  - `/data/filter_tags.jsonl` — 4179 Arten, `{"slug","tags":[10–30 Tags],"hut_durchmesser_min_cm","hut_durchmesser_max_cm"}`
  - `/data/facets.json` — 9 Gruppen, 109 Optionen mit deutschen Labels + Counts
  - `/data/best_images.jsonl` — 4200 Arten, größtes JPG je Art
  - `/data/pictogram_map.json` — 109 Tags → Piktogramm-Dateinamen + Tooltips
- Piktogramme: `internal/web/static/piktogramme/` (embed), served unter `/static/piktogramme/<name>.jpg`
- Filter-API: `?q=text`, `?f=tag1,tag2` (AND), `?dmin=N&dmax=N` (cm), Paginierung 50/Seite
- Bilder: `/media/<Pfad>` aus `/images` (IMAGE_DIR). 15×15-GIFs = Marker, ignorieren.
- Docker-Befehl auf dieser Maschine: `sg docker -c 'APP_PORT=18080 docker compose up -d --build'`

## Abnahme (alle müssen greifen)

1. `curl 'http://localhost:18080/?f=braun,lamellen,gut&dmin=5&dmax=12'` → plausible
   Trefferzahl (aktuell 181), keine 4179.
2. Filter-Panel mit Piktogrammen sichtbar; Tooltip mit deutscher Beschreibung;
   aktiver Zustand farblich erkennbar.
3. Essbarkeits-Badge auf jeder Karte (farbig, konsistent); Detailseite mit
   Essbarkeits-Banner; "Nicht erfasst" neutral.
4. Karten-Thumbnail aus best_images.jsonl (größenvergleichen mit altem first-JPG).
5. Filter-Panel + Ergebnis-Label zusammen im Viewport (sticky); aktive Filter als
   Chips mit Einzel-Entfernen.
6. Mobile Viewport (390px): Filter-UI gut bedienbar, keine horizontale Scrollbar.
7. `go test ./...` grün, Docker-Build ohne Warnungen, Container auf 18080 up.
8. Lightbox funktioniert weiter.
