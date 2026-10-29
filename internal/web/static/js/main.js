// Einstiegspunkt für alle Seiten. Aufbau:
//   config.js   Merkmals-Vokabular und Deck-Optionen
//   state.js    Suchzustand <-> URL
//   mushroom.js Pilzzeichnung an die Auswahl anpassen
//   deck.js     Auswahl-Deck beim Antippen eines Pilzteils
//   results.js  Trefferliste, Filter-Chips, Trefferleiste
//   offline.js  Offline-Paket, Offline-Suche, Offline-Artseite
//   detail.js   Artseite (Lightbox, Zurück-Link)
import { currentMonthTag, DEFAULT_GROUP, sporeColorHex } from "./config.js";
import { closeDeck, currentPart, initDeck, pulsePart, renderDeck } from "./deck.js";
import { initBackLinks, initLightbox } from "./detail.js";
import { updateMushroom } from "./mushroom.js";
import { initOffline, initOfflineDetail, loadOfflinePack, searchOffline } from "./offline.js";
import { renderActiveFilters, renderDiameter, renderMessage, renderOptionStates, renderResults, syncDock } from "./results.js";
import { paramsForRequest, readStateFromURL, resetState, setDiameter, state, syncURL, toggleTagInState } from "./state.js";
import { fetchJSON, readSession, scrollBehavior, writeSession } from "./util.js";

initOffline();
initBackLinks();
initLightbox();
initOfflineDetail();

const results = document.querySelector("#species-results");
if (results) initFinder();

