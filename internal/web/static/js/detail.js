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
  let lastSwipe = 0; // Zeitpunkt der letzten Wischgeste

  function show(index) {
    current = (index + imageButtons.length) % imageButtons.length;
    const button = imageButtons[current];
    const source = button.querySelector("img");
    // Erst die (geladene) Vorschau zeigen, dann das große Original nachladen.
    // Offline oder ohne Original bleibt die Vorschau stehen.
    lightboxImage.src = source.currentSrc || source.src;
    lightboxImage.alt = source.alt;
    counter.textContent = `${current + 1} / ${imageButtons.length}`;
    const full = button.dataset.full;
    if (!full) return;
    const shown = current;
    const loader = new Image();
    loader.onload = () => { if (current === shown) lightboxImage.src = full; };
    loader.src = full;
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
  lightbox.addEventListener("click", (event) => {
    if (Date.now() - lastSwipe < 400) return; // Klick nach einer Wischgeste schließt nicht
    if (event.target === lightbox) close();
  });
  initSwipe();
  document.addEventListener("keydown", (event) => {
    if (!lightbox.classList.contains("is-open")) return;
    if (event.key === "Escape") close();
    if (event.key === "ArrowLeft") show(current - 1);
    if (event.key === "ArrowRight") show(current + 1);
  });

  // Wischen (Handy): waagerecht blättern, das Bild folgt dabei dem Finger.
  // Mit zwei Fingern oder bei hineingezoomter Seite (Pinch-Zoom) wird nicht geblättert.
  function initSwipe() {
    let startX = 0;
    let startY = 0;
    let deltaX = 0;
    let tracking = false;
    const zoomed = () => (window.visualViewport?.scale ?? 1) > 1.01;
    const settle = () => {
      lightboxImage.classList.remove("is-dragging");
      lightboxImage.style.transform = "";
    };
    lightbox.addEventListener("touchstart", (event) => {
      tracking = event.touches.length === 1 && !zoomed();
      if (!tracking) return settle();
      startX = event.touches[0].clientX;
      startY = event.touches[0].clientY;
      deltaX = 0;
    }, { passive: true });
    lightbox.addEventListener("touchmove", (event) => {
      if (!tracking) return;
      if (event.touches.length !== 1) {
        tracking = false;
        return settle();
      }
      deltaX = event.touches[0].clientX - startX;
      const deltaY = event.touches[0].clientY - startY;
      if (Math.abs(deltaY) > Math.abs(deltaX)) return settle();
      lightboxImage.classList.add("is-dragging");
      lightboxImage.style.transform = `translateX(${deltaX}px)`;
    }, { passive: true });
    lightbox.addEventListener("touchend", (event) => {
      if (!tracking) return;
      tracking = false;
      // Weg aus dem Endpunkt berechnen: touchmove kommt nicht auf jedem Gerät zuverlässig an.
      deltaX = event.changedTouches[0].clientX - startX;
      const deltaY = event.changedTouches[0].clientY - startY;
      settle();
      if (imageButtons.length < 2 || Math.abs(deltaX) < 50 || Math.abs(deltaX) < Math.abs(deltaY) * 1.5) return;
      lastSwipe = Date.now();
      const step = deltaX < 0 ? 1 : -1;
      show(current + step);
      // Neues Bild von der Seite hereinschieben, aus der gewischt wurde.
      lightboxImage.classList.add("is-dragging");
      lightboxImage.style.transform = `translateX(${step * 60}px)`;
      requestAnimationFrame(() => requestAnimationFrame(settle));
    });
    lightbox.addEventListener("touchcancel", () => {
      tracking = false;
      settle();
    });
  }
}
