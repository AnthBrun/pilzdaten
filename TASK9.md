# TASK9: Pilz-Lab-Rework — Essbarkeit als Chip, Boden-Standard, Büschel-Minis, Substrat näher

Vier zusammenhängende UX-Änderungen am Pilz-Lab (SVG + Template + app.js + app.css + facets.json). Keine neuen Dateien, keine Bibliotheken, bestehender Code-Stil. Port/Build unverändert.

## Dateigrundlage (aktuell)
- `internal/web/templates/index.html` — SVG `#mushroom` (viewBox `0 0 240 372`), `#option-deck`, `section.nonvisual-traits` (Zeile 109–112 rendert nur Gruppen spore/geruch/farbwechsel).
- `internal/web/static/app.js` — `partNames`, `partChoices`, `groups`, `state`, `updateMushroom()` (Zeile 164ff.), Chip-Handler Zeile ~345.
- `internal/web/static/app.css` — `.substrate*` (Zeile 80–97), `.edibility-*`, `.svg-note`.
- `../pilzsuche/data/facets.json` — Gruppe `"essbarkeit"` (title "Essbarkeit & Vorkommen") enthält 10 Tags: hervorragend, gut, schlecht, giftig, tot, nicht_selten, holz, auf_boden, auf_wiese, bueschelig.

## 1) Essbarkeit aus dem Pilz → Chip-Gruppe "Nicht zeizenbare Merkmale"
Ziel: Essbarkeit ist kein klickbarer Pilz-Part mehr (keine Aura, kein Punkt, kein Label), sondern eine normale Chip-Gruppe neben Sporenpulver/Geruch/Farbwechsel. Die Essbarkeit-Badges auf den Ergebnis-Karten bleiben unverändert.

**facets.json** — Gruppe `"essbarkeit"` aufspalten in zwei Gruppen (Tags, Reihenfolge, `count`-Werte pro Tag bleiben wie aktuell erhalten — nur Gruppierung ändert sich):
- `"essbarkeit"` (title "Essbarkeit"): `hervorragend`, `gut`, `schlecht`, `giftig`, `tot`
- `"vorkommen"` (title "Vorkommen"): `nicht_selten`, `holz`, `auf_boden`, `auf_wiese`, `bueschelig`
  (Die `vorkommen`-Gruppe wird NICHT als Chips gerendert — sie wird über den Untergrund-Part gesetzt. Sie existiert nur, damit `#facet-labels` alle Labels liefert und die Gruppierung konsistent ist.)

**index.html** — In `section.nonvisual-traits` eine vierte Chip-Gruppe für `essbarkeit` ergänzen, identisch im Aufbau zu spore/geruch/farbwechsel, aber mit `data-trait-group="edibility"` auf jedem Chip (Radio-Verhalten):
```html
{{if eq .Group "essbarkeit"}}<div class="trait-chip-group"><h5>Essbarkeit</h5><div class="trait-chips">{{range .Options}}<button class="trait-chip" type="button" data-trait-tag="{{.Tag}}" data-trait-group="edibility">{{.Label}}</button>{{end}}</div></div>{{end}}
```
Die Gruppe `vorkommen` darf in der Section nicht gerendert werden (Conditionen sind `eq .Group "spore"|"geruch"|"farbwechsel"|"essbarkeit"`).

**index.html (SVG)** — Den `<g class="mushroom-part" data-part="edibility">` (Punkt + `#edibility-label`, Zeile 90–92) komplett entfernen. Das `<ellipse id="edibility-aura">` (Zeile 39) entfernen.

**app.js**:
- `partNames`: Eintrag `edibility: "Essbarkeit"` entfernen.
- `partChoices`: Schlüssel `edibility` entfernen.
- `groups`: `edibility: { edibility: new Set(["hervorragend","gut","schlecht","giftig","tot"]) }` BEHALTEN — das Radio-Isolations-Set braucht der Chip-Handler weiter (siehe unten).
- `updateMushroom()`: den gesamten Essbarkeit-Block entfernen (`const edibility = activeFrom(...)` über `#edibility-label` / `#edibility-aura` / `aura-${...}`).
- Chip-Handler (Zeile ~345) so anpassen, dass er ein optionales `data-trait-group` weitergibt (Default bleibt `"multi"`):
  ```js
  document.querySelectorAll("[data-trait-tag]").forEach((button) => button.addEventListener("click", () => toggleTag(button.dataset.traitTag, button.dataset.traitGroup || "multi")));
  ```
  So ist Essbarkeit im Deck/Chip ein Radio-Tag (neue Wahl ersetzt die alte), weil `toggleTag(tag,"edibility")` über `groups` die anderen Essbarkeits-Tags entfernt.

