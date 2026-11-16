// Foto-Vorschläge: Ein Bildmodell (Repo pilz-ki) erkennt auf Fotos sichtbare
// Merkmale (Hutfarbe, Lamellen, Ring, Untergrund …) und füllt damit die
// Filter vor. Es läuft komplett im Browser (onnxruntime-web, WASM): Fotos
// verlassen das Gerät nie. Laufzeit und Modell (~30 MB) werden erst beim
// ersten Antippen von „Foto“ geladen und danach vom Service Worker gehalten.
//
// Die Regeln spiegeln pilzki/predict.py (Referenz in pilz-ki):
// Vorverarbeitung laut meta.json.input, Gate gegen Fotos ohne Pilz,
// Mittelwert über alle Fotos eines Fundes, höchstens ein Tag je exklusiver
// Gruppe, Schwelle je Tag. Wer hier etwas ändert, ändert es auch dort.
import { offlineCards } from "./offline.js";
import { escapeHTML, fetchJSON, t } from "./util.js";

const button = document.querySelector("#photo-button");
const input = document.querySelector("#photo-input");
const statusBox = document.querySelector("#photo-status");
// Cache-Name muss zu static/sw.js passen.
const MODEL_CACHE = "pilz-model";
const MAX_PHOTOS = 6;
const SIMILAR_SPECIES = 3;

let modelPromise = null;

const version = () => button.dataset.modelVersion;
const runtime = () => button.dataset.modelRuntime;
const modelURL = (name) => `/model/${name}?v=${encodeURIComponent(version())}`;

// Modell mit Fortschritt laden (nur der erste Abruf dauert, danach Cache).
async function fetchBytes(url, onProgress) {
  const response = await fetch(url);
  if (!response.ok) throw new Error(`HTTP ${response.status}`);
  // Bei gzip nennt Content-Length die gepackte Größe; der Server schickt die echte mit.
  const encoded = response.headers.get("Content-Encoding");
  const total = Number(response.headers.get(encoded ? "X-Decoded-Length" : "Content-Length")) || 0;
  if (!response.body || !total) return new Uint8Array(await response.arrayBuffer());
  const reader = response.body.getReader();
  const chunks = [];
  let loaded = 0;
  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;
    chunks.push(value);
    loaded += value.length;
    onProgress?.(Math.min(loaded, total), total);
  }
  const bytes = new Uint8Array(loaded);
  let offset = 0;
  for (const chunk of chunks) {
    bytes.set(chunk, offset);
    offset += chunk.length;
  }
  return bytes;
}

function loadModel(onProgress) {
  modelPromise ||= (async () => {
    const ort = await import(`${runtime()}ort.wasm.min.mjs`);
    ort.env.wasm.wasmPaths = runtime();
    // Mehrere Threads gehen nur mit Cross-Origin-Isolation; sonst einer.
    ort.env.wasm.numThreads = self.crossOriginIsolated ? Math.min(4, navigator.hardwareConcurrency || 1) : 1;
    const meta = await fetchJSON(modelURL("meta.json"));
    const bytes = await fetchBytes(modelURL("model.int8.onnx"), onProgress);
    const session = await ort.InferenceSession.create(bytes, { executionProviders: ["wasm"] });
    dropOldModels();
    return { ort, meta, session };
  })().catch((error) => {
    modelPromise = null;
    throw error;
  });
  return modelPromise;
}

// Ältere Modell- und Laufzeitstände aus dem Cache werfen.
async function dropOldModels() {
  if (!("caches" in window)) return;
  try {
    const cache = await caches.open(MODEL_CACHE);
    for (const request of await cache.keys()) {
      const url = new URL(request.url);
      const current = url.pathname.startsWith(runtime()) || url.searchParams.get("v") === version();
      if (!current) await cache.delete(request);
    }
  } catch (_error) { /* Cache nicht verfügbar: egal */ }
}

