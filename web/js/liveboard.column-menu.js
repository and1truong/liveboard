// LiveBoard: column context menu, rename, assistant modal, board title editor.
(function () {
  var LB = window.LB;
  var colCtxMenu = null;
  var assistantModalBackdrop = null;

  function hideColumnMenu() {
    if (colCtxMenu) {
      colCtxMenu.classList.remove("visible");
      colCtxMenu.innerHTML = "";
      colCtxMenu._forBtn = null;
    }
  }

  function showColumnMenu(btn) {
    if (!colCtxMenu) {
      colCtxMenu = document.createElement("div");
      colCtxMenu.id = "column-context-menu";
      colCtxMenu.className = "card-context-menu";
      colCtxMenu.setAttribute("role", "menu");
      document.body.appendChild(colCtxMenu);
    }

    hideColumnMenu();
    colCtxMenu._forBtn = btn;

    var columnName = btn.dataset.columnName;
    var slug = decodeURIComponent(window.location.pathname.replace(/^\/board\//, ""));

    colCtxMenu.appendChild(makeColItem("✏️", "Edit", false, function () {
      startColumnRename(btn, columnName, slug);
    }));

    colCtxMenu.appendChild(makeColItem("🗑", "Delete", true, function () {
      if (window.confirm('Delete column "' + columnName + '" and all its cards?')) {
        htmx.ajax('POST', '/board/' + encodeURIComponent(slug) + '/columns/delete', {
          values: { column_name: columnName, name: slug, version: LB.getBoardVersion() },
          target: '#board-content',
          swap: 'innerHTML'
        });
      }
    }));

    // Sort submenu
    var colEl = btn.closest(".column");
    var colIdx = colEl ? Array.from(colEl.parentNode.children).filter(function (c) { return c.classList.contains("column") && !c.classList.contains("add-column"); }).indexOf(colEl) : -1;

    if (colIdx >= 0) {
      var sortSep = document.createElement("div");
      sortSep.className = "ctx-separator";
      colCtxMenu.appendChild(sortSep);

      var sortLabel = document.createElement("div");
      sortLabel.className = "ctx-submenu-label";
      sortLabel.textContent = "Sort";
      colCtxMenu.appendChild(sortLabel);

      var sortSub = document.createElement("div");
      sortSub.className = "ctx-submenu";
      sortSub.appendChild(makeColItem("🔤", "By Name", false, function () {
        htmx.ajax('POST', '/board/' + encodeURIComponent(slug) + '/columns/sort', { values: { col_idx: String(colIdx), sort_by: "name", name: slug, version: LB.getBoardVersion() }, target: '#board-content', swap: 'innerHTML' });
      }));
      sortSub.appendChild(makeColItem("⚡", "By Priority", false, function () {
        htmx.ajax('POST', '/board/' + encodeURIComponent(slug) + '/columns/sort', { values: { col_idx: String(colIdx), sort_by: "priority", name: slug, version: LB.getBoardVersion() }, target: '#board-content', swap: 'innerHTML' });
      }));
      sortSub.appendChild(makeColItem("📅", "By Due Date", false, function () {
        htmx.ajax('POST', '/board/' + encodeURIComponent(slug) + '/columns/sort', { values: { col_idx: String(colIdx), sort_by: "due", name: slug, version: LB.getBoardVersion() }, target: '#board-content', swap: 'innerHTML' });
      }));
      colCtxMenu.appendChild(sortSub);
    }

    var sep = document.createElement("div");
    sep.className = "ctx-separator";
    colCtxMenu.appendChild(sep);

    var assistLabel = document.createElement("div");
    assistLabel.className = "ctx-submenu-label";
    assistLabel.textContent = "Assistant";
    colCtxMenu.appendChild(assistLabel);

    var sub = document.createElement("div");
    sub.className = "ctx-submenu";
    sub.appendChild(makeColItem("📋", "Summary", false, function () {
      showAssistantModal(columnName, slug, "summary");
    }));
    sub.appendChild(makeColItem("⚙️", "Custom instruction", false, function () {
      showAssistantModal(columnName, slug, "custom");
    }));
    colCtxMenu.appendChild(sub);

    colCtxMenu.classList.add("visible");

    // Position below the button
    var btnRect = btn.getBoundingClientRect();
    var vw = window.innerWidth;
    var vh = window.innerHeight;
    var menuRect = colCtxMenu.getBoundingClientRect();
    var left = btnRect.left;
    var top = btnRect.bottom + 4;
    if (left + menuRect.width > vw) left = btnRect.right - menuRect.width;
    if (top + menuRect.height > vh) top = btnRect.top - menuRect.height - 4;
    colCtxMenu.style.left = Math.max(0, left) + "px";
    colCtxMenu.style.top = Math.max(0, top) + "px";
  }

  function makeColItem(icon, label, danger, onClick) {
    return LB.makeItem(icon, label, danger, function () {
      hideColumnMenu();
      onClick();
    });
  }

  function startColumnRename(btn, currentName, slug) {
    var column = btn.closest(".column");
    if (!column) return;
    var header = column.querySelector(".column-header");
    var h3 = header && header.querySelector("h3");
    if (!h3) return;

    var input = document.createElement("input");
    input.type = "text";
    input.value = currentName;
    input.style.cssText = [
      "flex:1",
      "min-width:0",
      "padding:1px 4px",
      "font-size:var(--font-size-xs)",
      "font-weight:700",
      "text-transform:uppercase",
      "letter-spacing:0.06em",
      "color:var(--color-text-secondary)",
      "background:var(--color-surface)",
      "border:1px solid var(--color-accent)",
      "border-radius:var(--radius-sm)",
      "font-family:var(--font-sans)",
    ].join(";");

    h3.replaceWith(input);
    input.focus();
    input.select();

    var saved = false;
    function finish(save) {
      if (saved) return;
      saved = true;
      var newName = input.value.trim();
      input.replaceWith(h3);
      if (save && newName && newName !== currentName) {
        htmx.ajax('POST', '/board/' + encodeURIComponent(slug) + '/columns/rename', { values: { old_name: currentName, new_name: newName, name: slug, version: LB.getBoardVersion() }, target: '#board-content', swap: 'innerHTML' });
      }
    }

    input.addEventListener("blur", function () { finish(true); });
    input.addEventListener("keydown", function (e) {
      if (e.key === "Enter") { e.preventDefault(); finish(true); }
      if (e.key === "Escape") { finish(false); }
    });
  }

  function showAssistantModal(columnName, slug, mode) {
    if (assistantModalBackdrop) {
      assistantModalBackdrop.remove();
      assistantModalBackdrop = null;
    }

    var backdrop = document.createElement("div");
    backdrop.className = "card-modal-backdrop";
    backdrop.addEventListener("click", function (e) {
      if (e.target === backdrop) { backdrop.remove(); assistantModalBackdrop = null; }
    });

    var modal = document.createElement("div");
    modal.className = "card-modal";
    modal.style.maxWidth = "480px";

    var closeBtn = document.createElement("button");
    closeBtn.className = "card-modal-close";
    closeBtn.innerHTML = "&times;";
    closeBtn.addEventListener("click", function () { backdrop.remove(); assistantModalBackdrop = null; });
    modal.appendChild(closeBtn);

    var main = document.createElement("div");
    main.className = "card-modal-main";

    var hdr = document.createElement("div");
    hdr.className = "card-modal-section-header";
    hdr.innerHTML = '<span class="card-modal-section-icon">⚙️</span> Assistant — ' + columnName;
    main.appendChild(hdr);

    var desc = document.createElement("p");
    desc.style.cssText = "font-size:var(--font-size-sm);color:var(--color-text-secondary);margin-top:8px;";
    desc.textContent = mode === "summary"
      ? "Generate a summary of all cards in this column:"
      : "Custom instruction for the assistant:";
    main.appendChild(desc);

    var textarea = document.createElement("textarea");
    textarea.className = "card-modal-body";
    textarea.placeholder = mode === "summary"
      ? "E.g. Summarize as a priority list..."
      : "E.g. Write a status update based on these cards...";
    textarea.rows = 5;
    main.appendChild(textarea);

    var saveRow = document.createElement("div");
    saveRow.className = "card-modal-save-row";

    var runBtn = document.createElement("button");
    runBtn.className = "btn-primary btn-small";
    runBtn.textContent = "Run";
    runBtn.addEventListener("click", function () {
      // TODO: wire to AI backend
      backdrop.remove();
      assistantModalBackdrop = null;
    });
    saveRow.appendChild(runBtn);

    var cancelBtn = document.createElement("button");
    cancelBtn.className = "btn-small";
    cancelBtn.style.marginLeft = "8px";
    cancelBtn.textContent = "Cancel";
    cancelBtn.addEventListener("click", function () { backdrop.remove(); assistantModalBackdrop = null; });
    saveRow.appendChild(cancelBtn);

    main.appendChild(saveRow);
    modal.appendChild(main);
    backdrop.appendChild(modal);
    document.body.appendChild(backdrop);
    assistantModalBackdrop = backdrop;
    textarea.focus();
  }

  function attachBoardTitleDblClick() {
    var titleEl = document.querySelector(".board-title");
    if (titleEl && !titleEl.dataset.dblWired) {
      titleEl.dataset.dblWired = "1";
      titleEl.addEventListener("dblclick", function (e) {
        e.stopPropagation();
        var gearBtn = document.querySelector(".board-settings-btn");
        if (gearBtn) gearBtn.click();
      });
    }
  }

  function attachColumnMenus() {
    document.querySelectorAll(".column-menu-btn").forEach(function (btn) {
      if (btn.dataset.colMenuWired) return;
      btn.dataset.colMenuWired = "1";
      btn.addEventListener("click", function (e) {
        e.stopPropagation();
        if (colCtxMenu && colCtxMenu.classList.contains("visible") && colCtxMenu._forBtn === btn) {
          hideColumnMenu();
          return;
        }
        showColumnMenu(btn);
      });
    });

    document.querySelectorAll(".column-header h3").forEach(function (h3) {
      if (h3.dataset.dblWired) return;
      h3.dataset.dblWired = "1";
      h3.addEventListener("dblclick", function (e) {
        e.stopPropagation();
        var btn = h3.closest(".column-header").querySelector(".column-menu-btn");
        if (!btn) return;
        var slug = decodeURIComponent(window.location.pathname.replace(/^\/board\//, ""));
        startColumnRename(btn, btn.dataset.columnName, slug);
      });
    });
  }

  // Expose for cross-module use
  LB.hideColumnMenu = hideColumnMenu;
  LB.attachColumnMenus = attachColumnMenus;
  LB.attachBoardTitleDblClick = attachBoardTitleDblClick;

  // Expose internals needed by global dismiss handler
  LB._colCtxMenuContains = function (el) { return colCtxMenu && colCtxMenu.contains(el); };
  LB._colCtxMenuVisible = function () { return colCtxMenu && colCtxMenu.classList.contains("visible"); };
})();