**app.css** — `.edibility-aura`, `.edibility-dot`, `.mushroom-part[data-part="edibility"]...` und die `aura-*`-Regeln entfernen, die nur noch ohne Part existieren (unverwendet). Die `.edibility-badge`-Stile der Ergebnis-Karten bleiben (andere Klassen). Die `.edibility-choice .edibility-dot-choice`-Stile des Decks ebenfalls entfernen, da kein Essbarkeit-Deck-Part mehr existiert.

## 2) Text „Untergrund" entfernen + Boden als Standard
- **index.html (SVG)**: `<text id="growth-label">…Untergrund…</text>` (Zeile 62) komplett entfernen.
- **app.js** `updateMushroom()`: die beiden `#growth-label`-Zeilen (`const growthLabels = …` und `document.querySelector("#growth-label").textContent = …`) entfernen.
- **Boden als Standard**: Die Erd-Substrat-Sichtbarkeit ändert sich von "nur wenn `auf_boden` ausgewählt" zu **immer sichtbar als Grundfläche**, auf der der Pilz steht. Konkret in `updateMushroom()`: die Erd-Substrat auf `is-visible` setzen unabhängig von `auf_boden` (z.B. `document.querySelector('.substrate[data-substrate="erde"]').classList.add("is-visible")` als festen Zustand, nicht mehr per toggle). Die anderen drei (holz/wiese/bueschel) bleiben per `auf_*`/`bueschelig`-Tag getoggelt wie bisher. `auf_boden` bleibt weiterhin als wählbarer Filter-Tag im Untergrund-Deck (die Visualisierung ist dann einfach ohnehin an).

## 3) Büschelig/Strauß → kleine Kopien des großen Pilzes
Bedeutung von "büschelig": Pilze sehr dicht und in hoher Zahl an einer Stelle. Das aktuelle Strich-Symbol passt nicht → durch **kleine Pilz-Kopien** ersetzen, die idealerweise die Eigenschaften des großen Pilzes mitvererben.

- **index.html (SVG)**: Inhalt von `<g class="substrate substrate-cluster" data-substrate="bueschel">` ersetzen durch eine Reihe von 4–6 Mini-Pilzen auf der Bodenlinie. Jeder Mini-Pilz ist eine skalierte Kopie aus Hut + Stiel:
  ```html
  <g class="substrate substrate-cluster" data-substrate="bueschel" aria-hidden="true">
    <path class="ground-line" d="M48 300h144"/>
    <g class="mini-mushroom" transform="translate(58 300) scale(0.22)"><path class="mini-cap"/><rect class="mini-stem" x="106" y="132" width="28" height="58" rx="9"/></g>
    <g class="mini-mushroom" transform="translate(96 300) scale(0.28)"><path class="mini-cap"/><rect class="mini-stem" x="106" y="132" width="28" height="58" rx="9"/></g>
    <g class="mini-mushroom" transform="translate(132 300) scale(0.20)"><path class="mini-cap"/><rect class="mini-stem" x="106" y="132" width="28" height="58" rx="9"/></g>
    <g class="mini-mushroom" transform="translate(166 300) scale(0.25)"><path class="mini-cap"/><rect class="mini-stem" x="106" y="132" width="28" height="58" rx="9"/></g>
  </g>
  ```
  Die Mini-Pilze stehen dicht beieinander (überlappend erlaubt), variierende Skalen für natürliches Gesträuß. `translate`-y so, dass die Mini-Bodenlinie bei y≈300 liegt (siehe Punkt 4). Die exakten Koordinaten/Skalen darf Opencode so wählen, dass die Minis kompakt und lesbar direkt am Boden sitzen (keine cm-Leerstelle).
- **app.js** `updateMushroom()`: die Mini-Pilze mit den aktuellen Eigenschaften des großen Pilzes pflegen — jede `.mini-cap` bekommt das aktuelle Hut-Form-`d` (aus `shapePaths[selectedShape]`, Default `sonstige_form`) und `fill` = aktuelle Hutfarbe (`capColor ? colorFor(capColor) : "url(#capShade)"`); jede `.mini-stem` bekommt `fill` = aktuelle Stielfarbe (`stemColor ? colorFor(stemColor,"stiel_") : "url(#stemShade)"`). Am Ende von `updateMushroom()` eintragen:
  ```js
  const capD = shapePaths[selectedShape] || shapePaths.sonstige_form;
  const capFill = capColor ? colorFor(capColor) : "url(#capShade)";
  const stemFill = stemColor ? colorFor(stemColor, "stiel_") : "url(#stemShade)";
  document.querySelectorAll(".mini-cap").forEach((p) => { p.setAttribute("d", capD); p.setAttribute("fill", capFill); });
  document.querySelectorAll(".mini-stem").forEach((p) => { p.setAttribute("fill", stemFill); });
  ```
  (shapePaths/capColor/stemColor/selectedShape sind in `updateMushroom()` bereits vorhanden — nur wiederverwenden.)
