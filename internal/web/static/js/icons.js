// Kleine Symbole für die Optionen im Auswahl-Deck (28×18 Einheiten).

const svg = (content, extraClass = "") => `<svg class="substrate-option-icon ${extraClass}" viewBox="0 0 28 18" aria-hidden="true">${content}</svg>`;

// Querschnitt Hut + Unterseite: für Lamellen, Röhren, Stacheln, Leisten.
function undersideIcon(tag) {
  const cap = '<path class="option-cap" d="M2 9C4 4 9 2 14 2s10 2 12 7Z"/>';
  const band = '<path class="option-underside" d="M2 9c6 2 18 2 24 0l-2 4c-5 2-15 2-20 0Z"/>';
  const icons = {
    lamellen: `${cap}${band}<path d="M5 10.2l1.3 2.8M8 10.8l.8 2.7M11 11.1l.4 2.7M14 11.2v2.8M17 11.1l-.4 2.7M20 10.8l-.8 2.7M23 10.2l-1.3 2.8"/>`,
    roehren: `${cap}${band}<path class="option-pores" d="M6 10.9h0m3 .6h0m3 .3h0m4 0h0m3-.3h0m3-.6h0M7.5 12.6h0m3 .4h0m3 .1h0m3-.1h0m3-.4h0"/>`,
    stachel: `${cap}<path class="option-underside" d="M2 9c6 2 18 2 24 0l-1.6 2.6c-5 1.6-15.8 1.6-20.8 0Z"/><path class="option-underside" d="M4 11.5l1 3 1-3ZM7 12.1l1 3.6 1-3.6ZM10 12.5l1 4.2 1-4.2ZM13 12.6l1 4.5 1-4.5ZM16 12.5l1 4.2 1-4.2ZM19 12.1l1 3.6 1-3.6ZM22 11.5l1 3 1-3Z"/>`,
    leisten: `${cap}<path class="option-underside" d="M11.2 14h5.6l-.6 3.6h-4.4Z"/>${band}<path class="ridge-fan" d="M6.5 13.3V12.1l-1.4-1.6M6.5 12.1l1.4-1.6M11.5 14.0V12.8l-1.4-1.7M11.5 12.8l1.4-1.7M16.5 14.0V12.8l-1.4-1.7M16.5 12.8l1.4-1.7M21.5 13.3V12.1l-1.4-1.6M21.5 12.1l1.4-1.6M12.6 14.6l.2 2.4m1.2-2.4v2.6m1.4-2.6-.2 2.4"/>`,
  };
  return icons[tag] ? svg(icons[tag], "underside-option-icon") : "";
}

// Schnittbilder für Merkmale, die sich als Glyphe nicht erklären lassen.
function traitIcon(tag) {
  const cap = '<path class="option-cap" d="M3 8C5 3 9 2 14 2s9 1 11 6Z"/>';
  const wideCap = cap.replace("M3 8", "M2 9");
  const stem = '<path class="option-underside" d="M12 8h4v9h-4Z"/>';
  const band = '<path class="option-underside" d="M2 9c6 2 18 2 24 0l-2 4c-5 2-15 2-20 0Z"/>';
  const icons = {
    angewachsen: `${cap}${stem}<path class="option-gill" d="M4 8h8v4ZM24 8h-8v4Z"/>`,
    herablaufend: `${cap}${stem}<path class="option-gill" d="M4 8h8v8ZM24 8h-8v8Z"/>`,
    frei: `${cap}${stem}<path class="option-gill" d="M4 8h6l-.6 3.4ZM24 8h-6l.6 3.4Z"/>`,
    lamellen_eng: `${wideCap}${band}<path d="M4 10.2l.9 2.6M5.8 10.6l.7 2.6M7.6 10.9l.5 2.7M9.4 11.1l.4 2.7M11.2 11.2l.2 2.8M13 11.3v2.8M14.8 11.3v2.8M16.6 11.2l-.2 2.8M18.4 11.1l-.4 2.7M20.2 10.9l-.5 2.7M22 10.6l-.7 2.6M23.8 10.2l-.9 2.6"/>`,
    lamellen_gegabelt: `${wideCap}${band}<path d="M6.5 13.5V12l-1.3-1.6M6.5 12l1.2-1.6M11.5 14.2v-1.6l-1.3-1.6m1.3 1.6 1.2-1.6M16.5 14.2v-1.6l-1.3-1.6m1.3 1.6 1.2-1.6M21.5 13.5V12l-1.3-1.6M21.5 12l1.2-1.6"/>`,
    lamellen_gesaegt: `${wideCap}<path class="option-underside" d="M2 9c6 2 18 2 24 0l-2 4-1.5-.4-1.5 .8-1.5-.5-1.5.8-1.5-.4-1.5.6-1.5-.6-1.5.6-1.5-.6-1.5.6-1.5-.4-1.5.4-1.5-.6-1.5.4Z"/>`,
    genattert: '<path class="option-underside" d="M10 1h8l1 16h-10Z"/><path d="M10 4l2-1.4 2 1.4 2-1.4 2 1.4M9.8 8l2.1-1.4 2.1 1.4 2.1-1.4 2.1 1.4M9.6 12l2.2-1.4 2.2 1.4 2.2-1.4 2.2 1.4"/>',
    konsistenz_hart: '<path class="option-bark" d="M2 1h6v17H2Z"/><path class="option-cap" d="M8 6c8 0 16 1 18 4-4 2-12 2-18 1Z"/><path d="M10 8.5c5 0 10 .5 14 1.6"/>',
  };
  return icons[tag] ? svg(icons[tag], "underside-option-icon") : "";
}

