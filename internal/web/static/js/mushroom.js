// Überträgt die gewählten Merkmale auf die Pilzzeichnung
// (templates/mushroom.html). Die Zeichnung selbst ist reines SVG; hier
// werden nur Farben, Formen und Sichtbarkeiten umgeschaltet.
import { capShapes, colorFor, colorTags, groupTags, labelFor, partChoices, partNames, substrateTags, surfaceTags, undersideTypes } from "./config.js";
import { activeFrom } from "./util.js";

const capPaths = {
  spitz: "M30 132C50 75 81 38 120 28c39 10 70 47 90 104-45 15-135 15-180 0Z",
  schirm: "M30 132C44 78 71 62 120 61c49 1 76 17 90 71-45 15-135 15-180 0Z",
  trichter: "M30 132C22 120 18 96 16 76c50-12 158-12 208 0-2 20-6 44-14 56-45 15-135 15-180 0Z",
  sonstige_form: "M30 132C36 82 70 43 120 39c50 4 84 43 90 93-45 15-135 15-180 0Z",
};

// Längsschnitt passend zur gewählten Gruppe, alle im selben Rahmen um (190, 225).
const fleshShapes = {
  hutpilz: "M166 228c0-18 12-30 24-30s24 12 24 30c0 3-8 4-15 4l-2 20c0 4-14 4-14 0l-2-20c-7 0-15-1-15-4Z",
  kugel: "M190 199c14 0 23 10 23 23 0 10-5 16-9 20-2 3-3 7-3 10h-22c0-3-1-7-3-10-4-4-9-10-9-20 0-13 9-23 23-23Z",
  hut_koralle: "M182 252c1-6 1-11-1-15-7-3-13-10-14-18-1-5 4-7 7-3 1 5 4 9 8 11-2-6-2-13 0-19 1-5 7-5 8 0 1 4 1 9 0 13h1c-1-4-1-9 0-13 1-5 7-5 8 0 2 6 2 13 0 19 4-2 7-6 8-11 3-4 8-2 7 3-1 8-7 15-14 18-2 4-2 9-1 15Z",
  konsistenz_gallertartig: "M175 247c-9-2-11-13-5-19-5-9 2-20 12-19 4-9 17-9 21-1 9 1 13 11 8 19 6 7 1 18-8 19-4 6-14 7-19 2-4 1-7 0-9-1Z",
  f: "M167 240c2-15 10-28 23-33 9-3 18-1 22 5 3 5 0 10-6 11-7 1-12 4-15 11-3 8-8 13-15 12-5-1-9-2-9-6Z",
  sonstige_gattung: "M190 198c8 6 14 18 14 32 0 4-2 6-5 7v15h-18v-15c-3-1-5-3-5-7 0-14 6-26 14-32ZM190 210c-3 6-4 13-4 20 0 2 1 2 2 2v13h4v-13c1 0 2 0 2-2 0-7-1-14-4-20Z",
};

// Tag → Element, das bei Auswahl eingeblendet wird (.is-visible).
const overlays = {
  lamellen_eng: ".gill-dense",
  lamellen_gegabelt: ".gill-forks",
  lamellen_gesaegt: ".gill-serration",
  frei: ".gill-free",
  herablaufend: ".stem-decurrent",
  genattert: ".stem-pattern",
  hut_gerieft: ".surface-grooves",
  flocken: ".surface-speckles",
  hut_hygrophan: ".surface-hygro",
  haut_schmierig: ".surface-gloss",
  ring: ".stem-ring",
  netz: ".stem-net",
  knollig: ".stem-bulb",
  wurzel: ".stem-roots",
  stiel_hohl: ".stem-hollow",
  milch: ".flesh-milk",
  bruechig: ".flesh-crack",
  faserig: ".flesh-fiber-pattern",
  bueschelig: '.substrate[data-substrate="bueschel"]',
};
const substrateLayers = { auf_boden: "erde", holz: "holz", auf_wiese: "wiese" };

