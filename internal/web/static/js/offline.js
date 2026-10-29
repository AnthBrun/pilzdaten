// Offline-Betrieb: Das Offline-Paket (/api/offline, siehe internal/web/offline.go)
// enthält alle Arten in Kurzform. Damit funktionieren Suche, Filter und
// Artseiten im Funkloch; Fotos der wichtigsten Arten werden mitgespeichert.
import { renderCard } from "./results.js";
import { escapeHTML } from "./util.js";

const OFFLINE_DATA_URL = "/api/offline";
// Cache-Namen müssen zu static/sw.js passen.
const DATA_CACHE = "pilz-data";
const PACK_MEDIA_CACHE = "pilz-media-pack";
const DOWNLOAD_WORKERS = 6;

const offlineButton = document.querySelector("#offline-button");
let packPromise = null;

// Kurze Feldnamen wie in offlineSpecies (internal/web/offline.go):
// s slug, n Name, a alle Namen, w wissenschaftlich, y Synonyme, g Gattung,
// e/l Essbarkeit (Klasse/Text), i Bild, x alle Bilder, t Tags, d Durchmesser,
// p Priorität, c Häufigkeit, k Fotoanzahl, m/z Merkmale, r Verwechslungen, u Quelle.
export function loadOfflinePack() {
  if (!("caches" in window)) return Promise.resolve(null);
  packPromise ||= caches.match(OFFLINE_DATA_URL)
    .then((response) => (response ? response.json() : null))
    .then((pack) => (pack ? indexPack(pack) : null))
    .catch(() => null);
  return packPromise.then((pack) => {
    if (!pack) packPromise = null;
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

// --- Suche: spiegelt catalog.Filter und catalog.AvailableTags (internal/catalog/query.go).

function nameMatchRank(value, query) {
  value = String(value || "").trim().toLowerCase();
  if (value === query) return 0;
  if (value.startsWith(query)) return 1;
  if (value.includes(query)) return 2;
  return 3;
}

function searchRank(species, query) {
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

// Wie speciesRelevanceLess in internal/catalog/field.go.
function relevanceCompare(left, right) {
  const photoBucket = (species) => Math.floor(Math.min(species.k || 0, 40) / 10);
  return left.p - right.p || (right.c || 0) - (left.c || 0) || photoBucket(right) - photoBucket(left) || left.sortKey.localeCompare(right.sortKey);
}

const cardData = (species) => ({ slug: species.s, name: species.n, scientific_name: species.w, edibility_class: species.e, edibility_label: species.l, image: species.i || "" });

// params wie bei /api/search; liefert dieselbe Antwortform plus availableTags.
export function searchOffline(pack, params) {
  const query = (params.get("q") || "").trim().toLowerCase();
  const tags = (params.get("f") || "").split(",").filter(Boolean);
  const min = Number(params.get("dmin") || -Infinity);
  const max = Number(params.get("dmax") || Infinity);
  const page = Math.max(1, Number(params.get("page") || 1));
  const perPage = 50;
  const matches = pack.species.filter((species) => {
    if (query && searchRank(species, query) === Infinity) return false;
    const [low, high] = species.d;
    if (low && high && (low > max || high < min)) return false;
    return tags.every((tag) => species.tagSet.has(tag));
  });
  matches.sort(query ? (left, right) => searchRank(left, query) - searchRank(right, query) || relevanceCompare(left, right) : relevanceCompare);
  const availableTags = new Set(matches.flatMap((species) => species.t));
  const data = {
    count: matches.length, page, total_pages: Math.ceil(matches.length / perPage), per_page: perPage,
    results: matches.slice((page - 1) * perPage, page * perPage).map(cardData),
  };
  return { data, availableTags };
}

// --- "Offline speichern"-Knopf: Paket plus Fotos der Priorität-1-Arten laden.

function setButton(text, stateName) {
  offlineButton.querySelector(".offline-text").textContent = text;
  offlineButton.dataset.state = stateName;
}

async function downloadOfflinePack() {
  setButton("Lade Artdaten …", "busy");
  const response = await fetch(OFFLINE_DATA_URL, { cache: "no-store" });
  if (!response.ok) throw new Error("Offline-Paket nicht verfügbar");
  await (await caches.open(DATA_CACHE)).put(OFFLINE_DATA_URL, response.clone());
  const pack = indexPack(await response.json());
  packPromise = Promise.resolve(pack);
  // Seitenhülle, CSS und JS frisch in den Cache des Service Workers legen.
  navigator.serviceWorker.controller?.postMessage("refresh-shell");

  const images = pack.species.filter((species) => species.p === 1 && species.i).map((species) => species.i);
  const mediaCache = await caches.open(PACK_MEDIA_CACHE);
  const queue = [...images];
  let done = 0;
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
      if (done % 10 === 0 || done === images.length) setButton(`Fotos ${done}/${images.length}`, "busy");
    }
  }
  await Promise.all(Array.from({ length: DOWNLOAD_WORKERS }, worker));
}

export function initOffline() {
  if (!("serviceWorker" in navigator)) return;
  window.addEventListener("load", async () => {
    navigator.serviceWorker.register("/sw.js").catch(() => {});
    if (!offlineButton || !("caches" in window)) return;
    offlineButton.hidden = false;
    const stored = await caches.match(OFFLINE_DATA_URL);
    setButton(stored ? "Offline bereit" : "Offline speichern", stored ? "ready" : "idle");
  });
  offlineButton?.addEventListener("click", async () => {
    if (offlineButton.dataset.state === "busy") return;
    const question = offlineButton.dataset.state === "ready"
      ? "Offline-Daten sind gespeichert. Jetzt aktualisieren?"
      : "Für die Suche ohne Netz werden alle Artdaten und die Fotos der rund 600 bekanntesten Arten gespeichert (ca. 24 MB Download, ca. 35 MB Speicher). Am besten im WLAN laden. Jetzt speichern?";
    if (!window.confirm(question)) return;
    try {
      await downloadOfflinePack();
      setButton("Offline bereit", "ready");
    } catch (_error) {
      setButton("Fehler – erneut versuchen", "idle");
    }
  });
}

// --- Offline-Artseite (templates/offline.html): rendert eine Art aus dem Paket.
// Gegenstück zu templates/detail.html, bitte gemeinsam ändern.

export function initOfflineDetail() {
  const container = document.querySelector("#offline-detail");
  if (!container) return;
  // Offline fehlen ungespeicherte Fotos; ist die Leiste danach leer, verschwindet sie ganz.
  window.dropOfflinePhoto = (image) => {
    const section = image.closest(".photo-strip-section");
    image.parentNode.remove();
    if (section && !section.querySelector(".strip-photo")) section.remove();
  };
  const slug = decodeURIComponent(location.pathname.replace(/^\/art\//, ""));
  loadOfflinePack().then((pack) => {
    const species = pack?.bySlug.get(slug);
    if (!species) return;
    document.title = `${species.n} – Pilzdatenbank (offline)`;
    document.querySelector("#offline-message")?.remove();
    container.insertAdjacentHTML("beforeend", renderSpecies(pack, species));
  });
}

function renderTraitList(traits) {
  return `<dl class="traits-grid">${traits.map(([name, value]) => `<div class="trait-card"><dt>${escapeHTML(name)}</dt><dd>${escapeHTML(value)}</dd></div>`).join("")}</dl>`;
}

function renderSpecies(pack, species) {
  const dangerRank = { deadly: 0, poisonous: 1 };
  const related = (species.r || []).map((slug) => pack.bySlug.get(slug)).filter(Boolean)
    .sort((left, right) => (dangerRank[left.e] ?? 2) - (dangerRank[right.e] ?? 2));
  const dangerous = related.filter((other) => other.e in dangerRank);
  const images = species.x || [];
  const warning = species.e in dangerRank;
  return `
    <section class="detail-hero">
      <div class="detail-hero-copy"><p class="eyebrow">${escapeHTML(species.g || "PILZPORTRÄT")}</p><h1 class="detail-title">${escapeHTML(species.n)}</h1>${species.w ? `<p class="scientific-name">${escapeHTML(species.w)}</p>` : ""}</div>
      ${images[0] ? `<figure class="detail-hero-photo"><img src="${escapeHTML(images[0])}" alt="${escapeHTML(species.n)}" onerror="this.parentNode.remove()"></figure>` : ""}
    </section>
    ${images.length > 1 ? `<section class="photo-strip-section"><h2 class="strip-heading">Bilder <span class="strip-hint">(nur gespeicherte sichtbar)</span></h2><div class="photo-strip">${images.slice(1).map((image) => `<span class="strip-photo"><img src="${escapeHTML(image)}" alt="" loading="lazy" onerror="dropOfflinePhoto(this)"></span>`).join("")}</div></section>` : ""}
    ${dangerous.length ? `<section class="danger-banner"><strong>⚠ Verwechslungsgefahr mit giftigen Arten</strong><p>${dangerous.map((other) => `<a href="/art/${encodeURIComponent(other.s)}">${escapeHTML(other.n)}</a> <span class="danger-label">(${escapeHTML(other.l)})</span>`).join(", ")}</p></section>` : ""}
    ${related.length ? `<section class="detail-section related-section"><div class="detail-section-heading"><div><p class="eyebrow">ÄHNLICHE ARTEN – GENAU VERGLEICHEN</p><h2>Verwechslungspartner</h2></div></div><div class="species-grid related-strip">${related.map((other) => renderCard(cardData(other))).join("")}</div></section>` : ""}
    <section class="edibility-banner edibility-${escapeHTML(species.e)}"><span class="edibility-badge edibility-${escapeHTML(species.e)}">${escapeHTML(species.l)}</span><p>${warning ? "Giftiger Pilz — nicht verzehren." : "Nur die Angabe der Datenquelle – keine Verzehrfreigabe."}</p></section>
    ${species.m?.length ? `<section class="detail-section"><div class="detail-section-heading"><div><p class="eyebrow">ERKENNUNGSMERKMALE</p><h2>Merkmale</h2></div></div>${renderTraitList(species.m)}</section>` : ""}
    ${species.z?.length ? `<details class="supplemental-traits"><summary><span>Weitere Angaben</span></summary>${renderTraitList(species.z)}</details>` : ""}`;
}
