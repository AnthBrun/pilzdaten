# Aufgabe: Merkmal-Filter-Suche + größere Bilder (Lauf 2)

Kontext: Lauf 1 lieferte Live-Suche, A-Z-Verzeichnis, Detailseiten, Bild-Endpunkt.
**Problem laut Nutzer:** Ohne die praktische Merkmal-Filter-Suche wie im Original (123pilzsuche.de / 3000pilze.de)
ist die Seite im Wald nutzlos. Bilder sind zu klein. Haupt-Use-Case: Nutzer steht im Wald mit einem Pilz,
weiß nicht, was es ist → checkt Merkmale ab → Liste schrumpft auf die Kandidaten.

## WICHTIG — Rahmenbedingungen
- **Nur in `~/projects/pilzdaten/` arbeiten. Port 18080 bleibt exakt so (docker-compose `APP_PORT`).**
  Vor dem Neustarten: `sg docker -c 'docker compose up -d --build'` aus dem Projektordner (Docker nur via `sg docker`).
- Stack bleibt **Go Standardlib**, SSR, kein Framework, keine DB, kein jQuery, kein Build-Step für JS.
  Vanilla-JS im `static/app.js`. Kein Enterprise-Zubau, keine Abhängigkeiten in go.mod.
- Neue Datenquellen (bereits fertig extrahiert, ro mount via docker-compose ergänzen):
  - `../pilzsuche/data/filter_tags.jsonl` — 4181 Zeilen, Schema:
    `{"slug":"...","tags":["braun","lamellen","gut"],...,"hut_durchmesser_min_cm":2,"hut_durchmesser_max_cm":6}`
    (Tags sind whitespace-getrennte Eigenschafts-Keys aus dem Original, Durchmesser in cm, 0 = unbekannt)
  - `../pilzsuche/data/facets.json` — fertige Filter-Definition: 9 Gruppen (Essbarkeit & Vorkommen, Hut,
    Lamellen / Röhren / Stachel, Stiel, Fruchtfleisch / Schnitt, Sporenpulver, Wuchsform & Struktur,
    Farbwechsel, Geruch & Geschmack), je Option `{"tag","label","count"}` + `min_diameter_cm:1, max_diameter_cm:25`.
    **Labels UND Gruppierung 1:1 aus dieser Datei verwenden — nicht selbst erfinden.**
- Kontext-Regel: nur die unten genannten Dateien lesen (+ max. 3 weitere, die du zwingend brauchst).
  Nicht die ganze Codebase re-scannen, nicht die 4000 Zeilen species.jsonl parsen/ausgeben.

## Arbeitspaket 1: Filter-Suche (Kernstück)
Die Original-Logik (aus `pilzsuche.js` der Quellseite) ist:
1. **AND-Verknüpfung**: Art hat alle angekreuzten Tags → passt; sonst raus.
2. **Durchmesser-Range-Slider** (Dual, 1–25 cm): Art passt wenn `min_durchmesser > dmin ODER max_durchmesser < dmax` → raus.
   Arten mit unbekanntem Durchmesser (0) bleiben immer sichtbar.
3. **Live-Textsuche** (case-insensitive) über Name + Synonyme zusätzlich.

Umsetzung:
- `internal/catalog`: filter_tags.jsonl beim Start laden → map[slug]{Tags,MinCM,MaxCM};
  dazu `facets.json` laden und als JSON-Endpunkt `GET /api/facets` ausgeben.
- `GET /` (Startseite wird zur Suchseite): Filter-Queryparams unterstützen, z. B.
  `/?q=boletus&f=braun,lamellen,gut&dmin=5&dmax=12`. Serverseitig filtern:
  AND über `f`-Tags, Range über `dmin`/`dmax` (wie oben), `q` case-insensitiv über
  Name/Wissenschaftlich/Synonyme. Sortierung: **`count`-relevante Relevanz ist nicht nötig —**
  einfach: Arten mit mehr Treffern zuerst, dann alphabetisch. Ergebnisliste bleibt paginiert.
- UI auf `templates/index.html`: **Filter-Panel (linke Spalte, auf Mobile ausklappbar)**
  mit den 9 Gruppen aus /api/facets als Checkbox-Akkordeons + Titel, pro Option das deutsche Label
  und die `count` als kleines Badge. Darunter **Dual-Range-Slider** für Durchmesser (zwei
  `<input type=range>` überlagert oder zwei Sliders "von/bis" — einfach halten, Vanilla-JS,
  Anzeige "5 – 12 cm" / "alle Werte") + Reset-Button. **Live-Suchfeld** bleibt.
  Unten Ergebnis-Zeile "N Arten gefunden." Die A-Z-Alphabetzeile bleibt (Klick setzt Filter zurück).
  Jede Checkbox/Slider-Änderung triggert sofort (kein Submit-Button) einen reload mit aktuellen Queryparams
  (throttle ~150 ms beim Tippen).
- **Ergebniskarten**: statt der leeren ◌-Symbole **reales erstes Bild** der Art (Lazy-Load, `loading=lazy`)
  als kleine Vorschaustil (~64×64, object-fit:cover, abgerundet), daneben Name + wissenschaftlicher Name.
  Das erste Bilddatei-Pfad je Art ist schon in species.jsonl (`bilddateien[0]`) → über `/media/`.

## Arbeitspaket 2: Bilder größer + nutzbar
Quelle: Original-Bilder sind klein (viele < 20 kB, ~100–600 px). Deshalb:
- Detailseite: `.image-grid` von 3 auf **2 Spalten** (Desktop), `.mushroom-image img`:
  `object-fit: contain` (nicht cover — Pilz komplett sichtbar), Höhe `auto` mit `max-height: 560px`,
  dunkler Hintergrund (#f4f4ee), `min-height` klein lassen. Kein unnötiges Hochskalieren:
  `max-width:100%`, aber die Grid-Zellen dürfen breit sein.
- **Lightbox**: Klick aufs Bild → volles Overlay (Vanilla-JS + CSS, kein Library), Bild 1:1
  (`max-width:95vw; max-height:92vh; object-fit:contain`), Esc/Klick schließt, Pfeiltasten/Buttons
  zum Umschalten zwischen den Bildern der Art.
- Vorschaubilder in den Ergebniskarten (Arbeitspaket 1) sollen auf Klick direkt zur Detailseite führen
  (das passiert schon, da die Karte ein Link ist) — Lightbox nur auf der Detailseite.

## Abnahme (vor dem Commit selbst testen)
1. `sg docker -c 'docker compose up -d --build'` läuft fehlerfrei, Container auf 18080.
2. `curl 'http://127.0.0.1:18080/?f=braun,lamellen,gut&dmin=5&dmax=12'` liefert deutlich weniger
   Arten als `curl 'http://127.0.0.1:18080/'` und nur solche mit braun+lamellen+gut UND Durchmesser-Überlappung.
3. `curl 'http://127.0.0.1:18080/api/facets'` → JSON mit 9 Gruppen.
4. Artenseite zeigt Bilder als contain + Lightbox (HTML prüfen).
5. Bestehende Tests (`go test ./...`) gruen; ggf. neue Tests für den Filter (AND, Range, Unbekannt-Durchmesser).
6. Git-Commit(s) mit sinnvollen Messages, `git status` sauber.

Kein Refactoring außerhalb dieser Aufgabe. Keine neuen Go-Dependencies.
