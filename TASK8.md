# TASK8: Bugfix — Tab-Tap schließt das Options-Deck (1 Zeile)

## Bug (diagnostiziert & reproduziert)
Im Options-Deck (`#option-deck`) schließt **jedes** Tab-Tap das Deck sofort, anstatt die Section zu wechseln. Betroffen: alle Parts mit ≥2 Sections (Hut, Stiel, Unterseite, Fruchtfleisch). Die Mehrfach-Options-Karten funktionieren, Tabs aber nicht → Kern-Feature gebrochen.

## Root Cause
In `internal/web/static/app.js`:
1. `deckTabs`-Click-Handler (~Z.333) ruft `renderDeck(openPart)` → setzt `deckTabs.innerHTML = …` neu. Das **ersetzt den gerade geklickten Tab-Button** (Bubble-Phase, läuft vor dem Document-Handler).
2. Danach läuft der Document-Click-Handler (~Z.346):
   ```js
   document.addEventListener("click", (event) => {
     if (!deck || deck.hidden || event.target.closest("#option-deck, .mushroom-part")) return;
     closeDeck();
   });
   ```
   `event.target` ist der **entkoppelte (detached)** Tab-Button → `event.target.closest("#option-deck")` liefert `null` → Guard schlägt fehl → `closeDeck()` wird aufgerufen → Deck schließt sich.

## Fix (minimal, robust)
Mach den Document-Click-Guard unempfindlich gegen Re-Render-Desync, indem du nicht den (evtl. detacheden) `event.target` prüfst, sondern den **aktuellen Deck-Zustand + composedPath**. Ersetze den Document-Click-Handler durch:
```js
document.addEventListener("click", (event) => {
  if (!deck || deck.hidden) return;
  const inside = event.composedPath().some((node) => node === deck || (node.nodeType === 1 && node.classList && (node.classList.contains("mushroom-part"))));
  if (inside) return;
  closeDeck();
});
```
`event.composedPath()` enthält den vollständigen Pfadelement-Stack **einschließlich des zu diesem Zeitpunkt bereits entfernten Targets** und dessen Eltern, solange der Event noch läuft — zuverlässiger als `closest()` auf einem detacheden Node. (Alternativ akzeptabel: `event.target.isConnected` vor der Prüfung berücksichtigen ODER `deckTabs`-Handler mit `event.stopPropagation()` sichern. Wähle `composedPath()` als primäre Lösung.)

WICHTIG: Nichts anderes ändern. Keine weiteren Refactors.

## Abnahme
1. `node --check internal/web/static/app.js` → exit 0.
2. Per Browser: Cap antippen → Deck öffnet. Tab „Hutform" antippen → Deck bleibt offen, Carousel zeigt Hutform-Optionen (6), aktive Tab = Hutform. Tab „Hutoberfläche" → Pfeile ausgeblendet, 4 Karten. Kein Deck-Schließen durch Tab-Wechsel.
3. Klick außerhalb des Decks schließt es weiterhin. × schließt es. Escape schließt es.
4. Regression: Radio-Pfeile ‹ ›, Multi-Toggle, Untergrund-Minis, Filter-URL-Update alle wie TASK7.
