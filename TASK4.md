# TASK4: „Pilz-Charakter-Ersteller" — visueller SVG-Filter

## Vision
Der Filter ist kein Piktogramm-Raster mehr, sondern **ein interaktiver Pilz, den der Nutzer „erzeugt"** — wie die Charaktererstellung in einem Spiel. Der Nutzer steht im Wald, sieht einen Pilz und **setzt auf der Zeichnung die Eigenschaften, die er sieht: Farben auf den Hut, die Lamellenart, Stiel-Details.** Das Ergebnis-Set live mit (ohne Seiten-Reload). Das vorherige Piktogramm-Grid (`.facet-*`-Panel) wird **komplett entfernt**.

## Hard Constraints
- **Port 18080 FIX** — nicht ändern.
- **Keine neuen Filter-Tags!** Jede Bedienung mappt auf ein EXISTIERENDES Tag aus `data/facets.json` (siehe Mapping unten). Server-Filterlogik (`?f=`, `dmin`, `dmax`, `q`, AND) bleibt unverändert.
- Go stdlib, kein JS-Framework, kein Build-Step. Vanilla JS + SVG + CSS.
- `data/species.jsonl`, `filter_tags.jsonl`, `facets.json`, `best_images.jsonl` werden NICHT verändert.
- Detailseiten, A–Z-Verzeichnis, Lightbox, Suchfeld (`q`) bleiben unverändert.
- `go test ./...` muss grün bleiben; Docker-Build + Start auf 18080 am Ende verifizieren.

## Layout
**Desktop (2 Spalten, wie heute):**
- **Linke Spalte „Pilz-Lab" (sticky):**
  1. Großer SVG-Pilz (~380px hoch), zentrale Kachel.
  2. Darunter: **Diameterslider** (dmin/dmax, bleibt).
  3. Darunter: **Zeile „Nicht zeichenbare Merkmale"** (kleine Chips mit Text, 2 Gruppen, kompakt): *Sporenpulver* (6 Farben) und *Geruch & Geschmack* (15) + *Farbwechsel* (7). Gerüche kann man nicht zeichnen — hier sind Text-Chips korrekt.