// Diese Tags werden als Klasse has-<tag> an #mushroom gesetzt; das CSS
// blendet damit Oberflächen der Silhouetten ein und hebt sie über den Untergrund.
const classTags = [...surfaceTags, "bueschelig", ...substrateTags];

const $ = (selector) => document.querySelector(selector);

export function updateMushroom(tags) {
  const mushroom = $("#mushroom");
  if (!mushroom) return;
  const capColor = activeFrom(colorTags(""), tags);
  const undersideColor = activeFrom(colorTags("lr_"), tags);
  const stemColor = activeFrom(colorTags("stiel_"), tags);
  const fleshColor = activeFrom(colorTags("fl_"), tags);
  const capFill = capColor ? colorFor(capColor) : "url(#capShade)";
  const capD = capPaths[activeFrom(capShapes, tags)] || capPaths.sonstige_form;

  // Gruppe: Hutpilz-Zeichnung oder eine der Silhouetten ([data-body]).
  mushroom.dataset.group = activeFrom(groupTags, tags) || "";
  const bodyGroup = mushroom.dataset.group !== "hutpilz" ? mushroom.dataset.group : "";
  mushroom.classList.toggle("is-body", Boolean(bodyGroup));
  for (const tag of classTags) mushroom.classList.toggle(`has-${tag}`, tags.has(tag));
  if (bodyGroup) document.querySelectorAll(".mini-body").forEach((use) => use.setAttribute("href", `#art-${bodyGroup}`));
  mushroom.style.setProperty("--body", capFill);

  // Hut
  $("#cap-shape").setAttribute("d", capD);
  $(".cap-fill").style.fill = capFill;
  $(".cap-funnel").classList.toggle("is-visible", tags.has("trichter"));

  // Unterseite
  const underside = $('[data-part="underside"]');
  underside.style.setProperty("--underside", undersideColor ? colorFor(undersideColor, "lr_") : "#e8dfcb");
  underside.dataset.underside = activeFrom(undersideTypes, tags) || "lamellen";
  $(".stem-ridges").classList.toggle("is-visible", underside.dataset.underside === "leisten");

  // Stiel
  document.querySelectorAll(".stem-fill, .stem-tint").forEach((node) => { node.style.fill = stemColor ? colorFor(stemColor, "stiel_") : ""; });

  // Schnitt
  $("#flesh-shape").setAttribute("d", fleshShapes[bodyGroup || "hutpilz"]);
  $(".flesh-core").style.fill = fleshColor ? colorFor(fleshColor, "fl_") : "#eee7d7";

  for (const [tag, selector] of Object.entries(overlays)) $(selector).classList.toggle("is-visible", tags.has(tag));
  const substrate = activeFrom(substrateTags, tags);
  for (const [tag, layer] of Object.entries(substrateLayers)) {
    $(`.substrate[data-substrate="${layer}"]`).classList.toggle("is-visible", substrate === tag);
  }

  // Mini-Hutpilze beim Büschel übernehmen Hutform und Farben.
  const stemFill = stemColor ? colorFor(stemColor, "stiel_") : "url(#stemShade)";
  document.querySelectorAll(".mini-cap").forEach((path) => { path.setAttribute("d", capD); path.setAttribute("fill", capFill); });
  document.querySelectorAll(".mini-stem").forEach((path) => path.setAttribute("fill", stemFill));

  describeParts(tags);
}

// Tooltip und Screenreader-Text jedes Teils nennen die gewählten Merkmale.
function describeParts(tags) {
  document.querySelectorAll(".mushroom-part").forEach((part) => {
    const name = partNames[part.dataset.part];
    const sections = partChoices[part.dataset.part] || [];
    const selected = [...tags].filter((tag) => sections.some((section) => section.tags.includes(tag))).map(labelFor);
    const title = selected.length ? `${name}: ${selected.join(", ")}` : `${name} wählen`;
    part.setAttribute("title", title);
    part.setAttribute("aria-label", title);
    part.classList.toggle("has-selection", selected.length > 0);
  });
}
