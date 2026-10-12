(function () {
  const el = document.getElementById("map");
  const note = document.getElementById("mapNote");
  const concerts = window.gtConcerts || [];

  if (!el || typeof L === "undefined") return;

  const points = concerts.filter(function (c) {
    return c.found;
  });

  const skipped = concerts.length - points.length;
  if (skipped > 0 && note) {
    note.textContent =
      skipped === 1
        ? "1 location could not be placed on the map."
        : skipped + " locations could not be placed on the map.";
    note.hidden = false;
  }

  if (points.length === 0) {
    el.classList.add("map-empty");
    el.textContent = "No concert locations could be placed on the map.";
    return;
  }

  const map = L.map(el, { scrollWheelZoom: false });

  const tiles = {
    light: "https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png",
    dark: "https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png",
  };

  const root = document.documentElement;
  let layer = null;

  function currentTheme() {
    return root.getAttribute("data-theme") === "dark" ? "dark" : "light";
  }

  function drawTiles() {
    if (layer) map.removeLayer(layer);
    layer = L.tileLayer(tiles[currentTheme()], {
      attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors',
      maxZoom: 18,
    }).addTo(map);
  }

  drawTiles();

  new MutationObserver(drawTiles).observe(root, {
    attributes: true,
    attributeFilter: ["data-theme"],
  });

  const bounds = [];

  points.forEach(function (c) {
    const popup = document.createElement("div");
    const location = document.createElement("strong");
    location.className = "popup-loc";
    location.textContent = c.location;
    popup.appendChild(location);

    if (c.dates && c.dates.length > 0) {
      const dates = document.createElement("ul");
      dates.className = "popup-dates";

      c.dates.forEach(function (date) {
        const item = document.createElement("li");
        item.textContent = date;
        dates.appendChild(item);
      });

      popup.appendChild(dates);
    }

    L.marker([c.lat, c.lon])
      .addTo(map)
      .bindPopup(popup);

    bounds.push([c.lat, c.lon]);
  });

  if (bounds.length === 1) {
    map.setView(bounds[0], 6);
  } else {
    map.fitBounds(bounds, { padding: [40, 40] });
  }

  map.on("focus", function () {
    map.scrollWheelZoom.enable();
  });
  map.on("blur", function () {
    map.scrollWheelZoom.disable();
  });
})();