- **app.css**: `.mini-cap { stroke: none; }`, `.mini-stem { stroke: none; }` (Mini-Kopien ohne Kontur, flach wie Silhouetten). Die alten `.substrate-cluster .substrate-base`-Regeln entfernen.

## 4) Untergrund direkt unter den Pilz (keine cm-Leerstelle)
Der Stiel endet bei y≈305 (Bulb/Roots). Die Substrate liegen aktuell mit Oberkante bei y≈307–326 → spürbarer Abstand. Alle Substrate so nach oben setzen, dass ihre **Oberkante bei y≈305** liegt (direkt unter dem Stiel, maximal 2–4 px Luft):
- Erd-Hügel: Oberkante des Hügels auf y≈305 (aktuell ~326 → ~21 px hoch).
- Holz-Stamm: `rect` y auf ~305 (aktuell 307, OK, minimal anpassen), Jahresringe/Äste entsprechend.
- Wiese: `ground-line` + Grashalme so, dass die Bodenlinie bei y≈305 liegt (aktuell ~330 → hoch).
- Büschel: `ground-line` bei y≈305 (siehe Punkt 3).
- Hit-Fläche `<rect class="substrate-hitarea">` (aktuell y=298 h=52) anpassen, dass sie die Substrat-Zone ab y≈300 bis ~y≈330 abdeckt.
- **viewBox**: Da `#growth-label` (y≈364) und `#edibility-label`/Punkt (y≈348–352) entfernt werden, leergezogene Fläche unten abschneiden → `viewBox` von `0 0 240 372` auf `0 0 240 330` setzen (genug für Substrat-Oberkante 305 + Substrat-Höhe + kleiner Rand). `#edibility-aura`-Ellipse ist bereits entfernt (Punkt 1).
- Die Deck-Animation `transform: translateY(26px)` von `.substrate` darf bleiben (gleitet weiter von unten ein), aber der Endzustand `translateY(0)` muss die Substrate auf y≈305 landen lassen.

## Abnahme (per Browser 1280px + 390px + curl)
1. `node --check internal/web/static/app.js` → 0. `sg docker -c 'docker compose config --quiet'` → OK.
2. **Essbarkeit**: Kein klickbarer Pilz-Part/kein Punkt/keine Aura mehr im SVG. In "Nicht zeizenbare Merkmale" ist eine Chip-Gruppe "Essbarkeit" mit 5 Chips (Hervorragend essbar … Essbar? je nach Label). Chip "Giftig" anklicken → URL `?f=giftig`; dann "Gut" → `giftig` wird durch `gut` ersetzt (Radio, nicht kumulativ). Ergebnis-Badges zeigen weiter die Essbarkeit.
3. **Untergrund-Text**: Kein "Untergrund"-Text im SVG mehr. Bei leerem Filter ist die Erd-Grundfläche **sichtbar** (Pilz steht auf Boden). Holz wählen → Holz zeigt sich; Erd-Grundfläche bleibt als Basis sichtbar.
4. **Büschel**: "Büschelig" wählen → mehrere kleine Pilz-Kopien am Boden (nicht mehr Striche). Hutfarbe auf "rot" + Form "spitz" setzen → die Mini-Pilze werden rot + spitz (erben die Eigenschaften). Hutfarbe "braun" → Minis braun.
5. **Abstand**: Substrat-Oberkante liegt direkt unter dem Stiel (y≈305), keine Leerstelle. viewBox `0 0 240 330`. Auf 390px: kein Clipping, Substrat sichtbar, kein horizontaler Scroll.
6. **Regression**: Untergrund-Deck (4 Karten mit Minis), Pfeile/Multi-Verhalten, Filter-URL, `?f=grau,lamellen,lr_weiß,stiel_weiß` → UI-Count = API-Count (368), Scroll-Pin + Fokus in Suche unverändert.

## Stil
Bestehender Code-Stil, keine neuen Dateien. Alle SVG-Koordinaten so anpassen, dass es optisch sitzt (Opencode darf die exakten Pixel iterativ wählen, solange die Abnahme-Ergebnisse stimmen). Keine JS-Bibliothek.
