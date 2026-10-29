# TASK7: Options-Deck (Carousel) statt Popover — wie Charakter-Ersteller (Sims/Skyrim)

## Kontext
Go-SSR + Vanilla JS, SVG-Pilz-Lab. TASK4/6 brachten Popover + Wisch-Geste. User-Feedback: **Wischen-über-den-Pilz war Falschverstehen.** Gewünscht: Beim Antippen eines Parts werden die Optionen **direkt als sichtbare Reihe (Carousel) dargestellt** — man scrollt mit dem Finger horizontal durch die Items ODER iteriert per **Pfeil-Buttons ‹ ›** durch die Auswahl (zirkulär). Wie in Character-Creators: Part markieren → Auswahlleiste mit Items, ausgewähltes Item hervorgehoben/zentriert.

## Datei-Übersicht (aktuell)
- `internal/web/templates/index.html`: `#swatch-popover` (Zeile ~91, absolute, in `.mushroom-lab`) mit `#popover-title`, `#popover-close`, `#popover-options`.
- `internal/web/static/app.js`: `partChoices` (6 Parts, je 1–3 Sections mit `kind` color/icon/edibility + `group` multi/colors/shape/type/edibility), `openPopover()`, `closePopover()`, `renderPopover()`, `toggleTag()`, `updateMushroom()`; **TASK6-Code der ENTFERNT wird**: `cyclePart`, `showCycleFeedback`, `cycleBadgeTimer`, `cyclePulseTimers`, die `pointerdown/pointermove/pointerup/pointercancel`-Handler + `suppressNextClick`-Logik im `mushroom-part`-forEach.
- `internal/web/static/app.css`: `.swatch-popover*`, `.option-section`, `.option-list`, `.popover-choice`, `.cycle-badge`, `popoverIn`/`badgeIn`-Keyframes, mobile Bottom-Sheet (Zeile ~255).

## Ziel-Design

### Struktur (index.html)
`#swatch-popover` komplett ersetzen durch:
```html
<div id="option-deck" role="dialog" aria-modal="false" aria-label="Optionen auswählen" hidden>
  <div class="deck-header">
    <strong id="deck-title">Merkmal wählen</strong>
    <button id="deck-close" type="button" aria-label="Schließen">×</button>
  </div>
  <div id="deck-tabs" class="deck-tabs"></div>          <!-- Pills, 1 pro Section des Parts (nur wenn Part >1 Sections hat) -->
  <div class="deck-carousel-row">
    <button id="deck-prev" class="deck-arrow" type="button" aria-label="Vorherige Option">‹</button>
    <div id="deck-carousel" class="deck-carousel"></div>
    <button id="deck-next" class="deck-arrow" type="button" aria-label="Nächste Option">›</button>
  </div>
</div>
```

