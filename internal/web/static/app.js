const searchInput = document.querySelector("#species-search");
const minSlider = document.querySelector("#diameter-min");
const maxSlider = document.querySelector("#diameter-max");
const diameterValue = document.querySelector("#diameter-value");
const results = document.querySelector("#species-results");
const resultLabel = document.querySelector("#results-label");
const resultNumber = document.querySelector(".result-number");
const activeFilters = document.querySelector("#active-filters");
const pagination = document.querySelector(".pagination");
const deck = document.querySelector("#option-deck");
const deckTitle = document.querySelector("#deck-title");
const deckSections = document.querySelector("#deck-sections");
const facetLabels = new Map([...document.querySelectorAll("#facet-labels [data-tag]")].map((node) => [node.dataset.tag, node.dataset.label]));
const customLabels = { leisten: "Leisten" };

const resultsColumn = document.querySelector("#results");
const resultsDock = document.querySelector("#results-dock");
const offlineButton = document.querySelector("#offline-button");
const monthTags = Array.from({ length: 12 }, (_, index) => `monat_${index + 1}`);
const currentMonthTag = monthTags[new Date().getMonth()];
const OFFLINE_DATA_URL = "/api/offline";
const REQUEST_TIMEOUT_MS = 6000;

const colors = ["weiß", "gelb", "braun", "schwarz", "rot", "orange", "blau", "gruen", "rosa", "grau"];
const colorHex = { "weiß": "#eeeade", gelb: "#e5c64d", braun: "#8b5e3c", schwarz: "#333332", rot: "#bd4b45", orange: "#d98740", blau: "#6d94a7", gruen: "#718b62", rosa: "#d69da5", grau: "#999b96" };
const sporeColorHex = { "sp_weiß": "#f2efe4", sp_gelb: "#d9b936", sp_rot: "#bd4b45", sp_braun: "#8b5e3c", sp_violett: "#846497", sp_schwarz: "#333332" };
const partNames = { cap: "Hut", underside: "Unterseite", stem: "Stiel", flesh: "Fruchtfleisch", growth: "Untergrund und Wuchsform" };
const partChoices = {
  cap: [
    { title: "Hutfarbe", kind: "color", group: "colors", tags: colors.map((color) => color) },
    { title: "Hutform", kind: "icon", group: "shape", tags: ["spitz", "schirm", "kugel", "trichter", "f", "sonstige_form"] },
    { title: "Hutoberfläche", kind: "icon", group: "multi", tags: ["hut_gerieft", "hut_hygrophan", "flocken", "haut_schmierig"] },
  ],
  underside: [
    { title: "Unterseitenform", kind: "icon", group: "type", tags: ["lamellen", "roehren", "stachel", "leisten"] },
    { title: "Farbe der Unterseite", kind: "color", group: "colors", tags: colors.map((color) => `lr_${color}`) },
  ],
  stem: [
    { title: "Stielfarbe", kind: "color", group: "colors", tags: colors.map((color) => `stiel_${color}`) },
    { title: "Stieldetails · kombinierbar", kind: "icon", group: "multi", tags: ["ring", "netz", "knollig", "wurzel", "stiel_hohl"] },
  ],
  flesh: [
    { title: "Farbe", kind: "color", group: "colors", tags: colors.map((color) => `fl_${color}`) },
    { title: "Merkmale · kombinierbar", kind: "icon", group: "multi", tags: ["milch", "bruechig", "faserig"] },
  ],
  growth: [
    { title: "Untergrund", kind: "icon", group: "substrate", tags: ["auf_boden", "auf_wiese", "holz"] },
    { title: "Wuchsform", kind: "icon", group: "multi", tags: ["bueschelig"] },
  ],
};
const groups = {
  cap: { colors: new Set(colors), shape: new Set(["spitz", "schirm", "kugel", "trichter", "f", "sonstige_form"]) },
  underside: { colors: new Set(colors.map((color) => `lr_${color}`)), type: new Set(["lamellen", "roehren", "stachel", "leisten"]) },
  stem: { colors: new Set(colors.map((color) => `stiel_${color}`)) },
  flesh: { colors: new Set(colors.map((color) => `fl_${color}`)) },
  growth: { substrate: new Set(["auf_boden", "auf_wiese", "holz"]) },
  edibility: { edibility: new Set(["hervorragend", "gut", "schlecht", "giftig", "tot"]) },
  spore: { spore: new Set(Object.keys(sporeColorHex)) },
  saison: { saison: new Set(monthTags) },
};
const currentURL = new URL(window.location.href);
const state = {
  tags: new Set((currentURL.searchParams.get("f") || "").split(",").map((tag) => tag.trim()).filter(Boolean)),
  q: currentURL.searchParams.get("q") || "",
  min: minSlider?.value || minSlider?.min || "1",
  max: maxSlider?.value || maxSlider?.max || "25",
  page: Number(currentURL.searchParams.get("page") || 1),
  letter: currentURL.searchParams.get("letter") || "",
  availableTags: null,
};
// Beim ersten Öffnen ohne Filter zählt der aktuelle Monat: Im Wald sucht man,
// was jetzt wächst. Der Filter ist als Chip sichtbar und mit einem Tipp weg.
if (results && !currentURL.search && !readSession("pilz-season-default")) {
  state.tags.add(currentMonthTag);
  writeSession("pilz-season-default", "1");
}
let openPart = "";
let updateTimer;
let requestSequence = 0;
let lastRenderKey = "";
let pendingHistoryMode = "push";
const pulseTimers = new WeakMap();

document.querySelectorAll(".spore-option").forEach((button) => {
  button.style.setProperty("--spore-color", sporeColorHex[button.dataset.traitTag] || "#999b96");
});

function readSession(key) {
  try { return sessionStorage.getItem(key); } catch (_error) { return null; }
}

function writeSession(key, value) {
  try { sessionStorage.setItem(key, value); } catch (_error) { /* privater Modus */ }
}