- **Rechte Spalte:** Ergebnis-Zeile (count + Seitenanzeige), Filter-Chips (aktiv, einzeln entfernbar, „Zurücksetzen"), Ergebnis-Karten. Alles bleibt sticky-on-scroll wie heute.
- **Mobile (≤ 768px):** Pilz-Lab zuerst (Pilz, Slider, Chips-Zeilen), dann Ergebnisse. Ergebnis-Count sticky oben. **Kein horizontales Scrollen** bei 390px.

## Der SVG-Pilz (Kernstück)
Eine **sauber gezeichnete, stilisierte Seitenansicht** eines Pilzes (Hut, Lamellen-Unterseite, Stiel), `viewBox` ca. 200×300, in Schichten mit IDs/`data-part`. Stil: modern flach mit dezenten Gradienten, Wald-Palette, gut lesbar auf hellem Hintergrund. Der Pilz zeigt **immer die aktuell gewählten Farben/Formen** (default = neutral grau-beige, wenn nicht gewählt).

### Interaktion
- **Hover/Fokus:** Part hellt sich auf, `cursor:pointer`, nativer Tooltip (label + aktuelle Wahl). Tastaturbedienbar: jeder Part ist ein `<g role="button" tabindex="0">`, Enter/Space = öffnen.
- **Klick auf Part:** öffnet einen **Swatch-Popover** (kleines Panel neben dem Part; mobile: unten als Bottom-Sheet) mit den Optionen dieses Parts. Auswahl setzt das Tag. **Klick auf die bereits aktive Option = Tag wird gewischt (entfernt)** — so „wischt" der Nutzer Eigenschaften wieder weg. Klick auf anderes Part schließt den Popover.
- **Aktive Wahl am Pilz sichtbar:** z.B. ausgewählte Hutfarbe ist die Füllung, Ring/Netz/Knolle erscheinen als gezeichnete Elemente, Essbarkeits-Aura leuchtet.
- **Reset-Button** „Alles wischen" neben dem Pilz.

### Parts und Mapping (Teil → existente Tags)
| Part (am Pilz) | Options-UI | Tags (facets.json, exakt) |
|---|---|---|
| **Hutfarbe** | 10 Farbswatches | `weiß` `gelb` `braun` `schwarz` `rot` `orange` `blau` `gruen` `rosa` `grau` |
| **Hutform** | Mini-Icons (5+1) | `spitz` (spitz/conical), `schirm` (schirmförmig), `kugel` (kugelig), `trichter` (trichterförmig), `f` (fächerförmig), `sonstige_form` (normal/abgeflacht) |
| **Hutoberfläche** | Icons | `hut_gerieft` `hut_hygrophan` `flocken` (flockig) `haut_schmierig` |
| **Lamellenart** | 3 Icons (Unterseite: Lamellen/Röhren/Stacheln) | `lamellen` `roehren` `stachel` |
| **Lamellenfarbe** | 10 Swatches (Unterseite) | `lr_weiß` `lr_gelb` `lr_braun` `lr_schwarz` `lr_rot` `lr_orange` `lr_blau` `lr_gruen` `lr_rosa` `lr_grau` |
| **Stielfarbe** | 10 Swatches | `stiel_weiß` `stiel_gelb` `stiel_braun` `stiel_schwarz` `stiel_rot` `stiel_orange` `stiel_blau` `stiel_gruen` `stiel_rosa` `stiel_grau` |
| **Stieldetails** | Icons (alle kombinierbar! Mehrfachauswahl erlaubt) | `ring` `netz` `knollig` `wurzel` `stiel_hohl` |
| **Fruchtfleisch** (Schnitt: kleine Schnittansicht am Stiel) | 10 Swatches + 3 Icons | `fl_weiß` `fl_gelb` `fl_braun` `fl_schwarz` `fl_rot` `fl_orange` `fl_blau` `fl_gruen` `fl_rosa` `fl_grau` `milch` `bruechig` `faserig` |
| **Wuchsform** (kleine Icons links/unten am Pilz) | 4 Icons (kombinierbar) | `bueschelig` (Strauß), `holz` (auf Holz), `auf_boden`, `auf_wiese` |
| **Essbarkeit** | 5 farbige Buttons (unten am Pilz als „Aura" um den Pilz: grün-grün/grau/orange/rot + Badge) | `hervorragend` `gut` `schlecht` `giftig` `tot` |

Wichtig: **Farbgruppen (Hut/Lamellen/Stiel/Fleisch) = Radio-Verhalten** (eine Farbe pro Gruppe). **Detailgruppen (Hutoberfläche, Stieldetails, Wuchsform, Fleisch-Icons) = Multi** (kombinierbar). Essbarkeit = Radio. Das Ergebnis ist weiterhin **AND** über alle aktiven Tags (Server-Logik).

## Live-Updates ohne Reload
- Neue Endpunkt **`/api/search`**: `GET /api/search?q=&f=&dmin=&dmax=&page=` → JSON:
  ```json
  {"count": 181, "page": 1, "total_pages": 4, "per_page": 50,
   "results": [{"slug":"…", "name":"…", "edibility":"gut", "edibility_label":"Gut essbar", "image":"/media/…/x.jpg"}]}
  ```
  Gleiche Filterlogik wie die HTML-Seite; `image` = best_images-Logik (größtes JPG), Fallback erstes JPG.
- JS hält State (`f`, `dmin`, `dmax`, `q`, `page`), rendert Karten mit einem kleinen JS-Template (gleiche Karten-HTML/CSS-Klassen wie heute, inkl. Essbarkeits-Badge), aktualisiert `history.replaceState` (URL bleibt shareable) und scrollt NICHT.
- Paginierung: „Seite x von y" + Weiter/Zurück-Buttons in der Ergebnisliste (nur im JS-Flow; normale Navigation geht weiter).
- **Fallback:** wenn `fetch` fehlschlägt, normales Page-Reload mit Query-Param.

## Ergebnisliste
- Count-Zeile + Chips (aktiv, einzeln ✕-entfernbar) bleiben wie heute.
- Karten: best-image, Name, lat. Name, Essbarkeits-Badge — bestehende Klassen beibehalten.

## Abnahmetests (alle müssen passen)
1. Lade `http://127.0.0.1:18080/` → SVG-Pilz sichtbar, kein Piktogramm-Grid mehr.
2. Klick Hutfarbe → Swatches → „braun" → **ohne Reload** count = 2875, URL enthält `f=braun`, Pilz-Hut wird braun.
3. Klick „braun" erneut → Tag weg, count zurück auf Grundgesamtheit (4179).
4. Kombi: braun + Lamellen weiß + Ring + Gut essbar → identische Trefferzahl wie `?f=braun,lr_weiß,ring,gut` per GET.
5. `?f=braun,lamellen,gut&dmin=5&dmax=12` direkt per URL → Pilz zeigt die Auswahl (Zustand aus URL rekonstruieren), 181 Treffer.
6. Essbarkeits-Aura: „giftig" → roter Glow/Badge sichtbar am Pilz.
7. `/api/search?f=braun` count == count der HTML-Seite `?f=braun`.
8. Mobile 390px: kein horizontales Scrollen, Popover als Bottom-Sheet bedienbar.
9. `go test ./...` grün, Docker-Build, Container auf 18080.

## Stil
- Modern, ruhig, Wald-Optik (dunkles Grün, Warmgrau, Akzent Orange). Kein Cartoon-Overkill — der Pilz soll **klar und anatomisch nachvollziehbar** sein (Lamellen erkennbar, Stiel, optional Vola/Ring).
- Labels deutsch (aus `facets.json`-Labels).
- Barrierefrei: `title`/`aria-label` pro Part, Tastatur-Navigation.