### Verhalten (app.js)
1. **Part antippen/Klick** → Deck öffnet auf diesem Part (Click-Handler der `.mushroom-part` bleibt, `openPopover` → `openDeck(part)` umbenennen; Keydown-Enter/Space bleibt).
2. **Tabs:** Für jedes `partChoices[part]`-Element eine Tab-Pill mit `section.title` (z.B. „Hutfarbe · Hutform · Hutoberfläche"). Aktive Section fett/hervorgehoben. Nur rendern, wenn das Part >1 Sections hat (sonst Tabs-Container `hidden`).
3. **Carousel** rendert die **aktive Section**:
   - **Radio-Sections (kind color/icon/edibility, group != multi):** Horizontal scrollbare Reihe von Options-Karten. Karte = bestehende Darstellung (Swatch-Kreis + Label / Icon-Kästchen + Label / Edibility-Dot + Label). Ausgewählte Karte: hervorgehoben (dickere grüne Border, heller Hintergrund) UND **zentriert im Carousel** (`scrollTo` mit `behavior: "smooth"`, Ziel = Kartemitte − Carousel-Halberbreite).
   - **Multi-Sections (group multi, z.B. Hutoberfläche, Stieldetails, Untergrund):** Gleiche Karten, aber **Toggle-Verhalten** (mehrfach aktiv, `is-selected` pro Karte), **Pfeile werden ausgeblendet** (`hidden`), Karte-Tap toggelt den Tag.
   - Untergrund-Part (`growth`): die 4 Karten bekommen je ein kleines Inline-SVG-Icon, das die jeweilige Substrat-Lage miniert (einfache Wiederverwendung der TASK6-Strichzeichnungen als 28×18px-SVG in der Karte, z.B. über eine `data-substrate`-Klasse + CSS-Bedingung oder direkt im JS-Template als SVG-Snippet).
4. **Pfeile ‹ ›** (nur bei Radio-Sections): **iterieren die Auswahl** um ±1 Tag in der Reihenfolge von `section.tags` (wrap-around, zirkulär). Aufruf `toggleTag(nextTag, section.group)` — Wipe-Logik bleibt identisch (Radio-Isolation pro Part). Nach dem Wechsel: neue Karte zentrieren + Part kurz pulsieren (`.is-cycled` 400ms, Keyframe bleibt).
5. **Tastatur:** Deck offen + `ArrowLeft`/`ArrowRight` (Document-Listener, `Escape` schließt wie bisher) → Pfeil-Verhalten.
6. **Scrollen mit dem Finger:** `#deck-carousel { overflow-x: auto; scroll-snap-type: x mandatory; -webkit-overflow-scrolling: touch; }`, Karten `scroll-snap-align: center;`. Swipe-in-the-List = natives Scrollen, KEIN JS-Pointer-Tracking nötig.
7. **Schließen:** `#deck-close`, Escape, Klick außerhalb von `.mushroom-part`/`#option-deck` (bestehender Document-Click-Handler anpassen), Wechsel auf anderes Part (Deck bleibt offen, Inhalt wechselt).
8. **TASK6-Wisch-Geste komplett entfernen** (s. oben). `showCycleFeedback`/`cycle-badge`/`cyclePart`/`cycleBadgeTimer`/`cyclePulseTimers` + alle `pointerdown/move/up/cancel`-Listener auf `.mushroom-part` löschen. Stattdessen: `toggleTag` zeigt bei Radio-Wechsel das Pulsieren des Parts (neue kleine Funktion `pulsePart(part)`), kein Badge mehr.
9. **URL-State/No-Op-Guard/Filter-Chips** bleiben unverändert (sie greifen über `state.tags`).

### CSS (app.css)
- `.swatch-popover*`-Regeln durch `#option-deck`-Regeln ersetzen:
  - **Desktop (≥769px):** `#option-deck` = absolute, `left: 0; right: 0; bottom: 6px` in `.mushroom-lab` (liegt unter der Stage, deckt den Filter-Sektionen-Überhang NICHT ab — `z-index: 6`). Breite = Kartenbreite, `max-height: 190px; overflow-y: auto;` Padding wie bisher, weiß, Radius, Schatten wie altes Popover. Öffnen: `animation: deckIn .18s ease` (fade + translateY(-6px)).
  - **Mobile (≤768px):** bestehendes Bottom-Sheet-Verhalten übernehmen: `position: fixed; left:0; right:0; bottom:0; top:auto; max-height: 72vh; border-radius: 16px 16px 0 0;` + `padding-bottom: env(safe-area-inset-bottom)` + kleiner Drag-Handle (12×4px abgerundeter Balken, mittig, oben im Deck). Öffnen: `deckSheetIn .2s ease` (translateY(30px)→0 + fade).
- `.deck-tabs`: Zeile aus Pills (flex, wrap, gap 5px, margin 8px 0 6px), aktive Pill: dunkleres Grün + weißer Text.
- `.deck-carousel-row`: flex, align-items center, gap 4px; `.deck-arrow`: 34×34px runde Buttons, `flex: none`, Rand-Hover, auf Touch `min-height: 38px`.
- `.deck-carousel`: `flex: 1; display: flex; gap: 6px; overflow-x: auto; scroll-snap-type: x mandatory; -webkit-overflow-scrolling: touch; padding: 4px 2px;` Scrollbar dezent (`::-webkit-scrollbar` height 4px, `#d5dbd0`).
- Options-Karten: bestehenden `.popover-choice`-Stil übernehmen/umbenennen (`.deck-option`), `flex: none; min-width: 76px; scroll-snap-align: center;` (Color-Karten dürfen schmaler: `min-width: 64px`), `is-selected`-Zustand = grün hervorgehoben.
- `prefers-reduced-motion`: `scroll-behavior`-Smooth und `deckIn`/`deckSheetIn` ausschalten (bestehender Block erweitern).

## Abnahme (per Browser + curl)
1. `node --check internal/web/static/app.js` → exit 0; `sg docker -c 'docker compose config --quiet'` → OK.
2. Cap antippen → Deck öffnet mit Tabs „Hutfarbe / Hutform / Hutoberfläche", aktive = Hutfarbe; Carousel zeigt 10 Farbkarten.
3. `#deck-next` 3× klicken → Hutfarbe wandert um 3 (weiß→gelb→braun→schwarz bei Start ohne Auswahl: erstes Next setzt „weiß"), jeweils Karte zentriert, Part pulsiert, URL `?f=…` stimmt.
4. Pfeile bei „Hutoberfläche" (multi) ausgeblendet; Karten-Tap toggelt (z.B. `hut_gerieft` + `flocken` gleichzeitig aktiv).
5. Untergrund-Part: 4 Karten mit Minis-Icons; Auswahl `holz` → Karte aktiv + Substrat-Lage im SVG sichtbar (TASK6 bleibt).
6. Keyboard: Deck offen, `ArrowRight` → Auswahl +1.
7. **Mobile 390px:** Deck = Bottom-Sheet (fixed, unten, Drag-Handle), kein horizontaler Seitenscroll, Carousel per Touch-Scroll durchgehend (scrollLeft ändert sich), Pfeile tippbar (≥38px).
8. **TASK6-Wisch-Geste weg:** `document.querySelector('[data-part="cap"]')` hat KEINE `pointerdown`-Listener mehr (Verifikation: synthetische pointer-Geste auf dem Cap verändert NICHTS).
9. Regression: Tags `grau`+`lamellen`+`lr_weiß`+`stiel_weiß` setzen → UI-Count = `/api/search`-Count (wie TASK6-Test, 368).
10. Schließen: ×, Escape, Klick außerhalb; `hidden`-Attribut korrekt.

## Stil
Bestehender Code-Stil. `openPopover`/`closePopover`/`renderPopover` → `openDeck`/`closeDeck`/`renderDeck` umbenennen (alle Aufrufstellen mitnehmen, z.B. `toggleTag` ruft `renderPopover(openPart)` — umbenennen in `renderDeck(openPart)`). Keine neuen Dateien, keine Bibliotheken.