// --- Startseite: Pilz-Lab, Filter und Treffer.
function initFinder() {
  const searchInput = document.querySelector("#species-search");
  const minSlider = document.querySelector("#diameter-min");
  const maxSlider = document.querySelector("#diameter-max");
  let updateTimer;
  let requestSequence = 0;
  let lastRenderKey = "";
  let pendingHistoryMode = "push";

  readStateFromURL();
  // Beim ersten Öffnen ohne Filter: Hutpilz (so ist die Zeichnung ohnehin
  // gestartet) und der aktuelle Monat – im Wald sucht man, was jetzt wächst.
  // Beides ist als Chip sichtbar und mit einem Tipp weg.
  if (!location.search && !readSession("pilz-season-default")) {
    state.tags.add(DEFAULT_GROUP);
    state.tags.add(currentMonthTag);
    writeSession("pilz-season-default", "1");
  }

  // Alles, was nur vom lokalen Zustand abhängt, sofort neu zeichnen.
  function refresh() {
    updateMushroom(state.tags);
    renderDeck(state.tags);
    renderDiameter();
    renderOptionStates();
  }

  // Treffer laden: online über die API, sonst aus dem Offline-Paket.
  async function updateResults() {
    const requestId = ++requestSequence;
    const historyMode = pendingHistoryMode;
    pendingHistoryMode = "replace";
    const params = paramsForRequest();
    const key = params.toString();
    const availableParams = new URLSearchParams(params);
    availableParams.delete("page");
    availableParams.delete("letter");
    let data;
    let availableTags;
    try {
      if (!navigator.onLine) throw new Error("offline");
      const availablePromise = fetchJSON(`/api/available-tags?${availableParams}`)
        .then((available) => new Set(available.tags || []))
        .catch(() => null);
      data = await fetchJSON(`/api/search?${params}`);
      availableTags = await availablePromise;
    } catch (_error) {
      const pack = await loadOfflinePack();
      if (requestId !== requestSequence) return;
      if (!pack) {
        renderMessage("Keine Verbindung. Speichere die Daten oben über „Offline speichern“, solange du Netz hast – dann funktioniert die Suche auch im Funkloch.");
        return;
      }
      ({ data, availableTags } = searchOffline(pack, params));
    }
    if (requestId !== requestSequence) return;
    state.availableTags = availableTags;
    renderOptionStates();
    if (key === lastRenderKey) return;
    const searchFocused = document.activeElement === searchInput;
    renderResults(data);
    renderActiveFilters();
    if (searchFocused) searchInput.focus({ preventScroll: true });
    syncURL(historyMode);
    lastRenderKey = key;
  }

  // Kurz entprellt, damit schnelles Tippen nur eine Anfrage auslöst.
  function scheduleUpdate(delay = 120, historyMode = "push") {
    clearTimeout(updateTimer);
    requestSequence++;
    pendingHistoryMode = historyMode;
    state.availableTags = null;
    renderOptionStates();
    updateTimer = setTimeout(updateResults, delay);
  }

  function changed(delay, historyMode) {
    refresh();
    scheduleUpdate(delay, historyMode);
  }

  function toggleTag(tag, group = "multi") {
    // Die Gruppenleiste wirkt wie Radio-Knöpfe: erneutes Antippen hebt nicht auf.
    if (group === "gruppe" && state.tags.has(tag)) return;
    const part = currentPart();
    const replaced = toggleTagInState(tag, group, part);
    refresh();
    if (replaced) pulsePart(part);
    scheduleUpdate();
  }

  function removeTag(tag) {
    state.tags.delete(tag);
    state.page = 1;
    changed(0);
  }

  function resetFilters() {
    resetState();
    state.tags.add(DEFAULT_GROUP);
    searchInput.value = "";
    closeDeck();
    changed(0);
  }

  function setDiameterFromSliders(source) {
    // Die Regler dürfen sich nicht überkreuzen.
    if (Number(minSlider.value) > Number(maxSlider.value)) {
      if (source === minSlider) maxSlider.value = minSlider.value;
      else minSlider.value = maxSlider.value;
    }
    setDiameter(minSlider.value, maxSlider.value);
    state.page = 1;
    changed(180, "replace");
  }

  // --- Ereignisse
  initDeck(() => state.tags, toggleTag);
  document.querySelectorAll("[data-trait-tag]").forEach((button) => button.addEventListener("click", () => toggleTag(button.dataset.traitTag, button.dataset.traitGroup || "multi")));
  document.querySelectorAll(".spore-option").forEach((button) => {
    button.style.setProperty("--spore-color", sporeColorHex[button.dataset.traitTag] || "#999b96");
  });

  document.querySelector("#active-filters")?.addEventListener("click", (event) => {
    const remove = event.target.closest("[data-remove-tag]");
    if (remove) removeTag(remove.dataset.removeTag);
    if (event.target.closest("[data-reset-diameter]")) { setDiameter(minSlider.min, maxSlider.max); changed(0); }
    if (event.target.closest("[data-clear-query]")) { state.q = ""; searchInput.value = ""; state.page = 1; changed(0); }
    if (event.target.closest("#reset-filters-inline")) resetFilters();
  });
  results.addEventListener("click", (event) => {
    const remove = event.target.closest("[data-remove-tag]");
    if (remove) removeTag(remove.dataset.removeTag);
  });
  document.querySelector("#reset-filters")?.addEventListener("click", resetFilters);

  document.querySelectorAll("[data-diameter]").forEach((button) => button.addEventListener("click", () => {
    const [low, high] = button.dataset.diameter.split("-");
    const active = state.min === low && state.max === high;
    setDiameter(active ? minSlider.min : low, active ? maxSlider.max : high);
    state.page = 1;
    changed(0);
  }));
  minSlider?.addEventListener("input", () => setDiameterFromSliders(minSlider));
  maxSlider?.addEventListener("input", () => setDiameterFromSliders(maxSlider));

  document.querySelector(".pagination")?.addEventListener("click", (event) => {
    const button = event.target.closest("[data-page]");
    if (!button) return;
    state.page = Math.max(1, state.page + Number(button.dataset.page));
    scheduleUpdate(0);
  });

  searchInput?.addEventListener("input", () => { state.q = searchInput.value.trim(); state.page = 1; scheduleUpdate(260, "replace"); });
  document.addEventListener("keydown", (event) => {
    if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") { event.preventDefault(); searchInput?.focus(); }
  });

  window.addEventListener("popstate", () => {
    readStateFromURL();
    if (searchInput) searchInput.value = state.q;
    changed(0, "replace");
  });

  // Trefferleiste (Handy) springt zur Ergebnisliste.
  const resultsDock = document.querySelector("#results-dock");
  window.addEventListener("scroll", syncDock, { passive: true });
  window.addEventListener("resize", syncDock);
  resultsDock?.addEventListener("click", (event) => {
    event.preventDefault();
    document.querySelector("#results").scrollIntoView({ behavior: scrollBehavior() });
  });

  document.querySelector(`[data-trait-tag="${currentMonthTag}"]`)?.classList.add("is-now");
  changed(0, "replace");
}
