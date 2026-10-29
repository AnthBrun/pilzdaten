// Artseite (templates/detail.html): Bild-Lightbox und Zurück-Link.

// "Zurück" führt zu den eben gefilterten Treffern statt auf eine frische Startseite.
export function initBackLinks() {
  document.querySelectorAll("[data-back-link]").forEach((link) => link.addEventListener("click", (event) => {
    if (document.referrer && new URL(document.referrer).origin === location.origin && history.length > 1) {
      event.preventDefault();
      history.back();
    }
  }));
}

export function initLightbox() {
  const lightbox = document.querySelector("#lightbox");
  if (!lightbox) return;
  const imageButtons = [...document.querySelectorAll(".image-open")];
  const lightboxImage = lightbox.querySelector(".lightbox-image");
  const counter = lightbox.querySelector(".lightbox-count");
  let current = 0;
  let returnFocus;

  function show(index) {
    current = (index + imageButtons.length) % imageButtons.length;
    const source = imageButtons[current].querySelector("img");
    lightboxImage.src = source.currentSrc || source.src;
    lightboxImage.alt = source.alt;
    counter.textContent = `${current + 1} / ${imageButtons.length}`;
  }
  function open(index) {
    returnFocus = document.activeElement;
    show(index);
    lightbox.classList.add("is-open");
    lightbox.setAttribute("aria-hidden", "false");
    document.body.classList.add("lightbox-active");
    lightbox.querySelector(".lightbox-close").focus();
  }
  function close() {
    lightbox.classList.remove("is-open");
    lightbox.setAttribute("aria-hidden", "true");
    document.body.classList.remove("lightbox-active");
    returnFocus?.focus();
  }

  imageButtons.forEach((button, index) => button.addEventListener("click", () => open(index)));
  lightbox.querySelector(".lightbox-close").addEventListener("click", close);
  lightbox.querySelector(".lightbox-prev").addEventListener("click", () => show(current - 1));
  lightbox.querySelector(".lightbox-next").addEventListener("click", () => show(current + 1));
  lightbox.addEventListener("click", (event) => { if (event.target === lightbox) close(); });
  document.addEventListener("keydown", (event) => {
    if (!lightbox.classList.contains("is-open")) return;
    if (event.key === "Escape") close();
    if (event.key === "ArrowLeft") show(current - 1);
    if (event.key === "ArrowRight") show(current + 1);
  });
}
