const searchInput = document.querySelector("#species-search");
const minSlider = document.querySelector("#diameter-min");
const maxSlider = document.querySelector("#diameter-max");
const diameterValue = document.querySelector("#diameter-value");
let reloadTimer;

function scheduleFilterReload() {
  clearTimeout(reloadTimer);
  reloadTimer = setTimeout(() => {
    const params = new URLSearchParams();
    const query = searchInput?.value.trim();
    if (query) params.set("q", query);
    const selected = [...document.querySelectorAll('.facet-option input[name="f"]:checked')].map((input) => input.value);
    if (selected.length) params.set("f", selected.join(","));
    if (minSlider && maxSlider) {
      if (minSlider.value !== minSlider.min) params.set("dmin", minSlider.value);
      if (maxSlider.value !== maxSlider.max) params.set("dmax", maxSlider.value);
    }
    const currentLetter = new URLSearchParams(window.location.search).get("letter");
    if (currentLetter) params.set("letter", currentLetter);
    window.location.href = `/?${params.toString()}`;
  }, 150);
}

function updateDiameterLabel() {
  if (!minSlider || !maxSlider || !diameterValue) return;
  diameterValue.textContent = minSlider.value === minSlider.min && maxSlider.value === maxSlider.max
    ? "alle Werte"
    : `${minSlider.value} – ${maxSlider.value} cm`;
}

searchInput?.addEventListener("input", scheduleFilterReload);
document.querySelectorAll('.facet-option input[name="f"]').forEach((input) => {
  input.addEventListener("change", scheduleFilterReload);
});

minSlider?.addEventListener("input", () => {
  if (Number(minSlider.value) > Number(maxSlider.value)) maxSlider.value = minSlider.value;
  updateDiameterLabel();
  scheduleFilterReload();
});
maxSlider?.addEventListener("input", () => {
  if (Number(maxSlider.value) < Number(minSlider.value)) minSlider.value = maxSlider.value;
  updateDiameterLabel();
  scheduleFilterReload();
});

document.querySelector("#reset-filters")?.addEventListener("click", () => {
  window.location.href = "/";
});

document.addEventListener("keydown", (event) => {
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
    event.preventDefault();
    searchInput?.focus();
  }
});

const lightbox = document.querySelector("#lightbox");
if (lightbox) {
  const imageButtons = [...document.querySelectorAll(".image-open")];
  const lightboxImage = lightbox.querySelector(".lightbox-image");
  const counter = lightbox.querySelector(".lightbox-count");
  let currentImage = 0;
  let returnFocus;

  function showImage(index) {
    currentImage = (index + imageButtons.length) % imageButtons.length;
    const source = imageButtons[currentImage].querySelector("img");
    lightboxImage.src = source.currentSrc || source.src;
    lightboxImage.alt = source.alt;
    counter.textContent = `${currentImage + 1} / ${imageButtons.length}`;
  }

  function openLightbox(index) {
    returnFocus = document.activeElement;
    showImage(index);
    lightbox.classList.add("is-open");
    lightbox.setAttribute("aria-hidden", "false");
    document.body.classList.add("lightbox-active");
    lightbox.querySelector(".lightbox-close").focus();
  }

  function closeLightbox() {
    lightbox.classList.remove("is-open");
    lightbox.setAttribute("aria-hidden", "true");
    document.body.classList.remove("lightbox-active");
    returnFocus?.focus();
  }

  imageButtons.forEach((button, index) => button.addEventListener("click", () => openLightbox(index)));
  lightbox.querySelector(".lightbox-close").addEventListener("click", closeLightbox);
  lightbox.querySelector(".lightbox-prev").addEventListener("click", () => showImage(currentImage - 1));
  lightbox.querySelector(".lightbox-next").addEventListener("click", () => showImage(currentImage + 1));
  lightbox.addEventListener("click", (event) => {
    if (event.target === lightbox) closeLightbox();
  });
  document.addEventListener("keydown", (event) => {
    if (!lightbox.classList.contains("is-open")) return;
    if (event.key === "Escape") closeLightbox();
    if (event.key === "ArrowLeft") showImage(currentImage - 1);
    if (event.key === "ArrowRight") showImage(currentImage + 1);
  });
}
