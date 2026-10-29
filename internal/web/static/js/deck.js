// Auswahl-Deck: öffnet sich beim Antippen eines Pilzteils und zeigt dessen
// Optionen aus partChoices als wischbare Reihen.
import { colorFor, labelFor, partChoices, partNames } from "./config.js";
import { optionIcon } from "./icons.js";
import { escapeHTML, scrollBehavior } from "./util.js";

const deck = document.querySelector("#option-deck");
const deckTitle = document.querySelector("#deck-title");
const deckSections = document.querySelector("#deck-sections");
const pulseTimers = new WeakMap();
let openPart = "";

export const currentPart = () => openPart;

export function renderDeck(tags) {
  const sections = partChoices[openPart] || [];
  if (!deck || !sections.length) return;
  // Wischposition jeder Reihe bleibt beim Neuzeichnen erhalten.
  const scrollPositions = new Map([...deckSections.querySelectorAll(".deck-carousel")].map((carousel) => [carousel.dataset.sectionIndex, carousel.scrollLeft]));
  deckTitle.textContent = `${partNames[openPart]} wählen`;
  deckSections.innerHTML = sections.map((section, index) => {
    const options = section.tags.map((tag) => renderOption(tag, section, tags.has(tag))).join("");
    return `<section class="deck-section"><h4>${escapeHTML(section.title)}</h4><div class="deck-carousel${section.kind === "color" ? " color-options" : ""}" data-section-index="${index}" role="group" aria-label="${escapeHTML(section.title)}">${options}</div></section>`;
  }).join("");
  deckSections.querySelectorAll(".deck-carousel").forEach((carousel) => {
    const priorScroll = scrollPositions.get(carousel.dataset.sectionIndex);
    if (priorScroll !== undefined) carousel.scrollLeft = priorScroll;
    else {
      const selected = carousel.querySelector(".deck-option.is-selected");
      if (selected) requestAnimationFrame(() => centerOption(selected, carousel));
    }
  });
}

function renderOption(tag, section, selected) {
  const isColor = section.kind === "color";
  const icon = optionIcon(tag, openPart, isColor ? colorFor(tag, section.prefix) : "");
  const label = escapeHTML(labelFor(tag));
  return `<button type="button" class="deck-option ${isColor ? "color-option" : "icon-choice"}${selected ? " is-selected" : ""}" data-choice-tag="${escapeHTML(tag)}" data-choice-group="${escapeHTML(section.group)}" aria-pressed="${selected}" title="${label}">${icon}<span>${label}</span></button>`;
}

function centerOption(option, carousel) {
  const optionRect = option.getBoundingClientRect();
  const carouselRect = carousel.getBoundingClientRect();
  const target = carousel.scrollLeft + optionRect.left - carouselRect.left + optionRect.width / 2 - carousel.clientWidth / 2;
  carousel.scrollTo({ left: Math.max(0, target), behavior: scrollBehavior() });
}

export function openDeck(part, tags) {
  openPart = part;
  renderDeck(tags);
  deck.hidden = false;
  document.querySelectorAll(".mushroom-part").forEach((node) => node.classList.toggle("is-open", node.dataset.part === part));
}

export function closeDeck() {
  if (!deck) return;
  deck.hidden = true;
  openPart = "";
  document.querySelectorAll(".mushroom-part").forEach((node) => node.classList.remove("is-open"));
}

// Kurzes Aufleuchten des Teils, wenn eine exklusive Wahl gewechselt hat.
export function pulsePart(partName) {
  const part = document.querySelector(`.mushroom-part[data-part="${partName}"]`);
  if (!part) return;
  clearTimeout(pulseTimers.get(part));
  part.classList.remove("is-cycled");
  void part.getBoundingClientRect();
  part.classList.add("is-cycled");
  pulseTimers.set(part, setTimeout(() => part.classList.remove("is-cycled"), 400));
}

// onChoose(tag, group) wird bei jedem Tipp auf eine Option aufgerufen.
export function initDeck(getTags, onChoose) {
  if (!deck) return;
  document.querySelectorAll(".mushroom-part").forEach((part) => {
    part.addEventListener("click", () => openDeck(part.dataset.part, getTags()));
    part.addEventListener("keydown", (event) => {
      if (event.key === "Enter" || event.key === " ") { event.preventDefault(); openDeck(part.dataset.part, getTags()); }
    });
  });
  document.querySelector("#deck-close")?.addEventListener("click", closeDeck);
  deckSections.addEventListener("click", (event) => {
    const choice = event.target.closest("[data-choice-tag]");
    if (choice) onChoose(choice.dataset.choiceTag, choice.dataset.choiceGroup);
  });
  // Tipp außerhalb von Deck und Zeichnung schließt das Deck.
  document.addEventListener("click", (event) => {
    if (deck.hidden) return;
    const inside = event.composedPath().some((node) => node === deck || node.classList?.contains("mushroom-part"));
    if (!inside) closeDeck();
  });
  document.addEventListener("keydown", (event) => {
    if (!deck.hidden && event.key === "Escape") closeDeck();
  });
}