function escapeHTML(value) {
  return String(value ?? "").replace(/[&<>"']/g, (char) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[char]);
}

function labelFor(tag) {
  return facetLabels.get(tag) || customLabels[tag] || tag.replaceAll("_", " ");
}

function paramsForRequest() {
  const params = new URLSearchParams();
  if (state.q) params.set("q", state.q);
  if (state.tags.size) params.set("f", [...state.tags].join(","));
  if (state.min !== minSlider.min) params.set("dmin", state.min);
  if (state.max !== maxSlider.max) params.set("dmax", state.max);
  if (state.page > 1) params.set("page", String(state.page));
  if (state.letter) params.set("letter", state.letter);
  return params;
}

function syncURL(historyMode = "replace") {
  const params = paramsForRequest();
  const nextURL = `${location.pathname}${params.size ? `?${params}` : ""}`;
  if (`${location.pathname}${location.search}` === nextURL) return;
  if (historyMode === "push") history.pushState(null, "", nextURL);
  else history.replaceState(null, "", nextURL);
}

function updateDiameterLabel() {
  if (!minSlider || !maxSlider || !diameterValue) return;
  diameterValue.textContent = minSlider.value === minSlider.min && maxSlider.value === maxSlider.max
    ? "alle Werte"
    : `${minSlider.value} – ${maxSlider.value} cm`;
  document.querySelectorAll("[data-diameter]").forEach((button) => {
    const [low, high] = button.dataset.diameter.split("-");
    const selected = minSlider.value === low && maxSlider.value === high;
    button.classList.toggle("is-selected", selected);
    button.setAttribute("aria-pressed", String(selected));
  });
}

function renderActiveFilters() {
  if (!activeFilters) return;
  const customDiameter = state.min !== minSlider.min || state.max !== maxSlider.max;
  const hasFilters = state.tags.size > 0 || state.q !== "" || customDiameter;
  activeFilters.hidden = !hasFilters;
  activeFilters.innerHTML = [
    ...[...state.tags].map((tag) => `<button class="filter-chip" type="button" data-remove-tag="${escapeHTML(tag)}">${escapeHTML(labelFor(tag))} <span aria-hidden="true">×</span></button>`),
    ...(customDiameter ? [`<button class="filter-chip" type="button" data-reset-diameter>Hut ${escapeHTML(state.min)}–${escapeHTML(state.max)} cm <span aria-hidden="true">×</span></button>`] : []),
    ...(state.q ? [`<button class="filter-chip" type="button" data-clear-query>„${escapeHTML(state.q)}“ <span aria-hidden="true">×</span></button>`] : []),
    ...(hasFilters ? ['<button class="reset-inline" type="button" id="reset-filters-inline">Zurücksetzen</button>'] : []),
  ].join("");
}

function renderCard(species) {
  const name = escapeHTML(species.name);
  const scientific = escapeHTML(species.scientific_name || "Wissenschaftlicher Name nicht erfasst");
  const edibilityClass = escapeHTML(species.edibility_class || "unknown");
  const edibilityLabel = escapeHTML(species.edibility_label || "Essbarkeit nicht erfasst");
  const image = species.image ? `<img class="card-image" src="${escapeHTML(species.image)}" alt="${name}" loading="lazy" decoding="async">` : '<span class="card-image card-image-empty" aria-hidden="true">✳</span>';
  return `<a class="species-card edibility-card-${edibilityClass}" href="/art/${escapeHTML(encodeURIComponent(species.slug))}">${image}<span class="card-copy"><strong>${name}</strong><em>${scientific}</em><span class="visually-hidden">Essbarkeit: ${edibilityLabel}. Keine Verzehrfreigabe.</span></span><span class="card-arrow" aria-hidden="true">↗</span></a>`;
}

function renderResults(data) {
  const y = window.scrollY;
  resultLabel.textContent = `${data.count} passende ${data.count === 1 ? "Art" : "Arten"}`;
  resultNumber.textContent = data.total_pages > 1 ? `Seite ${data.page} von ${data.total_pages}` : "";
  const activeMonth = monthTags.find((tag) => state.tags.has(tag));
  const monthHint = activeMonth ? ` <button class="reset-inline" type="button" data-remove-tag="${activeMonth}">Ohne Monatsfilter suchen</button>` : "";
  results.innerHTML = data.results.length
    ? data.results.map(renderCard).join("")
    : `<p class="empty-state">Keine passenden Arten gefunden. Entferne zuletzt gesetzte Merkmale.${monthHint}</p>`;
  window.scrollTo(0, y);
  updateDock(data);
  pagination.innerHTML = `${data.page > 1 ? '<button type="button" data-page="-1">← Zurück</button>' : "<span></span>"}${data.total_pages > 1 ? `<span>Seite ${data.page} / ${data.total_pages}</span>` : ""}${data.page < data.total_pages ? '<button type="button" data-page="1">Weiter →</button>' : ""}`;
}

async function updateResults() {
  const requestId = ++requestSequence;
  const historyMode = pendingHistoryMode;
  pendingHistoryMode = "replace";
  const params = paramsForRequest();
  const key = params.toString();
  const url = `/api/search${params.size ? `?${params}` : ""}`;
  state.availableTags = null;
  applyExhausted();
  const availableParams = new URLSearchParams(params);
  availableParams.delete("page");
  availableParams.delete("letter");
  const availableURL = `/api/available-tags${availableParams.size ? `?${availableParams}` : ""}`;
  let data;
  let availableTags;
  try {
    if (!navigator.onLine) throw new Error("offline");
    const availablePromise = fetchJSON(availableURL)
      .then((available) => new Set(available.tags || []))
      .catch(() => null);
    data = await fetchJSON(url);
    availableTags = await availablePromise;
  } catch (_error) {
    const pack = await loadOfflinePack();
    if (requestId !== requestSequence) return;
    if (!pack) {
      results.innerHTML = '<p class="empty-state">Keine Verbindung. Speichere die Daten oben über „Offline speichern“, solange du Netz hast – dann funktioniert die Suche auch im Funkloch.</p>';
      return;
    }
    ({ data, availableTags } = searchOffline(pack, params));
  }
  try {
    if (requestId !== requestSequence) return;
    state.availableTags = availableTags;
    applyExhausted();
    if (key === lastRenderKey) return;
    const searchFocused = document.activeElement === searchInput;
    renderResults(data);
    renderActiveFilters();
    if (searchFocused && searchInput) searchInput.focus({ preventScroll: true });
    syncURL(historyMode);
    lastRenderKey = key;
  } catch (error) {
    console.error(error);
  }
}

async function fetchJSON(url) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
  try {
    const response = await fetch(url, { headers: { Accept: "application/json" }, signal: controller.signal });
    if (!response.ok) throw new Error(`HTTP ${response.status}`);
    return await response.json();
  } finally {
    clearTimeout(timer);
  }
}

function scheduleUpdate(delay = 120, historyMode = "push") {
  clearTimeout(updateTimer);
  requestSequence++;
  pendingHistoryMode = historyMode;
  state.availableTags = null;
  applyExhausted();
  updateTimer = setTimeout(updateResults, delay);
}

function applyExhausted() {
  const hasAvailability = state.availableTags instanceof Set;
  document.querySelectorAll("[data-trait-tag], #option-deck [data-choice-tag]").forEach((option) => {
    const tag = option.dataset.traitTag || option.dataset.choiceTag;
    const exhausted = hasAvailability && !state.tags.has(tag) && !state.availableTags.has(tag);
    option.classList.toggle("is-exhausted", exhausted);
  });
}

function colorFor(tag, prefix = "") {
  const name = prefix ? tag.replace(prefix, "") : tag;
  return colorHex[name] || "#b9b2a2";
}

function activeFrom(set) {
  return [...set].find((tag) => state.tags.has(tag));
}

function updateMushroom() {
  if (!document.querySelector(".cap-fill")) return;
  const capColor = activeFrom(new Set(colors));
  const undersideColor = activeFrom(new Set(colors.map((color) => `lr_${color}`)));
  const stemColor = activeFrom(new Set(colors.map((color) => `stiel_${color}`)));
  const fleshColor = activeFrom(new Set(colors.map((color) => `fl_${color}`)));
  document.querySelector(".cap-fill").style.fill = capColor ? colorFor(capColor) : "url(#capShade)";
  document.querySelectorAll(".stem-fill, .stem-tint").forEach((node) => { node.style.fill = stemColor ? colorFor(stemColor, "stiel_") : ""; });
  const underside = document.querySelector('[data-part="underside"]');
  underside.style.setProperty("--underside", undersideColor ? colorFor(undersideColor, "lr_") : "#e8dfcb");
  document.querySelector(".flesh-core").style.fill = fleshColor ? colorFor(fleshColor, "fl_") : "#eee7d7";
  const shapePaths = {
    spitz: "M30 132C50 75 81 38 120 28c39 10 70 47 90 104-45 15-135 15-180 0Z",
    schirm: "M30 132C44 78 71 62 120 61c49 1 76 17 90 71-45 15-135 15-180 0Z",
    kugel: "M30 132C22 92 52 42 120 40c68 2 98 52 90 92-45 15-135 15-180 0Z",
    trichter: "M30 132C22 120 18 96 16 76c50-12 158-12 208 0-2 20-6 44-14 56-45 15-135 15-180 0Z",
    f: "M30 132C27 106 40 84 60 70q10-12 26-10 14-8 34-4 20-4 34 4 16-2 26 10c20 14 32 36 30 62-45 15-135 15-180 0Z",
    sonstige_form: "M30 132C36 82 70 43 120 39c50 4 84 43 90 93-45 15-135 15-180 0Z",
  };

  const selectedShape = activeFrom(new Set(["spitz", "schirm", "kugel", "trichter", "f", "sonstige_form"]));
  const capD = shapePaths[selectedShape] || shapePaths.sonstige_form;
  const capFill = capColor ? colorFor(capColor) : "url(#capShade)";
  const stemFill = stemColor ? colorFor(stemColor, "stiel_") : "url(#stemShade)";
  document.querySelector("#cap-shape").setAttribute("d", capD);
  document.querySelector(".cap-funnel").classList.toggle("is-visible", selectedShape === "trichter");
  document.querySelector(".surface-grooves").classList.toggle("is-visible", state.tags.has("hut_gerieft"));
  document.querySelector(".surface-speckles").classList.toggle("is-visible", state.tags.has("flocken"));
  document.querySelector(".surface-hygro").classList.toggle("is-visible", state.tags.has("hut_hygrophan"));
  document.querySelector(".surface-gloss").classList.toggle("is-visible", state.tags.has("haut_schmierig"));
  underside.dataset.underside = activeFrom(new Set(["lamellen", "roehren", "stachel", "leisten"])) || "lamellen";
  document.querySelector(".stem-ridges").classList.toggle("is-visible", underside.dataset.underside === "leisten");
  for (const [tag, selector] of [["ring", ".stem-ring"], ["netz", ".stem-net"], ["knollig", ".stem-bulb"], ["wurzel", ".stem-roots"], ["stiel_hohl", ".stem-hollow"]]) {
    document.querySelector(selector).classList.toggle("is-visible", state.tags.has(tag));
  }
  document.querySelector(".flesh-milk").classList.toggle("is-visible", state.tags.has("milch"));
  document.querySelector(".flesh-crack").classList.toggle("is-visible", state.tags.has("bruechig"));
  document.querySelector(".flesh-fiber-pattern").classList.toggle("is-visible", state.tags.has("faserig"));
  const selectedSurface = activeFrom(new Set(["auf_boden", "auf_wiese", "holz"]));
  const substrates = { auf_boden: "erde", holz: "holz", auf_wiese: "wiese", bueschelig: "bueschel" };
  for (const [tag, substrate] of Object.entries(substrates)) {
    const element = document.querySelector(`.substrate[data-substrate="${substrate}"]`);
    const visible = tag === "bueschelig" ? state.tags.has(tag) : selectedSurface === tag;
    element.classList.toggle("is-visible", visible);
  }
  document.querySelectorAll(".mini-cap").forEach((p) => { p.setAttribute("d", capD); p.setAttribute("fill", capFill); });
  document.querySelectorAll(".mini-stem").forEach((p) => { p.setAttribute("fill", stemFill); });
  document.querySelectorAll(".mushroom-part").forEach((part) => {
    const name = partNames[part.dataset.part];
    const selected = [...state.tags].filter((tag) => (partChoices[part.dataset.part] || []).some((section) => section.tags.includes(tag))).map(labelFor);
    const title = selected.length ? `${name}: ${selected.join(", ")}` : `${name} wählen`;
    part.setAttribute("title", title);
    part.setAttribute("aria-label", title);
    part.classList.toggle("has-selection", selected.length > 0);
  });
  document.querySelectorAll("[data-trait-tag]").forEach((button) => button.classList.toggle("is-selected", state.tags.has(button.dataset.traitTag)));
  applyExhausted();
}

function toggleTag(tag, group = "multi") {
  const previousSelection = group === "multi" ? null : activeFrom(
    new Set((partChoices[openPart] || []).filter((section) => section.group === group).flatMap((section) => section.tags)),
  );
  if (state.tags.has(tag)) state.tags.delete(tag);
  else {
    if (group !== "multi") {
      // Nur innerhalb des aktuellen Parts wischen (Hutfarbe ersetzt keine Lamellenfarbe)
      const partGroup = openPart && groups[openPart] && groups[openPart][group] ? groups[openPart][group] : null;
      for (const existing of [...state.tags]) {
        const inGroup = partGroup
          ? partGroup.has(existing)
          : Object.entries(groups).some(([, map]) => map[group]?.has(existing));
        if (inGroup) state.tags.delete(existing);
      }
    }
    state.tags.add(tag);
  }
  state.page = 1;
  updateMushroom();
  if (openPart) renderDeck(openPart);
  if (group !== "multi" && previousSelection !== tag && state.tags.has(tag)) pulsePart(openPart);
  scheduleUpdate();
}

function renderDeck(part) {
  const sections = partChoices[part] || [];
  if (!sections.length) return;
  const scrollPositions = new Map([...deckSections.querySelectorAll(".deck-carousel")].map((carousel) => [carousel.dataset.sectionIndex, carousel.scrollLeft]));
  deckTitle.textContent = `${partNames[part]} wählen`;
  deckSections.innerHTML = sections.map((section, index) => {
    const options = section.tags.map((tag) => {
      const selected = state.tags.has(tag);
      const prefix = part === "cap" ? "" : part === "stem" ? "stiel_" : part === "flesh" ? "fl_" : "lr_";
      const swatchColor = section.kind === "color" ? colorFor(tag, prefix) : "";
      const kindClass = section.kind === "color" ? "color-option" : "icon-choice";
      const specialIcon = part === "growth" ? substrateIconFor(tag) : part === "underside" ? undersideIconFor(tag) : "";
      const icon = specialIcon || (section.kind === "color" ? `<span class="swatch" style="--swatch:${swatchColor}"></span>` : `<span class="choice-icon" aria-hidden="true">${iconFor(tag)}</span>`);
      return `<button type="button" class="deck-option ${kindClass}${selected ? " is-selected" : ""}" data-choice-tag="${escapeHTML(tag)}" data-choice-group="${escapeHTML(section.group)}" aria-pressed="${selected}" title="${escapeHTML(labelFor(tag))}">${icon}<span>${escapeHTML(labelFor(tag))}</span></button>`;
    }).join("");
    return `<section class="deck-section"><h4>${escapeHTML(section.title)}</h4><div class="deck-carousel${section.kind === "color" ? " color-options" : ""}" data-section-index="${index}" role="group" aria-label="${escapeHTML(section.title)}">${options}</div></section>`;
  }).join("");
  applyExhausted();
  deckSections.querySelectorAll(".deck-carousel").forEach((carousel) => {
    const priorScroll = scrollPositions.get(carousel.dataset.sectionIndex);
    if (priorScroll !== undefined) carousel.scrollLeft = priorScroll;
    else {
      const selected = carousel.querySelector(".deck-option.is-selected");
      if (selected) requestAnimationFrame(() => centerDeckOption(selected, carousel));
    }
  });
}

function centerDeckOption(option, carousel) {
  const optionRect = option.getBoundingClientRect();
  const carouselRect = carousel.getBoundingClientRect();
  const target = carousel.scrollLeft + optionRect.left - carouselRect.left + optionRect.width / 2 - carousel.clientWidth / 2;
  carousel.scrollTo({ left: Math.max(0, target), behavior: window.matchMedia("(prefers-reduced-motion: reduce)").matches ? "auto" : "smooth" });
}

function iconFor(tag) {
  return ({ spitz: "△", schirm: "⌒", kugel: "◠", trichter: "▽", f: "◖", sonstige_form: "⌒", lamellen: "≋", roehren: "○", stachel: "⋮", hut_gerieft: "╱", hut_hygrophan: "◒", flocken: "⁘", haut_schmierig: "◌", ring: "◉", netz: "▦", knollig: "⬭", wurzel: "♧", stiel_hohl: "∥", milch: "◉", bruechig: "⌁", faserig: "╱", bueschelig: "♧", holz: "▰", auf_wiese: "♣" })[tag] || "•";
}

function undersideIconFor(tag) {
  const cap = '<path class="option-cap" d="M2 9C4 4 9 2 14 2s10 2 12 7Z"/>';
  const band = '<path class="option-underside" d="M2 9c6 2 18 2 24 0l-2 4c-5 2-15 2-20 0Z"/>';
  const icons = {
    lamellen: `${cap}${band}<path d="M5 10.2l1.3 2.8M8 10.8l.8 2.7M11 11.1l.4 2.7M14 11.2v2.8M17 11.1l-.4 2.7M20 10.8l-.8 2.7M23 10.2l-1.3 2.8"/>`,
    roehren: `${cap}${band}<path class="option-pores" d="M6 10.9h0m3 .6h0m3 .3h0m4 0h0m3-.3h0m3-.6h0M7.5 12.6h0m3 .4h0m3 .1h0m3-.1h0m3-.4h0"/>`,
    stachel: `${cap}<path class="option-underside" d="M2 9c6 2 18 2 24 0l-1.6 2.6c-5 1.6-15.8 1.6-20.8 0Z"/><path class="option-underside" d="M4 11.5l1 3 1-3ZM7 12.1l1 3.6 1-3.6ZM10 12.5l1 4.2 1-4.2ZM13 12.6l1 4.5 1-4.5ZM16 12.5l1 4.2 1-4.2ZM19 12.1l1 3.6 1-3.6ZM22 11.5l1 3 1-3Z"/>`,
    leisten: `${cap}<path class="option-underside" d="M11.2 14h5.6l-.6 3.6h-4.4Z"/>${band}<path class="ridge-fan" d="M6.5 13.3V12.1l-1.4-1.6M6.5 12.1l1.4-1.6M11.5 14.0V12.8l-1.4-1.7M11.5 12.8l1.4-1.7M16.5 14.0V12.8l-1.4-1.7M16.5 12.8l1.4-1.7M21.5 13.3V12.1l-1.4-1.6M21.5 12.1l1.4-1.6M12.6 14.6l.2 2.4m1.2-2.4v2.6m1.4-2.6-.2 2.4"/>`,
  };
  return icons[tag] ? `<svg class="substrate-option-icon underside-option-icon" viewBox="0 0 28 18" aria-hidden="true">${icons[tag]}</svg>` : "";
}

function substrateIconFor(tag) {
  const icons = {
    bueschelig: '<svg class="substrate-option-icon" viewBox="0 0 28 18" aria-hidden="true"><path class="option-bed" d="M2 15c6-2 18-2 24 0v3H2Z"/><path class="cluster-cap" d="M3 10c.5-3 2-5 4-5s3.5 2 4 5c-2 .7-6 .7-8 0Zm8 1c.5-4 2-7 4-7s3.5 3 4 7c-2 .7-6 .7-8 0Zm8-1c.5-3 2-5 4-5s3.5 2 4 5c-2 .7-6 .7-8 0Z"/><path d="M6 10v5m7-4v4m7-5v5"/></svg>',
    holz: '<svg class="substrate-option-icon substrate-option-wood" viewBox="0 0 28 18" aria-hidden="true"><path class="option-bed" d="M2 15c6-2 18-2 24 0v3H2Z"/><rect x="2" y="8" width="24" height="7" rx="2"/><path d="M5 10h13m-8 2h11m-2-4c4 0 5 5 1 6"/></svg>',
    auf_boden: '<svg class="substrate-option-icon substrate-option-earth" viewBox="0 0 28 18" aria-hidden="true"><path class="option-bed" d="M2 15c6-2 18-2 24 0v3H2Z"/><path class="option-earth-surface" d="M2 12c6-2 18-2 24 0v4H2Z"/><path d="M7 13h2m5-2h2m4 3h2"/></svg>',
    auf_wiese: '<svg class="substrate-option-icon substrate-option-grass" viewBox="0 0 28 18" aria-hidden="true"><path class="option-bed" d="M2 15c6-2 18-2 24 0v3H2Z"/><path class="option-grass-turf" d="M2 12c6-2 18-2 24 0v4H2Z"/><path d="M4 13 6 7m3 6V5m3 8 2-7m3 7 1-5m3 5 2-7m3 7 1-5"/><circle cx="15" cy="5" r="1.5"/></svg>',
  };
  return icons[tag] || "";
}

function openDeck(part) {
  openPart = part;
  renderDeck(part);
  deck.hidden = false;
  document.querySelectorAll(".mushroom-part").forEach((node) => node.classList.toggle("is-open", node.dataset.part === part));
}

function closeDeck() {
  deck.hidden = true;
  openPart = "";
  document.querySelectorAll(".mushroom-part").forEach((node) => node.classList.remove("is-open"));
}

function pulsePart(partName) {
  const part = document.querySelector(`.mushroom-part[data-part="${partName}"]`);
  if (!part) return;
  clearTimeout(pulseTimers.get(part));
  part.classList.remove("is-cycled");
  void part.getBoundingClientRect();
  part.classList.add("is-cycled");
  pulseTimers.set(part, setTimeout(() => part.classList.remove("is-cycled"), 400));
}

document.querySelectorAll(".mushroom-part").forEach((part) => {
  part.addEventListener("click", () => openDeck(part.dataset.part));
  part.addEventListener("keydown", (event) => {
    if (event.key === "Enter" || event.key === " ") { event.preventDefault(); openDeck(part.dataset.part); }
  });
});
document.querySelector("#deck-close")?.addEventListener("click", closeDeck);
deckSections?.addEventListener("click", (event) => {
  const choice = event.target.closest("[data-choice-tag]");
  if (choice) toggleTag(choice.dataset.choiceTag, choice.dataset.choiceGroup);
});
document.querySelectorAll("[data-trait-tag]").forEach((button) => button.addEventListener("click", () => toggleTag(button.dataset.traitTag, button.dataset.traitGroup || "multi")));
document.addEventListener("click", (event) => {
  if (!deck || deck.hidden) return;
  const inside = event.composedPath().some((node) => node === deck || (node.nodeType === 1 && node.classList && node.classList.contains("mushroom-part")));
  if (inside) return;
  closeDeck();
});
document.addEventListener("keydown", (event) => {
  if (!deck || deck.hidden) return;
  if (event.key === "Escape") closeDeck();
});

activeFilters?.addEventListener("click", (event) => {
  const remove = event.target.closest("[data-remove-tag]");
  if (remove) { state.tags.delete(remove.dataset.removeTag); state.page = 1; updateMushroom(); scheduleUpdate(0); }
  if (event.target.closest("[data-reset-diameter]")) { minSlider.value = minSlider.min; maxSlider.value = maxSlider.max; state.min = minSlider.value; state.max = maxSlider.value; updateDiameterLabel(); scheduleUpdate(0); }
  if (event.target.closest("[data-clear-query]")) { state.q = ""; searchInput.value = ""; state.page = 1; scheduleUpdate(0); }
  if (event.target.closest("#reset-filters-inline")) resetFilters();
});
results?.addEventListener("click", (event) => {
  const remove = event.target.closest("[data-remove-tag]");
  if (remove) { state.tags.delete(remove.dataset.removeTag); state.page = 1; updateMushroom(); scheduleUpdate(0); }
});
document.querySelectorAll("[data-diameter]").forEach((button) => button.addEventListener("click", () => {
  const [low, high] = button.dataset.diameter.split("-");
  const active = minSlider.value === low && maxSlider.value === high;
  minSlider.value = active ? minSlider.min : low;
  maxSlider.value = active ? maxSlider.max : high;
  state.min = minSlider.value; state.max = maxSlider.value; state.page = 1;
  updateDiameterLabel(); scheduleUpdate(0);
}));
pagination?.addEventListener("click", (event) => {
  const button = event.target.closest("[data-page]");
  if (!button) return;
  state.page = Math.max(1, state.page + Number(button.dataset.page));
  scheduleUpdate(0);
});

function resetFilters() {
  state.tags.clear(); state.q = ""; state.page = 1; state.letter = "";
  searchInput.value = ""; minSlider.value = minSlider.min; maxSlider.value = maxSlider.max;
  state.min = minSlider.value; state.max = maxSlider.value;
  updateDiameterLabel(); updateMushroom(); closeDeck(); scheduleUpdate(0);
}
document.querySelector("#reset-filters")?.addEventListener("click", resetFilters);

searchInput?.addEventListener("input", () => { state.q = searchInput.value.trim(); state.page = 1; scheduleUpdate(260, "replace"); });
minSlider?.addEventListener("input", () => {
  if (Number(minSlider.value) > Number(maxSlider.value)) maxSlider.value = minSlider.value;
  state.min = minSlider.value; state.max = maxSlider.value; state.page = 1; updateDiameterLabel(); scheduleUpdate(180, "replace");
});
maxSlider?.addEventListener("input", () => {
  if (Number(maxSlider.value) < Number(minSlider.value)) minSlider.value = maxSlider.value;
  state.min = minSlider.value; state.max = maxSlider.value; state.page = 1; updateDiameterLabel(); scheduleUpdate(180, "replace");
});

document.addEventListener("keydown", (event) => {
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") { event.preventDefault(); searchInput?.focus(); }
});

window.addEventListener("popstate", () => {
  if (!minSlider || !maxSlider) return;
  const params = new URLSearchParams(location.search);
  state.tags = new Set((params.get("f") || "").split(",").filter(Boolean));
  state.q = params.get("q") || "";
  state.page = Math.max(1, Number(params.get("page") || 1) || 1);
  state.letter = params.get("letter") || "";
  if (searchInput) searchInput.value = state.q;
  minSlider.value = params.get("dmin") || minSlider.min;
  maxSlider.value = params.get("dmax") || maxSlider.max;
  if (Number(minSlider.value) > Number(maxSlider.value)) [minSlider.value, maxSlider.value] = [maxSlider.value, minSlider.value];
  state.min = minSlider.value; state.max = maxSlider.value;
  updateDiameterLabel(); updateMushroom(); scheduleUpdate(0, "replace");
});

if (results) {
  document.querySelector(`[data-trait-tag="${currentMonthTag}"]`)?.classList.add("is-now");
  updateDiameterLabel();
  updateMushroom();
  scheduleUpdate(0, "replace");
  window.addEventListener("scroll", syncDock, { passive: true });
  window.addEventListener("resize", syncDock);
  resultsDock?.addEventListener("click", (event) => {
    event.preventDefault();
    resultsColumn.scrollIntoView({ behavior: window.matchMedia("(prefers-reduced-motion: reduce)").matches ? "auto" : "smooth" });
  });
}

const lightbox = document.querySelector("#lightbox");
if (lightbox) {
  const imageButtons = [...document.querySelectorAll(".image-open")];
  const lightboxImage = lightbox.querySelector(".lightbox-image");
  const counter = lightbox.querySelector(".lightbox-count");
  let currentImage = 0;
  let returnFocus;
  function showImage(index) {
    currentImage = (index + imageButtons.length) % imageButtons.length;
    const source = imageButtons[currentImage].querySelector("img");
    lightboxImage.src = source.currentSrc || source.src; lightboxImage.alt = source.alt;
    counter.textContent = `${currentImage + 1} / ${imageButtons.length}`;
  }
  function openLightbox(index) {
    returnFocus = document.activeElement; showImage(index); lightbox.classList.add("is-open");
    lightbox.setAttribute("aria-hidden", "false"); document.body.classList.add("lightbox-active"); lightbox.querySelector(".lightbox-close").focus();
  }
  function closeLightbox() {
    lightbox.classList.remove("is-open"); lightbox.setAttribute("aria-hidden", "true");
    document.body.classList.remove("lightbox-active"); returnFocus?.focus();
  }
  imageButtons.forEach((button, index) => button.addEventListener("click", () => openLightbox(index)));
  lightbox.querySelector(".lightbox-close").addEventListener("click", closeLightbox);
  lightbox.querySelector(".lightbox-prev").addEventListener("click", () => showImage(currentImage - 1));
  lightbox.querySelector(".lightbox-next").addEventListener("click", () => showImage(currentImage + 1));
  lightbox.addEventListener("click", (event) => { if (event.target === lightbox) closeLightbox(); });
  document.addEventListener("keydown", (event) => {
    if (!lightbox.classList.contains("is-open")) return;
    if (event.key === "Escape") closeLightbox();
    if (event.key === "ArrowLeft") showImage(currentImage - 1);
    if (event.key === "ArrowRight") showImage(currentImage + 1);
  });
}

// --- Trefferleiste: auf dem Handy liegt die Ergebnisliste unter dem Pilz-Lab.
// Die Leiste zeigt nach jedem Tipp sofort, wie viele Arten noch passen.
function updateDock(data) {
  if (!resultsDock) return;
  document.querySelector("#dock-number").textContent = data.count;
  document.querySelector("#dock-thumbs").innerHTML = data.results.slice(0, 3)
    .filter((species) => species.image)
    .map((species) => `<img src="${escapeHTML(species.image)}" alt="" loading="lazy" decoding="async">`).join("");
  resultsDock.classList.remove("is-updated");
  void resultsDock.offsetWidth;
  resultsDock.classList.add("is-updated");
  syncDock();
}

function syncDock() {
  if (!resultsDock || !resultsColumn) return;
  resultsDock.hidden = resultsColumn.getBoundingClientRect().top < window.innerHeight - 60;
}

// --- Offline-Paket: alle Arten kompakt im Cache, damit Suche, Filter und
// Artseiten auch ohne Netz funktionieren.
let offlinePackPromise = null;

function loadOfflinePack() {
  if (!("caches" in window)) return Promise.resolve(null);
  offlinePackPromise ||= caches.match(OFFLINE_DATA_URL)
    .then((response) => (response ? response.json() : null))
    .then((pack) => (pack ? indexPack(pack) : null))
    .catch(() => null);
  return offlinePackPromise.then((pack) => {
    if (!pack) offlinePackPromise = null;
    return pack;
  });
}

function indexPack(pack) {
  pack.bySlug = new Map(pack.species.map((species) => [species.s, species]));
  for (const species of pack.species) {
    species.tagSet = new Set(species.t);
    species.sortKey = species.n.toLowerCase().replace(/ä/g, "a").replace(/ö/g, "o").replace(/ü/g, "u").replace(/ß/g, "ss");
  }
  return pack;
}

function nameMatchRank(value, query) {
  value = String(value || "").trim().toLowerCase();
  if (value === query) return 0;
  if (value.startsWith(query)) return 1;
  if (value.includes(query)) return 2;
  return 3;
}

function offlineSearchRank(species, query) {
  let best = Infinity;
  species.a.split(",").forEach((name, index) => {
    const rank = nameMatchRank(name, query);
    if (rank < 3) best = Math.min(best, rank + (index > 0 ? 3 : 0));
  });
  const scientific = nameMatchRank(species.w, query);
  if (scientific < 3) best = Math.min(best, scientific + 6);
  for (const synonym of species.y || []) {
    const rank = nameMatchRank(synonym, query);
    if (rank < 3) best = Math.min(best, rank + 9);
  }
  return best;
}

function relevanceCompare(left, right) {
  const photoBucket = (species) => Math.floor(Math.min(species.k || 0, 40) / 10);
  return left.p - right.p || (right.c || 0) - (left.c || 0) || photoBucket(right) - photoBucket(left) || left.sortKey.localeCompare(right.sortKey);
}

// Spiegelt catalog.Filter und catalog.AvailableTags für den Offline-Betrieb.
function searchOffline(pack, params) {
  const query = (params.get("q") || "").trim().toLowerCase();
  const tags = (params.get("f") || "").split(",").filter(Boolean);
  const min = Number(params.get("dmin") || minSlider.min);
  const max = Number(params.get("dmax") || maxSlider.max);
  const page = Math.max(1, Number(params.get("page") || 1));
  const perPage = 50;
  const matches = pack.species.filter((species) => {
    if (query && offlineSearchRank(species, query) === Infinity) return false;
    const [low, high] = species.d;
    if (low && high && (low > max || high < min)) return false;
    return tags.every((tag) => species.tagSet.has(tag));
  });
  matches.sort(query ? (left, right) => offlineSearchRank(left, query) - offlineSearchRank(right, query) || relevanceCompare(left, right) : relevanceCompare);
  const availableTags = new Set();
  matches.forEach((species) => species.t.forEach((tag) => availableTags.add(tag)));
  const totalPages = Math.ceil(matches.length / perPage);
  const data = {
    count: matches.length, page, total_pages: totalPages, per_page: perPage,
    results: matches.slice((page - 1) * perPage, page * perPage).map((species) => ({
      slug: species.s, name: species.n, scientific_name: species.w, edibility_class: species.e, edibility_label: species.l, image: species.i || "",
    })),
  };
  return { data, availableTags };
}

// --- Offline speichern: Artdaten plus Fotos der wichtigsten Arten.
const PACK_MEDIA_CACHE = "pilz-media-pack";

function setOfflineButton(text, stateName) {
  if (!offlineButton) return;
  offlineButton.querySelector(".offline-text").textContent = text;
  offlineButton.dataset.state = stateName;
}

async function refreshOfflineButton() {
  if (!offlineButton || !("caches" in window) || !("serviceWorker" in navigator)) return;
  offlineButton.hidden = false;
  const stored = await caches.match(OFFLINE_DATA_URL);
  setOfflineButton(stored ? "Offline bereit" : "Offline speichern", stored ? "ready" : "idle");
}

async function downloadOfflinePack() {
  setOfflineButton("Lade Artdaten …", "busy");
  const response = await fetch(OFFLINE_DATA_URL, { cache: "no-store" });
  if (!response.ok) throw new Error("Offline-Paket nicht verfügbar");
  const dataCache = await caches.open("pilz-data");
  await dataCache.put(OFFLINE_DATA_URL, response.clone());
  const pack = indexPack(await response.json());
  offlinePackPromise = Promise.resolve(pack);
  const shell = await caches.open("pilz-shell-v1");
  await shell.addAll(["/", "/offline", "/static/app.css", "/static/app.js"]);
  const images = pack.species.filter((species) => species.p === 1 && species.i).map((species) => species.i);
  const mediaCache = await caches.open(PACK_MEDIA_CACHE);
  let done = 0;
  const queue = [...images];
  async function worker() {
    while (queue.length) {
      const image = queue.shift();
      try {
        if (!(await mediaCache.match(image))) {
          const imageResponse = await fetch(image);
          if (imageResponse.ok) await mediaCache.put(image, imageResponse);
        }
      } catch (_error) { /* einzelnes Bild fehlt – weiter */ }
      done++;
      if (done % 10 === 0 || done === images.length) setOfflineButton(`Fotos ${done}/${images.length}`, "busy");
    }
  }
  await Promise.all(Array.from({ length: 6 }, worker));
}

offlineButton?.addEventListener("click", async () => {
  if (offlineButton.dataset.state === "busy") return;
  const ready = offlineButton.dataset.state === "ready";
  const question = ready
    ? "Offline-Daten sind gespeichert. Jetzt aktualisieren?"
    : "Für die Suche ohne Netz werden alle Artdaten und die Fotos der rund 600 bekanntesten Arten gespeichert (ca. 24 MB Download, ca. 35 MB Speicher). Am besten im WLAN laden. Jetzt speichern?";
  if (!window.confirm(question)) return;
  try {
    await downloadOfflinePack();
    setOfflineButton("Offline bereit", "ready");
  } catch (_error) {
    setOfflineButton("Fehler – erneut versuchen", "idle");
  }
});

if ("serviceWorker" in navigator) {
  window.addEventListener("load", () => {
    navigator.serviceWorker.register("/sw.js").catch(() => {});
    refreshOfflineButton();
  });
}

// --- Artseite: "Zurück" führt zu den eben gefilterten Treffern.
document.querySelectorAll("[data-back-link]").forEach((link) => link.addEventListener("click", (event) => {
  if (document.referrer && new URL(document.referrer).origin === location.origin && history.length > 1) {
    event.preventDefault();
    history.back();
  }
}));

// --- Offline-Artseite: rendert eine Art aus dem Offline-Paket.
const offlineDetail = document.querySelector("#offline-detail");
if (offlineDetail) {
  const slug = decodeURIComponent(location.pathname.replace(/^\/art\//, ""));
  loadOfflinePack().then((pack) => {
    const species = pack?.bySlug.get(slug);
    if (!species) return;
    document.title = `${species.n} – Pilzdatenbank (offline)`;
    document.querySelector("#offline-message")?.remove();
    offlineDetail.insertAdjacentHTML("beforeend", renderOfflineSpecies(pack, species));
  });
}

// Offline fehlen ungespeicherte Fotos; ist die Leiste danach leer, verschwindet sie ganz.
window.dropOfflinePhoto = (image) => {
  const section = image.closest(".photo-strip-section");
  image.parentNode.remove();
  if (section && !section.querySelector(".strip-photo")) section.remove();
};

function renderTraitList(traits) {
  return `<dl class="traits-grid">${traits.map(([name, value]) => `<div class="trait-card"><dt>${escapeHTML(name)}</dt><dd>${escapeHTML(value)}</dd></div>`).join("")}</dl>`;
}

function renderOfflineSpecies(pack, species) {
  const dangerRank = { deadly: 0, poisonous: 1 };
  const related = (species.r || []).map((slug) => pack.bySlug.get(slug)).filter(Boolean)
    .sort((left, right) => (dangerRank[left.e] ?? 2) - (dangerRank[right.e] ?? 2));
  const dangerous = related.filter((other) => other.e === "deadly" || other.e === "poisonous");
  const images = species.x || [];
  const warning = species.e === "deadly" || species.e === "poisonous";
  return `
    <section class="detail-hero">
      <div class="detail-hero-copy"><p class="eyebrow">${escapeHTML(species.g || "PILZPORTRÄT")}</p><h1 class="detail-title">${escapeHTML(species.n)}</h1>${species.w ? `<p class="scientific-name">${escapeHTML(species.w)}</p>` : ""}</div>
      ${images[0] ? `<figure class="detail-hero-photo"><img src="${escapeHTML(images[0])}" alt="${escapeHTML(species.n)}" onerror="this.parentNode.remove()"></figure>` : ""}
    </section>
    ${images.length > 1 ? `<section class="photo-strip-section"><h2 class="strip-heading">Bilder <span class="strip-hint">(nur gespeicherte sichtbar)</span></h2><div class="photo-strip">${images.slice(1).map((image) => `<span class="strip-photo"><img src="${escapeHTML(image)}" alt="" loading="lazy" onerror="dropOfflinePhoto(this)"></span>`).join("")}</div></section>` : ""}
    ${dangerous.length ? `<section class="danger-banner"><strong>⚠ Verwechslungsgefahr mit giftigen Arten</strong><p>${dangerous.map((other) => `<a href="/art/${encodeURIComponent(other.s)}">${escapeHTML(other.n)}</a> <span class="danger-label">(${escapeHTML(other.l)})</span>`).join(", ")}</p></section>` : ""}
    ${related.length ? `<section class="detail-section related-section"><div class="detail-section-heading"><div><p class="eyebrow">ÄHNLICHE ARTEN – GENAU VERGLEICHEN</p><h2>Verwechslungspartner</h2></div></div><div class="species-grid related-strip">${related.map((other) => renderCard({ slug: other.s, name: other.n, scientific_name: other.w, edibility_class: other.e, edibility_label: other.l, image: other.i })).join("")}</div></section>` : ""}
    <section class="edibility-banner edibility-${escapeHTML(species.e)}"><span class="edibility-badge edibility-${escapeHTML(species.e)}">${escapeHTML(species.l)}</span><p>${warning ? "Giftiger Pilz — nicht verzehren." : "Nur die Angabe der Datenquelle – keine Verzehrfreigabe."}</p></section>
    ${species.m?.length ? `<section class="detail-section"><div class="detail-section-heading"><div><p class="eyebrow">ERKENNUNGSMERKMALE</p><h2>Merkmale</h2></div></div>${renderTraitList(species.m)}</section>` : ""}
    ${species.z?.length ? `<details class="supplemental-traits"><summary><span>Weitere Angaben</span></summary>${renderTraitList(species.z)}</details>` : ""}`;
}
