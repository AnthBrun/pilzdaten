# TASK6: Substrat-Animation unter dem Pilz + Wischen statt nur Klick

## Kontext
Go-SSR + Vanilla JS (ohne Framework), SVG-Pilz-Lab (TASK4) in `internal/web/templates/index.html` + `internal/web/static/app.js` + `internal/web/static/app.css`. Keine JS-Bibliotheken — nur CSS-Transitions/Keyframes + Pointer-Events. Vorherige Läufe TASK1–TASK5 (TASK5: No-Op-Guard, Scroll-Pin, Fokus-Erhalt — NICHT anfassen).

## Anforderung (User)
1. **Untergrund (Substrat) gehört UNTER den Pilz** — nicht als kleines Icon oben links. Gewählt wird, was der Pilz drauf wächst: **Holz, Erde, Wiese, (Büschel)**.
2. **Animation:** Das gewählte Substrat **gleitet unter den Pilz** (Slide-in von unten + Fade-in), wie in einem Charakter-Ersteller.
3. **Wischen statt Klick:** Eigenschaften werden **durch Wischen** auf dem Pilz selbst geändert (horizontal über einen Part wischen = nächster Wert). Tappen öffnet weiterhin den Popover (bestehendes Verhalten bleibt).
4. Generell: schöne, subtile Animationen (Farbwechsel des SVGs fließen, Popover öffnet animiert, Part pulsiert kurz bei Wisch-Wechsel).

## A) Substrat unter dem Pilz (SVG + CSS + JS)

### SVG (`index.html`, `<svg id="mushroom" viewBox="0 0 240 330">`)
- **viewBox auf `0 0 240 372` erweitern** (42 px unten anhängen).
- `#edibility-aura` (cx=120 cy=159) bleibt; **`g[data-part="edibility"]` verschieben: Kreis cy 314→348, Text y 318→352.**
- **Altes growth-Icon löschen** (der `g[data-part="growth"]` mit `growth-icon`-Path oben links, x≈15-80, y≈230-287 — das kleine Büschel-Piktogramm + `#growth-label`).
- **Neues `g`-Element für das Substrat** als TAP-ZONE unter dem Pilz, ersetzt das alte growth-Part:
  ```
  <g class="mushroom-part" data-part="growth" role="button" tabindex="0" aria-label="Untergrund wählen" title="Untergrund wählen">
    <!-- 4 Substrat-Lagen, jede ein <g class="substrate" data-substrate="holz|erde|wiese|bueschel"> -->
    <text id="growth-label" class="svg-note" x="60" y="364">Untergrund</text>
  </g>
  ```
- Substrat-Lagen (einfache, klare Strichzeichnungen, Stil passend zum restlichen SVG — dünne Striche, gleiche Farbpalette wie `app.css` vars; jede Lage deckt ca. x=40…200, y=298…336):
  - **holz**: waagerechter Stamm (abgerundetes Rechteck) mit 2–3 Jahresring-Auschnitten auf der Stirnseite + 2 Rindenstriche.
  - **erde**: braune Bodenkuppe (abgerundetes Pfad) mit 3–4 Steinen/Kies (kleine Ellipsen) + 2 Wurzelstriche.
  - **wiese**: Grundlinie + 6–8 Grashalme (kurze geschwungene Striche) + 1 kleines Blumenkreiselchen.
  - **bueschel**: 3–4 kleine Stiel-Basen in Reih (wie ein Büschel junger Pilze, nur Stielansätze) auf flacher Linie.
- **Keine** `cap-fill`/`is-visible`-Kopplung: jede Substrat-Lage bekommt in `updateMushroom()` die Klasse `is-visible` per Tag: `holz`→`.substrate[data-substrate="holz"]`, `auf_boden`→`erde`, `auf_wiese`→`wiese`, `bueschelig`→`bueschel`. Mehrfachauswahl (Group ist `multi`!) zeigt alle gewählten Lagen gleichzeitig (Lagen überlappen leicht, Reihenfolge im SVG: erde, holz, wiese, bueschel).
- `#growth-label`-Text in `updateMushroom()` anpassen: keine Auswahl → „Untergrund", sonst die gewählten Labels (Bestand: `labelFor()`), z. B. „Untergrund · Holz".

### CSS (`app.css`)
- `.substrate { opacity: 0; transform: translateY(26px); transition: opacity .35s ease, transform .5s cubic-bezier(.2,.9,.3,1.15); pointer-events: none; }`
- `.substrate.is-visible { opacity: 1; transform: translateY(0); }`
- `@media (prefers-reduced-motion: reduce) { .substrate, .popover-choice, #swatch-popover { transition: none; animation: none; } }`
- SVG-Höhe passt sich automatisch an die neue viewBox an (`mushroom-svg` nutzt width:100%); `mushroom-stage` ggf. max-height prüfen, nichts soll clippen (Desktop + 390px).

## B) Wischen auf den Pilz-Parts (Pointer-Events, `app.js`)

