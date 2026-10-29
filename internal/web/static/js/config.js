// Merkmals-Vokabular des Pilz-Labs. Die Tag-Namen stammen aus
// filter_tags.jsonl (123pilzsuche.de) bzw. aus den abgeleiteten Tags in
// internal/catalog/field.go (hutpilz, monat_*, wald_*, baum_*).

export const COLORS = ["weiß", "gelb", "braun", "schwarz", "rot", "orange", "blau", "gruen", "rosa", "grau"];
const colorHex = { "weiß": "#eeeade", gelb: "#e5c64d", braun: "#8b5e3c", schwarz: "#333332", rot: "#bd4b45", orange: "#d98740", blau: "#6d94a7", gruen: "#718b62", rosa: "#d69da5", grau: "#999b96" };
export const sporeColorHex = { "sp_weiß": "#f2efe4", sp_gelb: "#d9b936", sp_rot: "#bd4b45", sp_braun: "#8b5e3c", sp_violett: "#846497", sp_schwarz: "#333332" };

// Farb-Tags tragen je nach Körperteil ein Präfix: "rot" (Hut), "lr_rot"
// (Lamellen/Röhren), "stiel_rot", "fl_rot" (Fleisch).
export const colorTags = (prefix) => COLORS.map((color) => `${prefix}${color}`);

export function colorFor(tag, prefix = "") {
  return colorHex[prefix ? tag.replace(prefix, "") : tag] || "#b9b2a2";
}

// Hauptgruppe wie auf 123pilzsuche.de; "hutpilz" ist abgeleitet (Lamellen, Röhren, Stacheln oder Leisten).
export const groupTags = ["hutpilz", "kugel", "hut_koralle", "konsistenz_gallertartig", "f", "sonstige_gattung"];
export const DEFAULT_GROUP = "hutpilz";
export const capShapes = ["spitz", "schirm", "trichter", "sonstige_form"];
export const surfaceTags = ["hut_gerieft", "hut_hygrophan", "flocken", "haut_schmierig", "konsistenz_hart"];
export const undersideTypes = ["lamellen", "roehren", "stachel", "leisten"];
export const substrateTags = ["auf_boden", "auf_wiese", "holz"];
const fleshTraits = ["milch", "bruechig", "faserig"];

export const monthTags = Array.from({ length: 12 }, (_, index) => `monat_${index + 1}`);
export const currentMonthTag = monthTags[new Date().getMonth()];

export const partNames = { cap: "Hut", underside: "Unterseite", stem: "Stiel", flesh: "Fruchtfleisch", growth: "Untergrund und Wuchsform", body: "Fruchtkörper" };

// Optionen, die beim Antippen eines Pilzteils erscheinen.
// group "multi" = frei kombinierbar; jede andere group ist exklusiv
// innerhalb ihres Teils (eine neue Hutfarbe ersetzt die alte).
export const partChoices = {
  cap: [
    { title: "Hutfarbe", kind: "color", prefix: "", group: "colors", tags: colorTags("") },
    { title: "Hutform", kind: "icon", group: "shape", tags: capShapes },
    { title: "Oberfläche & Konsistenz · kombinierbar", kind: "icon", group: "multi", tags: surfaceTags },
  ],
  underside: [
    { title: "Unterseitenform", kind: "icon", group: "type", tags: undersideTypes },
    { title: "Farbe der Unterseite", kind: "color", prefix: "lr_", group: "colors", tags: colorTags("lr_") },
    { title: "Ansatz am Stiel", kind: "icon", group: "ansatz", tags: ["angewachsen", "herablaufend", "frei"] },
    { title: "Lamellen-Details · kombinierbar", kind: "icon", group: "multi", tags: ["lamellen_eng", "lamellen_gegabelt", "lamellen_gesaegt"] },
  ],
  stem: [
    { title: "Stielfarbe", kind: "color", prefix: "stiel_", group: "colors", tags: colorTags("stiel_") },
    { title: "Stieldetails · kombinierbar", kind: "icon", group: "multi", tags: ["ring", "netz", "knollig", "wurzel", "stiel_hohl", "genattert"] },
  ],
  flesh: [
    { title: "Farbe", kind: "color", prefix: "fl_", group: "colors", tags: colorTags("fl_") },
    { title: "Merkmale · kombinierbar", kind: "icon", group: "multi", tags: fleshTraits },
  ],
  growth: [
    { title: "Untergrund", kind: "icon", group: "substrate", tags: substrateTags },
    { title: "Wuchsform", kind: "icon", group: "multi", tags: ["bueschelig"] },
  ],
  // Pilze ohne Hut (Bovist, Koralle, …) haben nur einen Fruchtkörper.
  body: [
    { title: "Farbe des Fruchtkörpers", kind: "color", prefix: "", group: "colors", tags: colorTags("") },
    { title: "Oberfläche & Konsistenz · kombinierbar", kind: "icon", group: "multi", tags: surfaceTags },
    { title: "Fleischfarbe (Schnitt)", kind: "color", prefix: "fl_", group: "fleshColors", tags: colorTags("fl_") },
    { title: "Fleisch · kombinierbar", kind: "icon", group: "multi", tags: fleshTraits },
  ],
};

// Exklusive Gruppen je Teil, abgeleitet aus partChoices, plus die
// Chip-Gruppen außerhalb der Zeichnung (data-trait-group im Template).
export const exclusiveGroups = {
  ...Object.fromEntries(Object.entries(partChoices).map(([part, sections]) => [
    part,
    Object.fromEntries(sections.filter((section) => section.group !== "multi").map((section) => [section.group, new Set(section.tags)])),
  ])),
  gruppe: { gruppe: new Set(groupTags) },
  edibility: { edibility: new Set(["hervorragend", "gut", "schlecht", "giftig", "tot"]) },
  spore: { spore: new Set(Object.keys(sporeColorHex)) },
  saison: { saison: new Set(monthTags) },
};

// Anzeigenamen kommen vom Server (#facet-labels), damit sie mit den Chips übereinstimmen.
const facetLabels = new Map([...document.querySelectorAll("#facet-labels [data-tag]")].map((node) => [node.dataset.tag, node.dataset.label]));
const customLabels = { leisten: "Leisten" };

export function labelFor(tag) {
  return facetLabels.get(tag) || customLabels[tag] || tag.replaceAll("_", " ");
}