// Foto -> Tensor [1, 3, S, S], RGB in [0, 1]: kurze Seite auf
// resize_short_side skalieren (hochwertiger Skalierer des Browsers), dann
// mittig center_crop Pixel 1:1 ausschneiden – dieselbe Reihenfolge wie
// predict.load_image in pilz-ki.
async function toTensor(file, inputSpec, ort) {
  const original = await createImageBitmap(file, { imageOrientation: "from-image" });
  const size = inputSpec.center_crop;
  const scale = inputSpec.resize_short_side / Math.min(original.width, original.height);
  const width = Math.max(size, Math.round(original.width * scale));
  const height = Math.max(size, Math.round(original.height * scale));
  const bitmap = await createImageBitmap(original, { resizeWidth: width, resizeHeight: height, resizeQuality: "high" });
  original.close?.();
  const canvas = document.createElement("canvas");
  canvas.width = size;
  canvas.height = size;
  const context = canvas.getContext("2d", { willReadFrequently: true });
  context.drawImage(bitmap, Math.floor((width - size) / 2), Math.floor((height - size) / 2), size, size, 0, 0, size, size);
  bitmap.close?.();
  const { data } = context.getImageData(0, 0, size, size);
  const plane = size * size;
  const pixels = new Float32Array(3 * plane);
  for (let index = 0; index < plane; index++) {
    pixels[index] = data[4 * index] / 255;
    pixels[plane + index] = data[4 * index + 1] / 255;
    pixels[2 * plane + index] = data[4 * index + 2] / 255;
  }
  return new ort.Tensor("float32", pixels, [1, 3, size, size]);
}

// Tags, die vorbelegt werden (wie evaluate.prefill / predict.prefill).
export function prefill(probabilities, meta) {
  const index = new Map(meta.tags.map((entry, position) => [entry.tag, position]));
  const threshold = new Map(meta.tags.map((entry) => [entry.tag, entry.threshold]));
  const p = (tag) => probabilities[index.get(tag)];
  const chosen = [];
  for (const group of meta.groups) {
    if (group.requires && p(group.requires) < meta.require_p) continue;
    if (group.exclusive) {
      const best = group.tags.reduce((left, right) => (p(right) > p(left) ? right : left));
      const limit = threshold.get(best);
      if (limit !== null && limit !== undefined && p(best) >= limit) chosen.push(best);
    } else {
      for (const tag of group.tags) {
        const limit = threshold.get(tag);
        if (limit !== null && limit !== undefined && p(tag) >= limit) chosen.push(tag);
      }
    }
  }
  return chosen;
}

function mean(rows, key) {
  const result = new Float32Array(rows[0][key].length);
  for (const row of rows) {
    const values = row[key];
    for (let index = 0; index < result.length; index++) result[index] += values[index] / rows.length;
  }
  return result;
}

// Mehrere Fotos eines Fundes: Fotos ohne Pilz (Gate) fallen raus, der Rest wird gemittelt.
export async function analyzePhotos(files, onProgress) {
  const { ort, meta, session } = await loadModel(onProgress);
  const rows = [];
  for (const file of files) {
    const output = await session.run({ image: await toTensor(file, meta.input, ort) });
    rows.push({
      tags: output.tag_probs.data,
      species: output.species_probs.data,
      fungus: meta.gate ? output.fungus_prob.data[0] : 1,
    });
  }
  const passing = meta.gate ? rows.filter((row) => row.fungus >= meta.gate.threshold) : rows;
  if (!passing.length) return { fungus: false, photos: rows.length, used: 0, tags: [], species: [], groups: meta.groups };
  const species = mean(passing, "species");
  const top = [...species.keys()].sort((left, right) => species[right] - species[left]).slice(0, SIMILAR_SPECIES);
  return {
    fungus: true,
    photos: rows.length,
    used: passing.length,
    tags: prefill(mean(passing, "tags"), meta),
    // Arten unter 1 % sind Rauschen; die wahrscheinlichste bleibt immer.
    species: top.filter((position, rank) => rank === 0 || species[position] >= 0.01).map((position) => ({ slug: meta.species[position], p: species[position] })),
    groups: meta.groups,
  };
}

// --- Oberfläche

function showStatus(html, kind = "info") {
  statusBox.hidden = false;
  statusBox.dataset.kind = kind;
  statusBox.innerHTML = html;
}

async function similarCards(species) {
  const slugs = species.map((entry) => entry.slug);
  try {
    if (!navigator.onLine) throw new Error("offline");
    return (await fetchJSON(`/api/species?slugs=${encodeURIComponent(slugs.join(","))}`)).results || [];
  } catch (_error) {
    return offlineCards(slugs);
  }
}

function megabytes(bytes) {
  return (bytes / 1e6).toFixed(0);
}

