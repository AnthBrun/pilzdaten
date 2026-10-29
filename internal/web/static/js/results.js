// Darstellung der Treffer: Artkarten, aktive Filter-Chips, Seitenwechsel,
// Durchmesser-Anzeige und die Trefferleiste unten auf dem Handy.
import { groupTags, labelFor, monthTags } from "./config.js";
import { hasCustomDiameter, state } from "./state.js";
import { escapeHTML } from "./util.js";

const results = document.querySelector("#species-results");
const resultLabel = document.querySelector("#results-label");
const resultNumber = document.querySelector(".result-number");
const activeFilters = document.querySelector("#active-filters");
const pagination = document.querySelector(".pagination");
const diameterValue = document.querySelector("#diameter-value");
const resultsColumn = document.querySelector("#results");
const resultsDock = document.querySelector("#results-dock");

// species: { slug, name, scientific_name, edibility_class, edibility_label, image }
export function renderCard(species) {
  const name = escapeHTML(species.name);
  const scientific = escapeHTML(species.scientific_name || "Wissenschaftlicher Name nicht erfasst");
  const edibilityClass = escapeHTML(species.edibility_class || "unknown");
  const edibilityLabel = escapeHTML(species.edibility_label || "Essbarkeit nicht erfasst");
  const image = species.image ? `<img class="card-image" src="${escapeHTML(species.image)}" alt="${name}" loading="lazy" decoding="async">` : '<span class="card-image card-image-empty" aria-hidden="true">✳</span>';
  return `<a class="species-card edibility-card-${edibilityClass}" href="/art/${escapeHTML(encodeURIComponent(species.slug))}">${image}<span class="card-copy"><strong>${name}</strong><em>${scientific}</em><span class="visually-hidden">Essbarkeit: ${edibilityLabel}. Keine Verzehrfreigabe.</span></span><span class="card-arrow" aria-hidden="true">↗</span></a>`;
}

// data: Antwort von /api/search (oder searchOffline).
export function renderResults(data) {
  const y = window.scrollY;
  resultLabel.textContent = `${data.count} passende ${data.count === 1 ? "Art" : "Arten"}`;
  resultNumber.textContent = data.total_pages > 1 ? `Seite ${data.page} von ${data.total_pages}` : "";
  // Bei null Treffern die zwei häufigsten Ursachen mit einem Tipp lösbar machen.
  const activeMonth = monthTags.find((tag) => state.tags.has(tag));
  const activeGroup = groupTags.find((tag) => state.tags.has(tag));
  const hints = [
    activeMonth && `<button class="reset-inline" type="button" data-remove-tag="${activeMonth}">Ohne Monatsfilter suchen</button>`,
    activeGroup && `<button class="reset-inline" type="button" data-remove-tag="${activeGroup}">In allen Pilzgruppen suchen</button>`,
  ].filter(Boolean).map((hint) => ` ${hint}`).join("");
  results.innerHTML = data.results.length
    ? data.results.map(renderCard).join("")
    : `<p class="empty-state">Keine passenden Arten gefunden. Entferne zuletzt gesetzte Merkmale.${hints}</p>`;
  window.scrollTo(0, y);
  updateDock(data);
  pagination.innerHTML = `${data.page > 1 ? '<button type="button" data-page="-1">← Zurück</button>' : "<span></span>"}${data.total_pages > 1 ? `<span>Seite ${data.page} / ${data.total_pages}</span>` : ""}${data.page < data.total_pages ? '<button type="button" data-page="1">Weiter →</button>' : ""}`;
}

export function renderMessage(html) {
  results.innerHTML = `<p class="empty-state">${html}</p>`;
}

export function renderActiveFilters() {
  if (!activeFilters) return;
  const customDiameter = hasCustomDiameter();
  const hasFilters = state.tags.size > 0 || state.q !== "" || customDiameter;
  activeFilters.hidden = !hasFilters;
  activeFilters.innerHTML = [
    ...[...state.tags].map((tag) => `<button class="filter-chip" type="button" data-remove-tag="${escapeHTML(tag)}">${escapeHTML(labelFor(tag))} <span aria-hidden="true">×</span></button>`),
    ...(customDiameter ? [`<button class="filter-chip" type="button" data-reset-diameter>Hut ${escapeHTML(state.min)}–${escapeHTML(state.max)} cm <span aria-hidden="true">×</span></button>`] : []),
    ...(state.q ? [`<button class="filter-chip" type="button" data-clear-query>„${escapeHTML(state.q)}“ <span aria-hidden="true">×</span></button>`] : []),
    ...(hasFilters ? ['<button class="reset-inline" type="button" id="reset-filters-inline">Zurücksetzen</button>'] : []),
  ].join("");
}

export function renderDiameter() {
  if (!diameterValue) return;
  diameterValue.textContent = hasCustomDiameter() ? `${state.min} – ${state.max} cm` : "alle Werte";
  document.querySelectorAll("[data-diameter]").forEach((button) => {
    const [low, high] = button.dataset.diameter.split("-");
    const selected = state.min === low && state.max === high;
    button.classList.toggle("is-selected", selected);
    button.setAttribute("aria-pressed", String(selected));
  });
}

// Chips und Deck-Optionen markieren: gewählt bzw. ohne Treffer ("erschöpft").
export function renderOptionStates() {
  const known = state.availableTags instanceof Set;
  document.querySelectorAll("[data-trait-tag]").forEach((button) => button.classList.toggle("is-selected", state.tags.has(button.dataset.traitTag)));
  document.querySelectorAll("[data-trait-tag], #option-deck [data-choice-tag]").forEach((option) => {
    const tag = option.dataset.traitTag || option.dataset.choiceTag;
    option.classList.toggle("is-exhausted", known && !state.tags.has(tag) && !state.availableTags.has(tag));
  });
}

// Trefferleiste: auf dem Handy liegt die Ergebnisliste unter dem Pilz-Lab.
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

export function syncDock() {
  if (!resultsDock || !resultsColumn) return;
  resultsDock.hidden = resultsColumn.getBoundingClientRect().top < window.innerHeight - 60;
}
