// LiveBoard: Cmd+K command palette.
(function () {
  var cmdPaletteOpen = false;
  var cmdPaletteItems = [];
  var cmdPaletteIdx = 0;
  var cmdPaletteSelectableCount = 0;

  function buildPaletteItems(query) {
    var data = window.__cmdPaletteData || { boards: [], activeSlug: "" };
    var path = window.location.pathname;
    var items = [];

    // Add boards (skip the active one)
    data.boards.forEach(function (b) {
      var url = "/board/" + b.slug;
      if (b.slug === data.activeSlug && path === url) return;
      items.push({ icon: b.icon || "\u2630", name: b.name, url: url });
    });

    // Add fixed pages with separator
    var hasFixed = false;
    if (path !== "/") {
      if (!hasFixed) { items.push({ separator: true }); hasFixed = true; }
      items.push({ icon: "\uD83C\uDFE0", name: "All Boards", url: "/" });
    }
    if (path !== "/settings") {
      if (!hasFixed) { items.push({ separator: true }); hasFixed = true; }
      items.push({ icon: "\u2699\uFE0F", name: "Settings", url: "/settings" });
    }

    if (query) {
      var q = query.toLowerCase();
      items = items.filter(function (it) {
        return !it.separator && it.name.toLowerCase().indexOf(q) !== -1;
      });
    }

    return items;
  }

  function renderPaletteList() {
    var list = document.getElementById("cmdPaletteList");
    if (!list) return;
    list.innerHTML = "";
    var selectableIdx = 0;
    cmdPaletteItems.forEach(function (it) {
      if (it.separator) {
        var sep = document.createElement("div");
        sep.className = "cmd-palette-separator";
        list.appendChild(sep);
        return;
      }
      var idx = selectableIdx++;
      var div = document.createElement("div");
      div.className = "cmd-palette-item" + (idx === cmdPaletteIdx ? " active" : "");
      div.setAttribute("data-idx", idx);
      div.innerHTML =
        '<span class="cmd-palette-item-icon">' + it.icon + "</span>" +
        '<span class="cmd-palette-item-name">' + it.name + "</span>";
      div.addEventListener("click", function () {
        window.location.href = it.url;
      });
      div.addEventListener("mouseenter", function () {
        cmdPaletteIdx = idx;
        updatePaletteActive();
      });
      list.appendChild(div);
    });
    cmdPaletteSelectableCount = selectableIdx;
  }

  function updatePaletteActive() {
    var list = document.getElementById("cmdPaletteList");
    if (!list) return;
    var items = list.querySelectorAll(".cmd-palette-item");
    for (var i = 0; i < items.length; i++) {
      var isActive = parseInt(items[i].getAttribute("data-idx")) === cmdPaletteIdx;
      items[i].classList.toggle("active", isActive);
      if (isActive) items[i].scrollIntoView({ block: "nearest" });
    }
  }

  function openCmdPalette() {
    var overlay = document.getElementById("cmdPalette");
    if (!overlay) return;
    cmdPaletteIdx = 0;
    cmdPaletteItems = buildPaletteItems("");
    overlay.style.display = "";
    renderPaletteList();
    var input = document.getElementById("cmdPaletteInput");
    if (input) {
      input.value = "";
      input.focus();
    }
    cmdPaletteOpen = true;
  }

  function closeCmdPalette() {
    var overlay = document.getElementById("cmdPalette");
    if (overlay) overlay.style.display = "none";
    cmdPaletteOpen = false;
  }

  document.addEventListener("keydown", function (e) {
    // Cmd+K or Ctrl+K to toggle
    if ((e.metaKey || e.ctrlKey) && e.key === "k") {
      e.preventDefault();
      if (cmdPaletteOpen) closeCmdPalette();
      else openCmdPalette();
      return;
    }

    if (!cmdPaletteOpen) return;

    if (e.key === "Escape") {
      e.preventDefault();
      closeCmdPalette();
      return;
    }
    if (e.key === "ArrowDown") {
      e.preventDefault();
      if (cmdPaletteSelectableCount > 0) {
        cmdPaletteIdx = (cmdPaletteIdx + 1) % cmdPaletteSelectableCount;
        updatePaletteActive();
      }
      return;
    }
    if (e.key === "ArrowUp") {
      e.preventDefault();
      if (cmdPaletteSelectableCount > 0) {
        cmdPaletteIdx = (cmdPaletteIdx - 1 + cmdPaletteSelectableCount) % cmdPaletteSelectableCount;
        updatePaletteActive();
      }
      return;
    }
    if (e.key === "Enter") {
      e.preventDefault();
      var selectable = cmdPaletteItems.filter(function (it) { return !it.separator; });
      if (selectable[cmdPaletteIdx]) {
        window.location.href = selectable[cmdPaletteIdx].url;
      }
      return;
    }
  });

  // Filter on input
  document.addEventListener("input", function (e) {
    if (e.target.id === "cmdPaletteInput") {
      cmdPaletteIdx = 0;
      cmdPaletteItems = buildPaletteItems(e.target.value);
      renderPaletteList();
    }
  });

  // Close on backdrop click
  document.addEventListener("click", function (e) {
    if (cmdPaletteOpen && e.target.id === "cmdPalette") {
      closeCmdPalette();
    }
  });
})();
