(function () {
  const input = document.getElementById("q");
  const box = document.getElementById("suggestions");
  const clear = document.getElementById("clear");
  if (!input || !box) return;

  let timer = null;
  let controller = null;
  let items = [];
  let active = -1;

  function syncClear() {
    if (clear) clear.hidden = input.value.length === 0;
  }

  function hide() {
    box.hidden = true;
    box.innerHTML = "";
    items = [];
    active = -1;
  }

  function render(list) {
    if (!list || list.length === 0) {
      box.innerHTML = '<div class="empty-hint">No matches</div>';
      box.hidden = false;
      items = [];
      active = -1;
      return;
    }

    box.innerHTML = "";
    list.forEach(function (s) {
      const el = document.createElement("div");
      el.className = "suggestion";
      el.setAttribute("role", "option");

      const name = document.createElement("span");
      name.className = "s-name";
      name.textContent = s.name;
      el.appendChild(name);

      el.addEventListener("mousedown", function (e) {
        e.preventDefault();
        window.location.href = "/artist/" + s.id;
      });

      box.appendChild(el);
    });

    items = Array.prototype.slice.call(box.children);
    active = -1;
    box.hidden = false;
  }

  function highlight(next) {
    if (items.length === 0) return;
    if (active >= 0) items[active].classList.remove("active");
    active = (next + items.length) % items.length;
    items[active].classList.add("active");
    items[active].scrollIntoView({ block: "nearest" });
  }

  function fetchSuggestions(q) {
    if (controller) controller.abort();
    controller = new AbortController();

    fetch("/suggest?q=" + encodeURIComponent(q), { signal: controller.signal })
      .then(function (res) {
        if (!res.ok) throw new Error(res.status);
        return res.json();
      })
      .then(render)
      .catch(function (err) {
        if (err.name !== "AbortError") hide();
      });
  }

  input.addEventListener("input", function () {
    const q = input.value.trim();
    syncClear();
    clearTimeout(timer);

    if (q.length < 2) {
      hide();
      return;
    }

    timer = setTimeout(function () {
      fetchSuggestions(q);
    }, 180);
  });

  input.addEventListener("keydown", function (e) {
    if (box.hidden) return;

    if (e.key === "ArrowDown") {
      e.preventDefault();
      highlight(active + 1);
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      highlight(active - 1);
    } else if (e.key === "Enter") {
      if (active >= 0 && items[active]) {
        e.preventDefault();
        items[active].dispatchEvent(new MouseEvent("mousedown"));
      }
    } else if (e.key === "Escape") {
      hide();
    }
  });

  input.addEventListener("blur", function () {
    setTimeout(hide, 120);
  });

  input.addEventListener("focus", function () {
    const q = input.value.trim();
    if (q.length >= 2) fetchSuggestions(q);
  });

  if (clear) {
    clear.addEventListener("click", function () {
      input.value = "";
      syncClear();
      hide();
      input.focus();
    });
  }

  syncClear();
})();