// onSuggest(tags, groups) setzt die Tags in den Suchzustand und liefert
// { applied, kept, conflicts } (siehe applySuggestions in state.js).
export function initPhoto(onSuggest) {
  if (!button || !input || !statusBox) return;
  button.addEventListener("click", () => input.click());
  input.addEventListener("change", async () => {
    const files = [...input.files].slice(0, MAX_PHOTOS);
    input.value = "";
    if (!files.length) return;
    button.disabled = true;
    const working = files.length === 1 ? t("Foto wird ausgewertet …") : t("{count} Fotos werden ausgewertet …", { count: files.length });
    showStatus(`<span class="photo-spinner" aria-hidden="true"></span>${escapeHTML(modelPromise ? working : t("Bildmodell wird geladen …"))}`);
    try {
      const result = await analyzePhotos(files, (loaded, total) => {
        showStatus(`<span class="photo-spinner" aria-hidden="true"></span>${escapeHTML(t("Bildmodell wird geladen … {loaded} von {total} MB (nur beim ersten Mal)", { loaded: megabytes(loaded), total: megabytes(total) }))}`);
      });
      renderResult(result, onSuggest(result.tags, result.groups));
    } catch (error) {
      console.error(error);
      const offline = !navigator.onLine;
      showStatus(escapeHTML(offline
        ? t("Ohne Netz geht das nur, wenn das Bildmodell schon einmal geladen wurde.")
        : t("Das Foto konnte nicht ausgewertet werden. Bitte später noch einmal versuchen.")), "error");
    } finally {
      button.disabled = false;
    }
  });
}

// Kompakte Zeile statt der großen Trefferkarte: Bild, Name, Wahrscheinlichkeit.
function similarRow(card, percent) {
  const name = escapeHTML(card.name);
  const image = card.image ? `<img src="${escapeHTML(card.image)}" alt="" loading="lazy" decoding="async">` : '<span class="photo-similar-empty" aria-hidden="true">✳</span>';
  return `<a class="photo-similar-item edibility-card-${escapeHTML(card.edibility_class || "unknown")}" href="/art/${escapeHTML(encodeURIComponent(card.slug))}">${image}<span class="photo-similar-name"><strong>${name}</strong><em>${escapeHTML(card.scientific_name || "")}</em></span><span class="photo-similar-score">${percent} %</span></a>`;
}

async function renderResult(result, { applied, kept, conflicts }) {
  statusBox.dataset.prefill = result.tags.join(","); // zum Nachvollziehen (und für Tests)
  if (!result.fungus) {
    showStatus(escapeHTML(result.photos === 1
      ? t("Auf dem Foto ist kein Pilz zu erkennen. Fotografiere den Fruchtkörper aus der Nähe.")
      : t("Auf keinem der Fotos ist ein Pilz zu erkennen. Fotografiere den Fruchtkörper aus der Nähe.")), "warn");
    return;
  }
  const lines = [];
  if (applied.length) {
    lines.push(applied.length === 1
      ? t("1 Merkmal aus dem Foto vorgeschlagen. Bitte prüfen – falsche einfach antippen und entfernen.")
      : t("{count} Merkmale aus dem Foto vorgeschlagen. Bitte prüfen – falsche einfach antippen und entfernen.", { count: applied.length }));
  } else if (kept.length) {
    lines.push(t("Das Foto bestätigt deine Auswahl, neue Merkmale ließen sich nicht sicher erkennen."));
  } else {
    lines.push(t("Kein Merkmal ließ sich sicher erkennen. Hilfreich sind Fotos von Hut, Unterseite und Stiel bei gutem Licht."));
  }
  if (conflicts.length) lines.push(conflicts.length === 1
    ? t("1 Vorschlag widerspricht deiner Auswahl und wurde nicht gesetzt.")
    : t("{count} Vorschläge widersprechen deiner Auswahl und wurden nicht gesetzt.", { count: conflicts.length }));
  if (result.used < result.photos) lines.push(t("{count} Fotos ohne erkennbaren Pilz wurden ignoriert.", { count: result.photos - result.used }));
  const text = lines.map((line) => `<p>${escapeHTML(line)}</p>`).join("");
  showStatus(text, applied.length ? "ok" : "info");
  const cards = await similarCards(result.species);
  if (!cards.length) return;
  const percent = new Map(result.species.map((entry) => [entry.slug, Math.round(entry.p * 100)]));
  showStatus(`${text}<p class="photo-similar-title">${escapeHTML(t("Ähnlich aussehende Arten – keine Bestimmung, nur zum Vergleichen:"))}</p><div class="photo-similar">${cards.map((card) => similarRow(card, percent.get(card.slug) ?? 0)).join("")}</div>`, applied.length ? "ok" : "info");
}
