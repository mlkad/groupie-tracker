(function () {
  const root = document.documentElement;
  const themeToggle = document.getElementById("themeToggle");

  if (themeToggle) {
    themeToggle.addEventListener("click", function () {
      const next = root.getAttribute("data-theme") === "dark" ? "light" : "dark";
      root.setAttribute("data-theme", next);
      try {
        localStorage.setItem("gt-theme", next);
      } catch (e) {}
    });
  }

  document.querySelectorAll(".group-head").forEach(function (head) {
    head.addEventListener("click", function () {
      const open = head.getAttribute("aria-expanded") === "true";
      head.setAttribute("aria-expanded", open ? "false" : "true");
    });
  });

  const filters = document.getElementById("filters");
  if (!filters) return;

  const toggle = document.getElementById("filtersToggle");
  const backdrop = document.getElementById("filtersBackdrop");
  const closeBtn = document.getElementById("filtersClose");

  function setPanel(open) {
    filters.classList.toggle("open", open);
    document.body.classList.toggle("filters-open", open);
    if (backdrop) backdrop.hidden = !open;
    if (toggle) toggle.setAttribute("aria-expanded", open ? "true" : "false");
  }

  if (toggle) toggle.addEventListener("click", function () { setPanel(true); });
  if (closeBtn) closeBtn.addEventListener("click", function () { setPanel(false); });
  if (backdrop) backdrop.addEventListener("click", function () { setPanel(false); });

  document.addEventListener("keydown", function (e) {
    if (e.key === "Escape") setPanel(false);
  });

  window.addEventListener("resize", function () {
    if (window.innerWidth > 1000) setPanel(false);
  });

  document.querySelectorAll(".range-slider").forEach(function (wrap) {
    const fill = wrap.querySelector(".range-fill");
    const inputs = wrap.querySelectorAll('input[type="range"]');
    if (inputs.length !== 2) return;

    const fromEl = inputs[0];
    const toEl = inputs[1];

    function paint() {
      const min = +fromEl.min;
      const max = +fromEl.max;
      const span = max - min || 1;
      const from = Math.min(+fromEl.value, +toEl.value);
      const to = Math.max(+fromEl.value, +toEl.value);

      fill.style.left = ((from - min) / span) * 100 + "%";
      fill.style.width = ((to - from) / span) * 100 + "%";
      return { from: from, to: to };
    }

    function clamp(el) {
      if (el === fromEl && +fromEl.value > +toEl.value) toEl.value = fromEl.value;
      if (el === toEl && +toEl.value < +fromEl.value) fromEl.value = toEl.value;

      const r = paint();
      wrap.dispatchEvent(new CustomEvent("rangechange", { detail: r, bubbles: true }));
    }

    fromEl.addEventListener("input", function () { clamp(fromEl); });
    toEl.addEventListener("input", function () { clamp(toEl); });

    wrap.addEventListener("pointerdown", function (e) {
      const rect = wrap.getBoundingClientRect();
      const min = +fromEl.min;
      const max = +fromEl.max;
      const at = min + ((e.clientX - rect.left) / rect.width) * (max - min);
      const nearFrom = Math.abs(at - +fromEl.value) <= Math.abs(at - +toEl.value);

      fromEl.style.zIndex = nearFrom ? 4 : 3;
      toEl.style.zIndex = nearFrom ? 3 : 4;
    });

    paint();
  });

  function link(rangeKey, fromInput, toInput) {
    const wrap = document.querySelector('[data-range="' + rangeKey + '"]');
    if (!wrap || !fromInput || !toInput) return;

    const inputs = wrap.querySelectorAll('input[type="range"]');
    const fromRange = inputs[0];
    const toRange = inputs[1];

    wrap.addEventListener("rangechange", function (e) {
      fromInput.value = e.detail.from;
      toInput.value = e.detail.to;
    });

    fromInput.min = toInput.min = fromRange.min;
    fromInput.max = toInput.max = fromRange.max;

    fromInput.addEventListener("change", function () {
      const v = parseInt(fromInput.value, 10);
      if (isNaN(v)) {
        fromInput.value = "";
        return;
      }
      fromRange.value = Math.min(Math.max(v, +fromRange.min), +toRange.value);
      fromRange.dispatchEvent(new Event("input"));
    });

    toInput.addEventListener("change", function () {
      const v = parseInt(toInput.value, 10);
      if (isNaN(v)) {
        toInput.value = "";
        return;
      }
      toRange.value = Math.max(Math.min(v, +toRange.max), +fromRange.value);
      toRange.dispatchEvent(new Event("input"));
    });
  }

  link("creation", document.getElementById("creationFrom"), document.getElementById("creationTo"));
  link("album", document.getElementById("albumFrom"), document.getElementById("albumTo"));

  const membersWrap = document.querySelector('[data-range="members"]');
  const membersFromLabel = document.getElementById("membersFromLabel");
  const membersToLabel = document.getElementById("membersToLabel");

  if (membersWrap && membersFromLabel && membersToLabel) {
    membersWrap.addEventListener("rangechange", function (e) {
      const max = +membersWrap.querySelector('input[type="range"]').max;
      membersFromLabel.textContent = e.detail.from;
      membersToLabel.textContent = e.detail.to >= max ? e.detail.to + "+" : e.detail.to;
    });
  }

  const locationSearch = document.getElementById("locationSearch");
  const locationList = document.getElementById("locationList");
  const showMore = document.getElementById("showMoreLocations");
  const COLLAPSED = 10;

  function locationItems() {
    return locationList ? Array.prototype.slice.call(locationList.querySelectorAll("li")) : [];
  }

  function syncLocations() {
    if (!locationList) return;

    const q = (locationSearch && locationSearch.value || "").trim().toLowerCase();
    const expanded = showMore && showMore.classList.contains("expanded");
    let shown = 0;

    locationItems().forEach(function (li) {
      if (li.classList.contains("no-match")) return;

      const hit = li.textContent.trim().toLowerCase().indexOf(q) !== -1;
      const overflow = !q && !expanded && shown >= COLLAPSED;

      li.hidden = !hit || overflow;
      if (hit && !overflow) shown++;
    });

    if (showMore) {
      const total = locationItems().filter(function (li) {
        return !li.classList.contains("no-match");
      }).length;
      showMore.hidden = Boolean(q) || total <= COLLAPSED;
      showMore.firstChild.textContent = expanded ? "Show less " : "Show more ";
    }
  }

  if (locationSearch) locationSearch.addEventListener("input", syncLocations);

  if (showMore) {
    showMore.addEventListener("click", function () {
      showMore.classList.toggle("expanded");
      syncLocations();
    });
  }

  syncLocations();


  const DEBOUNCE_MS = 300;
  let pending = null;
  let timer = null;

  const pageInput = document.getElementById("pageInput");

  function resetPage() {
    if (pageInput) pageInput.value = "1";
  }

  function applyFilters(immediate) {
    clearTimeout(timer);
    timer = setTimeout(request, immediate ? 0 : DEBOUNCE_MS);
  }

  function goToPage(page) {
    if (pageInput) pageInput.value = String(page);
    applyFilters(true);
  }

  const PAGE_WINDOW = 5;

  function pageRange(current, total) {
    if (total <= PAGE_WINDOW) {
      const all = [];
      for (let p = 1; p <= total; p++) all.push(p);
      return all;
    }

    let start = current - Math.floor(PAGE_WINDOW / 2);
    start = Math.max(1, Math.min(start, total - PAGE_WINDOW + 1));

    const range = [];
    for (let p = start; p < start + PAGE_WINDOW; p++) range.push(p);
    return range;
  }

  function renderPageNumbers(current, total) {
    const wrap = document.getElementById("pageNumbers");
    if (!wrap) return;

    wrap.innerHTML = "";

    pageRange(current, total).forEach(function (p) {
      const btn = document.createElement("button");
      btn.type = "button";
      btn.className = "page-num" + (p === current ? " current" : "");
      btn.textContent = String(p);
      btn.setAttribute("aria-label", "Page " + p);
      if (p === current) btn.setAttribute("aria-current", "page");
      btn.addEventListener("click", function () {
        if (p !== current) goToPage(p);
      });

      wrap.appendChild(btn);
    });
  }

  function bindPagination() {
    const nav = document.getElementById("pagination");
    if (!nav) return;

    const current = parseInt(nav.dataset.current, 10) || 1;
    const total = parseInt(nav.dataset.total, 10) || 1;
    const prev = nav.querySelector(".page-prev");
    const next = nav.querySelector(".page-next");

    if (prev) prev.addEventListener("click", function () {
      if (current > 1) goToPage(current - 1);
    });
    if (next) next.addEventListener("click", function () {
      if (current < total) goToPage(current + 1);
    });

    renderPageNumbers(current, total);
  }

  function request() {
    const results = document.getElementById("results");
    if (!results) return;

    const query = new URLSearchParams(new FormData(filters)).toString();
    const url = filters.action + (query ? "?" + query : "");

    if (pending) pending.abort();
    pending = new AbortController();

    results.classList.add("is-loading");

    fetch(url, { signal: pending.signal, headers: { "X-Requested-With": "fetch" } })
      .then(function (res) {
        if (!res.ok) throw new Error(res.status);
        return res.text();
      })
      .then(function (html) {
        const fresh = new DOMParser()
          .parseFromString(html, "text/html")
          .getElementById("results");

        if (fresh) {
          results.replaceWith(fresh);
          bindPagination();
        }
        history.replaceState(null, "", url);
      })
      .catch(function (err) {
        if (err.name !== "AbortError") results.classList.remove("is-loading");
      });
  }

  filters.addEventListener("submit", function (e) {
    e.preventDefault();
    resetPage();
    applyFilters(true);
    setPanel(false);
  });

  filters.addEventListener("input", function (e) {
    if (e.target.type === "range" || e.target.type === "number") {
      resetPage();
      applyFilters(false);
    }
  });

  filters.addEventListener("change", function (e) {
    if (e.target.type === "checkbox") {
      resetPage();
      applyFilters(true);
    }
  });

  document.addEventListener("change", function (e) {
    if (e.target.id === "sortSelect") {
      resetPage();
      applyFilters(true);
    }
  });

  bindPagination();

  const resetLink = filters.querySelector(".reset-btn");
  if (resetLink) {
    resetLink.addEventListener("click", function (e) {
      e.preventDefault();

      filters.querySelectorAll('input[type="checkbox"]').forEach(function (el) {
        el.checked = false;
      });

      filters.querySelectorAll(".range-slider").forEach(function (wrap) {
        const pair = wrap.querySelectorAll('input[type="range"]');
        if (pair.length !== 2) return;

        pair[0].value = pair[0].min;
        pair[1].value = pair[1].max;
        pair[0].dispatchEvent(new Event("input", { bubbles: true }));
        pair[1].dispatchEvent(new Event("input", { bubbles: true }));
      });

      filters.querySelectorAll('input[type="number"]').forEach(function (el) {
        el.value = "";
      });

      const sortSelect = document.getElementById("sortSelect");
      if (sortSelect) sortSelect.value = "name-asc";

      if (locationSearch) locationSearch.value = "";
      if (showMore) showMore.classList.remove("expanded");
      syncLocations();

      applyFilters(true);
      setPanel(false);
    });
  }
})();