### Verhalten
- Für jeden `g.mushroom-part` (außer `edibility` — der bleibt Klick/Radio wie bisher): **Pointer-Down → Pointer-Track**.
- **Tap** (Verschiebung < 8 px bis Pointer-Up): bestehendes Verhalten — Popover öffnen (wie heute per click).
- **Horizontaler Wisch** (|dx| > 24 px, |dx| > |dy|): **nächster (→) bzw. vorheriger (←) Wert** in der **ersten Radio-Gruppe** des Parts (Reihenfolge wie in `partChoices[part]`, `group !== "multi"`). Wrap-around (zirkulär). Aufruf: `toggleTag(nextTag, group)` — Wipe-Logik bleibt dadurch identisch.
- **Vertikaler Wisch** (|dy| > 24 px, |dy| > |dx|) NUR bei Maus-Zeigern (event.pointerType === "mouse"): zweite Radio-Gruppe des Parts (falls vorhanden, z. B. underside: Lamellenart ←horizontal, Farbe ←vertikal; cap/stem/flesh: Farbe ←horizontal; multi-Gruppen nie wischbar).
- **Touch:** `touch-action: pan-y` auf `g.mushroom-part` (CSS) — vertikales Seiting des Seiten-Scrolls bleibt möglich, horizontales Wischen auf dem Pilz wird dem Part zugestellt. Damit muss die Seite bei vertikalem Scrollstart auf dem Pilz NICHT blockiert sein.
- **Feedback:** bei jedem Wisch-Wechsel kurzes Pulsieren des Parts: CSS-Klasse `is-cycled` auf 400 ms (`@keyframes partPulse { 50% { opacity: .55; } }`, `transform-box: fill-box` nicht nötig) + eine kurze schwebende Beschriftung mit dem neuen Wert: kleine `span`-Badge (class `cycle-badge`, position:absolute, Style wie `.svg-note` aber weiß/abgerundet/Schatten) wird in `mushroom-stage` am Rand des Parts geplatzt (oben rechts des Part-BoundingBoxes per `getBoundingClientRect()` relativ zur Stage), 900 ms sichtbar, dann entfernt (ein aktives Badge sofort ersetzen).
- **Wichtig:** `pointerdown`/`pointermove`/`pointerup` mit `setPointerCapture`; `click`-Events nach einem echten Wisch unterdrücken. Der bestehende Part-Click-Handler sitzt bei `part.addEventListener("click", () => openPopover(part.dataset.part));` (app.js, Zeile ~262, im `querySelectorAll(".mushroom-part").forEach` zusammen mit dem Keydown-Handler) — dort via Flag `suppressNextClick` prüfen und einmal verwerfen.
- `edibility`-Part bleibt **nur** Klick (öffnet Popover) — kein Wischen, keine `touch-action`-Änderung.

### Keine Regressionsfläche
- `toggleTag`, `scheduleUpdate`, No-Op-Guard, URL-State, Radio-Isolation (TASK4-Radio-Fix), Popover-Logik bleiben unverändert.

## C) Subtile Animationen (CSS-only)
- **SVG-Farbwechsel fließen:** `.cap-fill, .stem-fill, .underside-fill, .flesh-core, .flesh-slice { transition: fill .35s ease; }`
- **Popover-Öffnen:** `#swatch-popover` bekommt `animation: popoverIn .16s ease` (Desktop: `from { opacity: 0; transform: translateY(-4px) scale(.98); }`, Mobile Bottom-Sheet im bestehenden Media-Query: `from { transform: translateY(24px); opacity: 0; }` via `animation-name` override `.popoverSheetIn`).
- **Optionen im Popover:** `.popover-choice { transition: background-color .15s ease, border-color .15s ease, color .15s ease; }` (hover/selected-States sind schon da).

## Abnahme (wird per Browser + curl geprüft)
1. `node --check internal/web/static/app.js` → exit 0; `sg docker -c 'docker compose config --quiet'` → OK.
2. **Substrat:** ohne Auswahl keine `.substrate.is-visible`; `holz` wählen (Popover oder Wisch) → genau `.substrate[data-substrate="holz"].is-visible`; `auf_wiese` zusätzlich → beide visible (multi); `Alles wischen` → keine visible.
3. **Wischen (Maus, Desktop):** `pointerdown/move/up`-Simultation horizontal über dem Cap (dx=+40) → Hutfarbe wechselt zum nächsten Farb-Tag in `colors`-Reihenfolge, `cycle-badge` erscheint, URL/Filter-Update greift (z. B. `?f=braun` → `?f=gelb` je Reihenfolge). Vertikaler Wisch auf underside (mouse, dy=+40) → Lamellen-Farbe wechselt, Lamellenart bleibt.
4. **Tap bleibt Tap:** kurzer Klick (dx<8) auf Cap → Popover öffnet (Bestand).
5. **Mobile 390px:** vertikales Scrollen mit Startpunkt auf dem Pilz funktioniert (touch-action: pan-y), horizontales Wischen auf dem Pilz scrollt die Seite nicht horizontal.
6. **Animationen:** `getComputedStyle(document.querySelector('#swatch-popover')).animationName` != 'none' nach Öffnen; `prefers-reduced-motion`-Block existiert.
7. **Keine Regression:** kombinierter Filter-Test wie TASK4: `braun` (Hut) + `lr_weiss` (Lamellen) + `gut` → API-Count = UI-Count.

## Stil
Bestehender Code-Stil (kein IIFE, Template-Strings, sparsame Kommentare Deutsch/Englisch). SVG-Strichzeichnungen minimalistisch halten (dünne Striche, vorhandene Klassen-Stil `cap-markings` etc.). Keine neuen Dateien außer dem TASK.