function substrateIcon(tag) {
  const bed = '<path class="option-bed" d="M2 15c6-2 18-2 24 0v3H2Z"/>';
  const icons = {
    bueschelig: ["", `${bed}<path class="cluster-cap" d="M3 10c.5-3 2-5 4-5s3.5 2 4 5c-2 .7-6 .7-8 0Zm8 1c.5-4 2-7 4-7s3.5 3 4 7c-2 .7-6 .7-8 0Zm8-1c.5-3 2-5 4-5s3.5 2 4 5c-2 .7-6 .7-8 0Z"/><path d="M6 10v5m7-4v4m7-5v5"/>`],
    holz: ["substrate-option-wood", `${bed}<rect x="2" y="8" width="24" height="7" rx="2"/><path d="M5 10h13m-8 2h11m-2-4c4 0 5 5 1 6"/>`],
    auf_boden: ["substrate-option-earth", `${bed}<path class="option-earth-surface" d="M2 12c6-2 18-2 24 0v4H2Z"/><path d="M7 13h2m5-2h2m4 3h2"/>`],
    auf_wiese: ["substrate-option-grass", `${bed}<path class="option-grass-turf" d="M2 12c6-2 18-2 24 0v4H2Z"/><path d="M4 13 6 7m3 6V5m3 8 2-7m3 7 1-5m3 5 2-7m3 7 1-5"/><circle cx="15" cy="5" r="1.5"/>`],
  };
  return icons[tag] ? svg(icons[tag][1], icons[tag][0]) : "";
}

// Fallback: einfache Glyphe.
const glyphs = { spitz: "△", schirm: "⌒", kugel: "◠", trichter: "▽", f: "◖", sonstige_form: "⌒", lamellen: "≋", roehren: "○", stachel: "⋮", hut_gerieft: "╱", hut_hygrophan: "◒", flocken: "⁘", haut_schmierig: "◌", ring: "◉", netz: "▦", knollig: "⬭", wurzel: "♧", stiel_hohl: "∥", milch: "◉", bruechig: "⌁", faserig: "╱", bueschelig: "♧", holz: "▰", auf_wiese: "♣" };

// Symbol einer Deck-Option: Farbfeld, Zeichnung oder Glyphe.
export function optionIcon(tag, part, swatchColor) {
  if (swatchColor) return `<span class="swatch" style="--swatch:${swatchColor}"></span>`;
  const drawn = traitIcon(tag) || (part === "growth" ? substrateIcon(tag) : part === "underside" ? undersideIcon(tag) : "");
  return drawn || `<span class="choice-icon" aria-hidden="true">${glyphs[tag] || "•"}</span>`;
}
