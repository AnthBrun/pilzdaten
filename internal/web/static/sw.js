// Service Worker der Pilzdatenbank: macht die App im Wald ohne Netz nutzbar.
// Seiten und Code kommen bevorzugt frisch aus dem Netz (mit kurzem Timeout,
// weil Funklöcher sonst minutenlang hängen), Bilder bevorzugt aus dem Cache.
const SHELL_CACHE = "pilz-shell-v4";
const PAGE_CACHE = "pilz-pages";
const MEDIA_CACHE = "pilz-media";
const PACK_MEDIA_CACHE = "pilz-media-pack";
const DATA_CACHE = "pilz-data";
// Bildmodell und onnxruntime-web (js/photo.js räumt alte Stände auf).
const MODEL_CACHE = "pilz-model";
// Alles, was die Seiten ohne Netz brauchen. Neue JS-Module hier eintragen.
const SHELL_URLS = [
  "/", "/offline", "/manifest.webmanifest",
  "/static/app.css", "/static/icon-192.png", "/static/icon-512.png",
  "/static/js/main.js", "/static/js/config.js", "/static/js/state.js", "/static/js/util.js", "/static/js/icons.js",
  "/static/js/mushroom.js", "/static/js/deck.js", "/static/js/results.js", "/static/js/offline.js", "/static/js/detail.js",
  "/static/js/photo.js",
];
const NETWORK_TIMEOUT_MS = 4000;
const MAX_RUNTIME_IMAGES = 400;
const MAX_PAGES = 150;

self.addEventListener("install", (event) => {
  event.waitUntil(caches.open(SHELL_CACHE).then((cache) => cache.addAll(SHELL_URLS)).then(() => self.skipWaiting()));
});

self.addEventListener("activate", (event) => {
  const keep = new Set([SHELL_CACHE, PAGE_CACHE, MEDIA_CACHE, PACK_MEDIA_CACHE, DATA_CACHE, MODEL_CACHE]);
  event.waitUntil(
    caches.keys()
      .then((names) => Promise.all(names.filter((name) => !keep.has(name)).map((name) => caches.delete(name))))
      .then(() => self.clients.claim()),
  );
});

function withTimeout(promise, ms) {
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => reject(new Error("timeout")), ms);
    promise.then((value) => { clearTimeout(timer); resolve(value); }, (error) => { clearTimeout(timer); reject(error); });
  });
}

async function trimCache(name, maxEntries) {
  const cache = await caches.open(name);
  const keys = await cache.keys();
  for (let index = 0; index < keys.length - maxEntries; index++) await cache.delete(keys[index]);
}

async function networkFirst(request, cacheName, cacheKey = request) {
  const cache = await caches.open(cacheName);
  try {
    const response = await withTimeout(fetch(request), NETWORK_TIMEOUT_MS);
    if (response.ok) cache.put(cacheKey, response.clone());
    return response;
  } catch (error) {
    const cached = await cache.match(cacheKey, { ignoreSearch: cacheKey === "/" });
    if (cached) return cached;
    throw error;
  }
}

async function handleNavigation(request) {
  const url = new URL(request.url);
  if (url.pathname.startsWith("/art/")) {
    try {
      const response = await networkFirst(request, PAGE_CACHE);
      trimCache(PAGE_CACHE, MAX_PAGES);
      return response;
    } catch (_error) {
      // Nicht gespeicherte Artseite: Die Offline-Hülle rendert sie aus dem Offline-Paket.
      return (await caches.match("/offline")) || Response.error();
    }
  }
  if (url.pathname === "/") {
    try {
      return await networkFirst(request, SHELL_CACHE, "/");
    } catch (_error) {
      return (await caches.match("/")) || Response.error();
    }
  }
  return fetch(request);
}

async function handleImage(request) {
  const cached = await caches.match(request);
  if (cached) return cached;
  const response = await fetch(request);
  if (response.ok) {
    const cache = await caches.open(MEDIA_CACHE);
    await cache.put(request, response.clone());
    trimCache(MEDIA_CACHE, MAX_RUNTIME_IMAGES);
  }
  return response;
}

// Modell und Laufzeit sind über ?v=<Version> bzw. ort-<Version>/ eindeutig
// benannt: einmal geladen, bleiben sie gültig und gehen auch ohne Netz.
async function handleModel(request) {
  const cache = await caches.open(MODEL_CACHE);
  const cached = await cache.match(request);
  if (cached) return cached;
  const response = await fetch(request);
  if (response.status === 200) await cache.put(request, response.clone());
  return response;
}

// "Offline speichern" in js/offline.js bittet darum, die Hülle neu zu laden.
self.addEventListener("message", (event) => {
  if (event.data === "refresh-shell") event.waitUntil(caches.open(SHELL_CACHE).then((cache) => cache.addAll(SHELL_URLS)));
});

self.addEventListener("fetch", (event) => {
  const { request } = event;
  if (request.method !== "GET") return;
  const url = new URL(request.url);
  if (url.origin !== self.location.origin) return;
  if (request.mode === "navigate") {
    event.respondWith(handleNavigation(request));
    return;
  }
  if (url.pathname.startsWith("/media/")) {
    event.respondWith(handleImage(request));
    return;
  }
  if (url.pathname.startsWith("/model/")) {
    event.respondWith(handleModel(request));
    return;
  }
  if (url.pathname.startsWith("/static/") || url.pathname === "/manifest.webmanifest") {
    event.respondWith(networkFirst(request, SHELL_CACHE));
  }
  // /api/* läuft direkt übers Netz; ohne Netz rechnet js/offline.js mit dem Offline-Paket.
});
