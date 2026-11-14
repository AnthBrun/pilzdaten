// Kleine Helfer ohne Bezug zur Pilz-Logik.

// Übersetzung: Der Server bettet das Wörterbuch „deutscher Text → Übersetzung“
// als <script id="i18n"> ein (siehe internal/i18n). Ohne Eintrag bleibt der
// deutsche Text. Platzhalter {name} werden aus params ersetzt.
// Texte immer als String-Literal übergeben: cmd/translate sucht nach t("…").
let dictionary;
export function t(text, params = {}) {
  if (dictionary === undefined) {
    try { dictionary = JSON.parse(document.getElementById("i18n")?.textContent || "{}"); } catch (_error) { dictionary = {}; }
  }
  return Object.entries(params).reduce((result, [key, value]) => result.replaceAll(`{${key}}`, String(value)), dictionary[text] || text);
}

const REQUEST_TIMEOUT_MS = 6000;

export function escapeHTML(value) {
  return String(value ?? "").replace(/[&<>"']/g, (char) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[char]);
}

export function readSession(key) {
  try { return sessionStorage.getItem(key); } catch (_error) { return null; }
}

export function writeSession(key, value) {
  try { sessionStorage.setItem(key, value); } catch (_error) { /* privater Modus */ }
}

export function scrollBehavior() {
  return window.matchMedia("(prefers-reduced-motion: reduce)").matches ? "auto" : "smooth";
}

// fetch mit Timeout: In Funklöchern hängt ein Request sonst minutenlang.
export async function fetchJSON(url) {
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

// Erstes Element aus `tags`, das gerade ausgewählt ist.
export function activeFrom(tags, selected) {
  return [...tags].find((tag) => selected.has(tag));
}
