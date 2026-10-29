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
const deckTabs = document.querySelector("#deck-tabs");
const deckCarousel = document.querySelector("#deck-carousel");
const deckPrev = document.querySelector("#deck-prev");
const deckNext = document.querySelector("#deck-next");
const facetLabels = new Map([...document.querySelectorAll("#facet-labels [data-tag]")].map((node) => [node.dataset.tag, node.dataset.label]));

const colors = ["weiß", "gelb", "braun", "schwarz", "rot", "orange", "blau", "gruen", "rosa", "grau"];
const colorHex = { "weiß": "#eeeade", gelb: "#e5c64d", braun: "#8b5e3c", schwarz: "#333332", rot: "#bd4b45", orange: "#d98740", blau: "#6d94a7", gruen: "#718b62", rosa: "#d69da5", grau: "#999b96" };
const partNames = { cap: "Hut", underside: "Unterseite", stem: "Stiel", flesh: "Fruchtfleisch", growth: "Untergrund" };
const partChoices = {
  cap: [
    { title: "Hutfarbe", kind: "color", group: "colors", tags: colors.map((color) => color) },
    { title: "Hutform", kind: "icon", group: "shape", tags: ["spitz", "schirm", "kugel", "trichter", "f", "sonstige_form"] },
    { title: "Hutoberfläche", kind: "icon", group: "multi", tags: ["hut_gerieft", "hut_hygrophan", "flocken", "haut_schmierig"] },
  ],
  underside: [
    { title: "Lamellenart", kind: "icon", group: "type", tags: ["lamellen", "roehren", "stachel"] },
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
  growth: [{ title: "Untergrund · kombinierbar", kind: "icon", group: "multi", tags: ["bueschelig", "holz", "auf_boden", "auf_wiese"] }],
};
const groups = {
  cap: { colors: new Set(colors), shape: new Set(["spitz", "schirm", "kugel", "trichter", "f", "sonstige_form"]) },
  underside: { colors: new Set(colors.map((color) => `lr_${color}`)), type: new Set(["lamellen", "roehren", "stachel"]) },
  stem: { colors: new Set(colors.map((color) => `stiel_${color}`)) },
  flesh: { colors: new Set(colors.map((color) => `fl_${color}`)) },
  edibility: { edibility: new Set(["hervorragend", "gut", "schlecht", "giftig", "tot"]) },
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
let openPart = "";
let activeSectionIndex = 0;
let updateTimer;
let requestSequence = 0;
let lastRenderKey = "";
const pulseTimers = new WeakMap();

function escapeHTML(value) {
  return String(value ?? "").replace(/[&<>"']/g, (char) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[char]);
}

function labelFor(tag) {
  return facetLabels.get(tag) || tag.replaceAll("_", " ");
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

function syncURL() {
  const params = paramsForRequest();
  history.replaceState(null, "", `${location.pathname}${params.size ? `?${params}` : ""}`);
}

function updateDiameterLabel() {
  if (!minSlider || !maxSlider || !diameterValue) return;
  diameterValue.textContent = minSlider.value === minSlider.min && maxSlider.value === maxSlider.max
    ? "alle Werte"
    : `${minSlider.value} – ${maxSlider.value} cm`;
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
  const image = species.image ? `<img class="card-image" src="${escapeHTML(species.image)}" alt="${name}" loading="lazy" decoding="async">` : '<span class="card-image card-image-empty" aria-hidden="true">✳</span>';
  return `<a class="species-card" href="/art/${escapeHTML(encodeURIComponent(species.slug))}">${image}<span class="card-copy"><strong>${name}</strong><em>${scientific}</em><span class="edibility-badge edibility-${escapeHTML(species.edibility_class || "unknown")}">${escapeHTML(species.edibility_label || "Essbarkeit nicht erfasst")}</span></span><span class="card-arrow" aria-hidden="true">↗</span></a>`;
}

function renderResults(data) {
  const y = window.scrollY;
  resultLabel.textContent = `${data.count} passende ${data.count === 1 ? "Art" : "Arten"}`;
  resultNumber.textContent = data.total_pages > 1 ? `Seite ${data.page} von ${data.total_pages}` : "";
  results.innerHTML = data.results.length
    ? data.results.map(renderCard).join("")
    : '<p class="empty-state">Keine passenden Arten gefunden. Passe deine Suche oder Merkmale an.</p>';
  window.scrollTo(0, y);
  pagination.innerHTML = `${data.page > 1 ? '<button type="button" data-page="-1">← Zurück</button>' : "<span></span>"}${data.total_pages > 1 ? `<span>Seite ${data.page} / ${data.total_pages}</span>` : ""}${data.page < data.total_pages ? '<button type="button" data-page="1">Weiter →</button>' : ""}`;
}

async function updateResults() {
  const requestId = ++requestSequence;
  const params = paramsForRequest();
  const key = params.toString();
  const url = `/api/search${params.size ? `?${params}` : ""}`;
  state.availableTags = null;
  applyExhausted();
  const availableParams = new URLSearchParams(params);
  availableParams.delete("page");
  availableParams.delete("letter");
  const availableURL = `/api/available-tags${availableParams.size ? `?${availableParams}` : ""}`;
  const availablePromise = fetch(availableURL, { headers: { Accept: "application/json" } })
    .then((response) => {
      if (!response.ok) throw new Error("Verfügbare Merkmale konnten nicht geladen werden");
      return response.json();
    })
    .then((data) => new Set(data.tags || []))
    .catch(() => null);
  try {
    const response = await fetch(url, { headers: { Accept: "application/json" } });
    if (!response.ok) throw new Error("Suche fehlgeschlagen");
    const data = await response.json();
    const availableTags = await availablePromise;
    if (requestId !== requestSequence) return;
    state.availableTags = availableTags;
    applyExhausted();
    if (key === lastRenderKey) return;
    const searchFocused = document.activeElement === searchInput;
    renderResults(data);
    renderActiveFilters();
    if (searchFocused && searchInput) searchInput.focus({ preventScroll: true });
    syncURL();
    lastRenderKey = key;
  } catch (_error) {
    if (requestId !== requestSequence) return;
    window.location.href = `/${params.size ? `?${params}` : ""}`;
  }
}

function scheduleUpdate(delay = 120) {
  clearTimeout(updateTimer);
  requestSequence++;
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
  document.querySelector(".stem-fill").style.fill = stemColor ? colorFor(stemColor, "stiel_") : "url(#stemShade)";
  document.querySelector(".underside-fill").style.fill = undersideColor ? colorFor(undersideColor, "lr_") : "#e8dfcb";
  document.querySelector(".flesh-core").style.fill = fleshColor ? colorFor(fleshColor, "fl_") : "#eee7d7";
  const shapePaths = {
    spitz: "M30 132C50 75 81 38 120 28c39 10 70 47 90 104-45 15-135 15-180 0Z",
    schirm: "M30 132C44 78 71 62 120 61c49 1 76 17 90 71-45 15-135 15-180 0Z",
    kugel: "M34 132C33 79 65 42 120 40c55 2 87 39 86 92-45 15-127 15-172 0Z",
    trichter: "M30 132C42 80 68 51 120 47c52 4 78 33 90 85-41 4-66-33-90-33s-49 37-90 33Z",
    f: "M36 132C53 82 81 48 120 39c34 10 57 43 72 93-43 15-111 15-156 0Z",
    sonstige_form: "M30 132C36 82 70 43 120 39c50 4 84 43 90 93-45 15-135 15-180 0Z",
  };
  const selectedShape = activeFrom(new Set(["spitz", "schirm", "kugel", "trichter", "f", "sonstige_form"]));
  const capD = shapePaths[selectedShape] || shapePaths.sonstige_form;
  const capFill = capColor ? colorFor(capColor) : "url(#capShade)";
  const stemFill = stemColor ? colorFor(stemColor, "stiel_") : "url(#stemShade)";
  document.querySelector("#cap-shape").setAttribute("d", capD);
  document.querySelector(".surface-grooves").classList.toggle("is-visible", state.tags.has("hut_gerieft"));
  document.querySelector(".surface-speckles").classList.toggle("is-visible", state.tags.has("flocken"));
  document.querySelector("#cap-shape").classList.toggle("is-hygrophan", state.tags.has("hut_hygrophan"));
  document.querySelector("#cap-shape").classList.toggle("is-greasy", state.tags.has("haut_schmierig"));
  document.querySelector(".underside-fill").classList.toggle("pattern-tubes", state.tags.has("roehren"));
  document.querySelector(".underside-fill").classList.toggle("pattern-spines", state.tags.has("stachel"));
  document.querySelector(".gill-lines").classList.toggle("is-hidden", state.tags.has("roehren") || state.tags.has("stachel"));
  document.querySelector(".tube-dots").classList.toggle("is-visible", state.tags.has("roehren"));
  document.querySelector(".spine-lines").classList.toggle("is-visible", state.tags.has("stachel"));
  for (const [tag, selector] of [["ring", ".stem-ring"], ["netz", ".stem-net"], ["knollig", ".stem-bulb"], ["wurzel", ".stem-roots"], ["stiel_hohl", ".stem-hollow"]]) {
    document.querySelector(selector).classList.toggle("is-visible", state.tags.has(tag));
  }
  document.querySelector(".flesh-slice").classList.toggle("flesh-milky", state.tags.has("milch"));
  document.querySelector(".flesh-slice").classList.toggle("flesh-brittle", state.tags.has("bruechig"));
  document.querySelector(".flesh-fiber-pattern").classList.toggle("is-visible", state.tags.has("faserig"));
  const substrates = { holz: "holz", auf_boden: "erde", auf_wiese: "wiese", bueschelig: "bueschel" };
  for (const [tag, substrate] of Object.entries(substrates)) {
    const element = document.querySelector(`.substrate[data-substrate="${substrate}"]`);
    if (substrate === "erde") element.classList.add("is-visible");
    else element.classList.toggle("is-visible", state.tags.has(tag));
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
  activeSectionIndex = Math.min(activeSectionIndex, sections.length - 1);
  const section = sections[activeSectionIndex];
  deckTitle.textContent = `${partNames[part]} wählen`;
  deckTabs.hidden = sections.length <= 1;
  deckTabs.innerHTML = sections.length > 1 ? sections.map((item, index) => `<button type="button" class="deck-tab${index === activeSectionIndex ? " is-active" : ""}" data-section-index="${index}" aria-pressed="${index === activeSectionIndex}">${escapeHTML(item.title)}</button>`).join("") : "";
  const isMulti = section.group === "multi";
  deckPrev.hidden = isMulti;
  deckNext.hidden = isMulti;
  deckCarousel.classList.toggle("color-options", section.kind === "color");
  deckCarousel.innerHTML = section.tags.map((tag) => {
    const selected = state.tags.has(tag);
    const swatchColor = section.kind === "color" ? colorFor(tag, part === "cap" ? "" : part === "stem" ? "stiel_" : part === "flesh" ? "fl_" : "lr_") : "";
    const kindClass = section.kind === "color" ? "swatch-choice" : "icon-choice";
    const substrate = part === "growth" ? substrateIconFor(tag) : "";
    const icon = substrate || (section.kind === "color" ? `<span class="swatch" style="--swatch:${swatchColor}"></span>` : `<span class="choice-icon" aria-hidden="true">${iconFor(tag)}</span>`);
    return `<button type="button" class="deck-option ${kindClass}${section.kind === "color" ? " color-option" : ""}${selected ? " is-selected" : ""}" data-choice-tag="${escapeHTML(tag)}" data-choice-group="${escapeHTML(section.group)}" aria-pressed="${selected}" title="${escapeHTML(labelFor(tag))}">${icon}<span>${escapeHTML(labelFor(tag))}</span></button>`;
  }).join("");
  applyExhausted();
  if (!isMulti) {
    const selected = deckCarousel.querySelector(".deck-option.is-selected");
    if (selected) requestAnimationFrame(() => centerDeckOption(selected));
    else deckCarousel.scrollLeft = 0;
  }
}

function centerDeckOption(option) {
  const optionRect = option.getBoundingClientRect();
  const carouselRect = deckCarousel.getBoundingClientRect();
  const target = deckCarousel.scrollLeft + optionRect.left - carouselRect.left + optionRect.width / 2 - deckCarousel.clientWidth / 2;
  deckCarousel.scrollTo({ left: Math.max(0, target), behavior: window.matchMedia("(prefers-reduced-motion: reduce)").matches ? "auto" : "smooth" });
}

function iconFor(tag) {
  return ({ spitz: "△", schirm: "⌒", kugel: "◠", trichter: "▽", f: "◖", sonstige_form: "⌒", lamellen: "≋", roehren: "○", stachel: "⋮", hut_gerieft: "╱", hut_hygrophan: "◒", flocken: "⁘", haut_schmierig: "◌", ring: "◉", netz: "▦", knollig: "⬭", wurzel: "♧", stiel_hohl: "∥", milch: "◉", bruechig: "⌁", faserig: "╱", bueschelig: "♧", holz: "▰", auf_boden: "⌂", auf_wiese: "♣" })[tag] || "•";
}

function substrateIconFor(tag) {
  const icons = {
    bueschelig: '<svg class="substrate-option-icon" viewBox="0 0 28 18" aria-hidden="true"><path d="M2 16h24"/><path class="cluster-cap" d="M3 10c.5-3 2-5 4-5s3.5 2 4 5c-2 .7-6 .7-8 0Zm8 1c.5-4 2-7 4-7s3.5 3 4 7c-2 .7-6 .7-8 0Zm8-1c.5-3 2-5 4-5s3.5 2 4 5c-2 .7-6 .7-8 0Z"/><path d="M6 10v6m7-5v5m7-6v6"/></svg>',
    holz: '<svg class="substrate-option-icon substrate-option-wood" viewBox="0 0 28 18" aria-hidden="true"><rect x="2" y="10" width="24" height="6" rx="2"/><path d="M5 12h13m-8 2h11m-2-4c4 0 5 5 1 6"/></svg>',
    auf_boden: '<svg class="substrate-option-icon substrate-option-earth" viewBox="0 0 28 18" aria-hidden="true"><path d="M2 11c6-2 18-2 24 0v5H2z"/><path d="M2 11c6-2 18-2 24 0M7 14h2m5-2h2m4 3h2"/></svg>',
    auf_wiese: '<svg class="substrate-option-icon substrate-option-grass" viewBox="0 0 28 18" aria-hidden="true"><path d="M2 15h24M5 15l2-6m3 6V7m3 8 2-7m3 7 1-5m3 5 2-7"/><circle cx="15" cy="6" r="1.5"/></svg>',
  };
  return icons[tag] || "";
}

function openDeck(part) {
  openPart = part;
  activeSectionIndex = 0;
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

function cycleSection(direction) {
  const section = (partChoices[openPart] || [])[activeSectionIndex];
  if (!section || section.group === "multi") return;
  const currentIndex = section.tags.findIndex((tag) => state.tags.has(tag));
  const startIndex = currentIndex < 0 ? (direction > 0 ? -1 : 0) : currentIndex;
  const nextIndex = (startIndex + direction + section.tags.length) % section.tags.length;
  toggleTag(section.tags[nextIndex], section.group);
}

document.querySelectorAll(".mushroom-part").forEach((part) => {
  part.addEventListener("click", () => openDeck(part.dataset.part));
  part.addEventListener("keydown", (event) => {
    if (event.key === "Enter" || event.key === " ") { event.preventDefault(); openDeck(part.dataset.part); }
  });
});
document.querySelector("#deck-close")?.addEventListener("click", closeDeck);
deckTabs?.addEventListener("click", (event) => {
  const tab = event.target.closest("[data-section-index]");
  if (!tab) return;
  activeSectionIndex = Number(tab.dataset.sectionIndex);
  renderDeck(openPart);
});
deckCarousel?.addEventListener("click", (event) => {
  const choice = event.target.closest("[data-choice-tag]");
  if (choice) toggleTag(choice.dataset.choiceTag, choice.dataset.choiceGroup);
});
deckPrev?.addEventListener("click", () => cycleSection(-1));
deckNext?.addEventListener("click", () => cycleSection(1));
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
  if (event.key === "ArrowLeft" || event.key === "ArrowRight") {
    if (document.activeElement.matches("input, textarea, select")) return;
    event.preventDefault();
    cycleSection(event.key === "ArrowRight" ? 1 : -1);
  }
});

activeFilters?.addEventListener("click", (event) => {
  const remove = event.target.closest("[data-remove-tag]");
  if (remove) { state.tags.delete(remove.dataset.removeTag); state.page = 1; updateMushroom(); scheduleUpdate(0); }
  if (event.target.closest("[data-reset-diameter]")) { minSlider.value = minSlider.min; maxSlider.value = maxSlider.max; state.min = minSlider.value; state.max = maxSlider.value; updateDiameterLabel(); scheduleUpdate(0); }
  if (event.target.closest("[data-clear-query]")) { state.q = ""; searchInput.value = ""; state.page = 1; scheduleUpdate(0); }
  if (event.target.closest("#reset-filters-inline")) resetFilters();
});
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

searchInput?.addEventListener("input", () => { state.q = searchInput.value.trim(); state.page = 1; scheduleUpdate(260); });
minSlider?.addEventListener("input", () => {
  if (Number(minSlider.value) > Number(maxSlider.value)) maxSlider.value = minSlider.value;
  state.min = minSlider.value; state.max = maxSlider.value; state.page = 1; updateDiameterLabel(); scheduleUpdate(180);
});
maxSlider?.addEventListener("input", () => {
  if (Number(maxSlider.value) < Number(minSlider.value)) minSlider.value = maxSlider.value;
  state.min = minSlider.value; state.max = maxSlider.value; state.page = 1; updateDiameterLabel(); scheduleUpdate(180);
});

document.addEventListener("keydown", (event) => {
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") { event.preventDefault(); searchInput?.focus(); }
});

window.addEventListener("popstate", () => {
  if (!minSlider || !maxSlider) return;
  const params = new URLSearchParams(location.search);
  state.tags = new Set((params.get("f") || "").split(",").filter(Boolean));
  state.q = params.get("q") || ""; state.page = Number(params.get("page") || 1);
  state.min = params.get("dmin") || minSlider.min; state.max = params.get("dmax") || maxSlider.max;
  if (searchInput) searchInput.value = state.q;
  minSlider.value = state.min; maxSlider.value = state.max;
  updateDiameterLabel(); updateMushroom(); scheduleUpdate(0);
});

updateMushroom();
scheduleUpdate(0);

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
