# TASK5: Mobile-Interaktions-Bugs fixen (Fokus-Verlust in Suche, Scroll-Jump am Slider)

## Kontext
Go-SSR-Webapp (nur stdlib) + Vanilla JS ohne Framework. Betroffen: `internal/web/static/app.js`, ggf. `internal/web/static/app.css`. Port 18080, Docker-Compose. Vorherige Läufe: TASK1–TASK4 (SVG-Pilz-Lab, /api/search JSON).

## Report (echtes iOS-Smartphone, Viewport ~390px)
1. **Suche verliert Fokus:** In der Suche (`#species-search`, oben in `.hero`) tippt man — mit jedem Tastendruck wird 260 ms später `updateResults()` → `renderResults()` aufgerufen, das `#species-results.innerHTML` komplett ersetzt (30 Karten), zusätzlich `renderActiveFilters()` (`#active-filters.innerHTML`) und `history.replaceState`. Die sich ändernde Dokumentenhöhe + Re-Render lässt iOS Safari die Seite neu scrollen und das fokussierte Input-Field blurren → die virtuelle Tastatur klappt zu, mitten im Tippen.
2. **Seite springt nach oben beim Slider:** Die Range-Inputs `#diameter-min` / `#diameter-max` (in `section.diameter-filter` innerhalb `aside.mushroom-lab`; mobile ist `.mushroom-lab` `position: relative`, NICHT sticky). Berührt/Zieht man den Slider, löst derselbe Re-Render-Pfad (`scheduleUpdate(180)`) aus → die Seite scrollt an den Seitenanfang (scrollY ≈ 0), der Slider ist damit nicht bedienbar.

Gemeinsame Root-Cause: aggressiver Full-DOM-Re-Render bei jedem Input-Event ändert die Dokumentenhöhe während der Interaktion; iOS Safari resettet dabei Scrollposition und Fokus. In Desktop-Chrome/Headless nicht reproduzierbar — daher die Fixes defensiv bauen.

## Zu implementieren (nur `app.js` + ggf. 2 CSS-Zeilen)

### A. No-Op-Guard gegen redundante Re-Renders (app.js, `updateResults`)
- Vor dem Render: `const key = paramsForRequest().toString();`
- Modulebene: `let lastRenderKey = "";`
- Nach dem erfolgreichen Fetch (nach dem `requestId !== requestSequence`-Check): wenn `key === lastRenderKey` → ganz ohne `renderResults`/`renderActiveFilters`/`syncURL` zurück (nur `lastRenderKey` ist schon gleich). Erst bei geänderter `key` rendern und `lastRenderKey = key;` setzen.
- Achtung: bei gleicher key aber anderer Datenmenge (Server-Seitenwechsel) nicht relevant, da page in params ist — key deckt das ab.

### B. Scrollposition erhalten (app.js, `renderResults`)
- Am Anfang von `renderResults(data)`: `const y = window.scrollY;`
- Direkt nach dem innerHTML-Swap (noch im selben Synchronous-Block, vor Pagination-Write OK): `window.scrollTo({ top: y, behavior: "instant" });`
- Dadurch wird der Jump beim Slider-Render eliminiert, egal ob iOS scrollt.

### C. Fokus der Suche erhalten (app.js, `updateResults`)
- Vor dem Render-Block: `const searchFocused = document.activeElement === searchInput;`
- Nach `renderResults`/`renderActiveFilters`: `if (searchFocused && searchInput) searchInput.focus({ preventScroll: true });`
- `preventScroll: true` verhindert, dass der Browser beim Re-Focus wieder in die Nähe des Inputs scrollt.

### D. iOS-Scroll-Coupling am Slider verhindern (app.css)
- `.diameter-range { ... touch-action: none; }` (zusätzlich zur bestehenden Regel `.diameter-range { width: 100%; accent-color: var(--green); cursor: pointer; }` → `touch-action: none;` ergänzen).
- Damit übernimmt der Range-Input alle Touches; iOS scrollt das Dokument nicht mehr mit, solange der Finger auf dem Slider liegt. (Native range-Inputs funktionieren damit auf iOS 15+.)

## NICHT ändern
- API `/api/search`, Debounce-Werte (260/180 ms), `toggleTag`, Popover-Logik, URL-State (`syncURL` bleibt beim echten Key-Wechsel), Templates.
- Kein Framework, keine zusätzlichen Dateien.

## Abnahme (werden per Browser + curl geprüft)
1. `node --check internal/web/static/app.js` → exit 0.
2. `sg docker -c 'docker compose config --quiet'` → OK (falls nichts Docker-relevantes geändert).
3. Desktop + 390px-Emulation: Suche "p" → Ergebnisliste rendert, Fokus bleibt im Input (aktives Element), URL `?q=p`.
4. 390px: scrollen bis Slider sichtbar (scrollY > 500), `dmin` auf 5 setzen → `window.scrollY` bleibt ±10px beim vorherigen Wert (kein Jump auf 0), URL enthält `?dmin=5`.
5. Gleiche Parameter doppelt: zweiter identischer Render ersetzt `#species-results`-innerHTML NICHT (No-Op-Guard) — verifizierbar durch `lastRenderKey`-Logik, visuell durch stabile DOM-Node-Referenz.

## Stil
Im bestehenden Code-Stil bleiben: IIFE-freies Modul-Top-Level, keine Typ-Anotationen, Template-Strings wie im restlichen File. Kommentare sparsam, Deutsch oder Englisch wie vor Ort.
