// Suchzustand der Startseite. Er spiegelt sich 1:1 in der URL
// (?f=tags&q=…&dmin=…&dmax=…&page=…&letter=…), damit Zurück, Teilen und
// Neuladen dieselben Treffer zeigen.
import { exclusiveGroups } from "./config.js";

const minSlider = document.querySelector("#diameter-min");
const maxSlider = document.querySelector("#diameter-max");

export const state = {
  tags: new Set(),
  q: "",
  min: "",
  max: "",
  page: 1,
  letter: "",
  // Tags, die mit der aktuellen Auswahl noch Treffer liefern (null = unbekannt).
  availableTags: null,
};

export const diameterBounds = () => ({ min: minSlider?.min ?? "1", max: maxSlider?.max ?? "25" });

export function readStateFromURL() {
  const params = new URLSearchParams(location.search);
  state.tags = new Set((params.get("f") || "").split(",").map((tag) => tag.trim()).filter(Boolean));
  state.q = params.get("q") || "";
  state.page = Math.max(1, Number(params.get("page") || 1) || 1);
  state.letter = params.get("letter") || "";
  const bounds = diameterBounds();
  let min = params.get("dmin") || bounds.min;
  let max = params.get("dmax") || bounds.max;
  if (Number(min) > Number(max)) [min, max] = [max, min];
  setDiameter(min, max);
}

export function setDiameter(min, max) {
  if (minSlider) minSlider.value = min;
  if (maxSlider) maxSlider.value = max;
  state.min = minSlider?.value ?? min;
  state.max = maxSlider?.value ?? max;
}

export function hasCustomDiameter() {
  const bounds = diameterBounds();
  return state.min !== bounds.min || state.max !== bounds.max;
}

export function paramsForRequest() {
  const params = new URLSearchParams();
  const bounds = diameterBounds();
  if (state.q) params.set("q", state.q);
  if (state.tags.size) params.set("f", [...state.tags].join(","));
  if (state.min !== bounds.min) params.set("dmin", state.min);
  if (state.max !== bounds.max) params.set("dmax", state.max);
  if (state.page > 1) params.set("page", String(state.page));
  if (state.letter) params.set("letter", state.letter);
  return params;
}

export function syncURL(historyMode = "replace") {
  const params = paramsForRequest();
  const nextURL = `${location.pathname}${params.size ? `?${params}` : ""}`;
  if (`${location.pathname}${location.search}` === nextURL) return;
  if (historyMode === "push") history.pushState(null, "", nextURL);
  else history.replaceState(null, "", nextURL);
}

// Schaltet einen Tag um. In exklusiven Gruppen ersetzt er die bisherige
// Wahl, aber nur innerhalb des offenen Teils (eine Hutfarbe ersetzt keine
// Lamellenfarbe). Liefert true, wenn eine exklusive Wahl gewechselt hat.
export function toggleTagInState(tag, group, part) {
  if (state.tags.has(tag)) {
    state.tags.delete(tag);
    state.page = 1;
    return false;
  }
  let replaced = false;
  if (group !== "multi") {
    const partGroup = exclusiveGroups[part]?.[group];
    const sameGroup = partGroup
      ? (existing) => partGroup.has(existing)
      : (existing) => Object.values(exclusiveGroups).some((groups) => groups[group]?.has(existing));
    for (const existing of [...state.tags]) {
      if (sameGroup(existing)) state.tags.delete(existing);
    }
    replaced = true;
  }
  state.tags.add(tag);
  state.page = 1;
  return replaced;
}

export function resetState() {
  state.tags.clear();
  state.q = "";
  state.page = 1;
  state.letter = "";
  const bounds = diameterBounds();
  setDiameter(bounds.min, bounds.max);
}
