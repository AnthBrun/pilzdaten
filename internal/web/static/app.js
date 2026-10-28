const searchInput = document.querySelector("#species-search");
const resultGrid = document.querySelector("#species-results");
const resultsLabel = document.querySelector("#results-label");
const searchHint = document.querySelector("#search-hint");
let requestTimer;

function makeCard(species) {
  const link = document.createElement("a");
  link.className = "species-card";
  link.href = `/art/${encodeURIComponent(species.slug)}`;

  const symbol = document.createElement("span");
  symbol.className = "card-symbol";
  symbol.setAttribute("aria-hidden", "true");
  symbol.textContent = "◌";
  const copy = document.createElement("span");
  copy.className = "card-copy";
  const name = document.createElement("strong");
  name.textContent = species.name_trivial;
  const scientific = document.createElement("em");
  scientific.textContent = species.name_wissenschaftlich || "Wissenschaftlicher Name nicht erfasst";
  copy.append(name, scientific);
  const arrow = document.createElement("span");
  arrow.className = "card-arrow";
  arrow.setAttribute("aria-hidden", "true");
  arrow.textContent = "↗";
  link.append(symbol, copy, arrow);
  return link;
}

async function searchSpecies() {
  const query = searchInput.value.trim();
  if (!query) {
    window.location.href = "/";
    return;
  }
  resultsLabel.textContent = `Suchergebnisse für „${query}“`;
  searchHint.textContent = "Suche in Trivialnamen, wissenschaftlichen Namen und Synonymen …";
  try {
    const response = await fetch(`/api/search?q=${encodeURIComponent(query)}`);
    if (!response.ok) throw new Error("Suche fehlgeschlagen");
    const species = await response.json();
    resultGrid.replaceChildren();
    if (species.length === 0) {
      const empty = document.createElement("p");
      empty.className = "empty-state";
      empty.textContent = "Keine passenden Arten gefunden. Versuche einen anderen Suchbegriff.";
      resultGrid.append(empty);
    } else {
      resultGrid.append(...species.map(makeCard));
    }
    searchHint.textContent = `${species.length}${species.length === 30 ? "+" : ""} Treffer`;
  } catch {
    searchHint.textContent = "Die Suche ist gerade nicht erreichbar.";
  }
}

searchInput?.addEventListener("input", () => {
  clearTimeout(requestTimer);
  if (!searchInput.value.trim()) {
    window.location.replace("/");
    return;
  }
  requestTimer = setTimeout(searchSpecies, 180);
});

document.addEventListener("keydown", (event) => {
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
    event.preventDefault();
    searchInput?.focus();
  }
});
