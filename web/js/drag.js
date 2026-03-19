// Drag-and-drop for board cards: between columns and within-column reordering.
// Works with jfyne/live by clicking hidden trigger buttons wired by live.js.
(function () {
  var draggingCard = null; // DOM element being dragged
  var draggingSourceColumn = null;

  // === CONTEXT MENU ===
  var ctxMenu = null;
  var ctxTargetCard = null;

  function buildContextMenu() {
    var el = document.createElement("div");
    el.id = "card-context-menu";
    el.className = "card-context-menu";
    el.setAttribute("role", "menu");
    document.body.appendChild(el);
    return el;
  }

  function hideContextMenu() {
    if (ctxMenu) {
      ctxMenu.classList.remove("visible");
      ctxMenu.innerHTML = "";
    }
    ctxTargetCard = null;
  }

  function makeItem(icon, label, danger, onClick) {
    var btn = document.createElement("button");
    btn.className = "ctx-item" + (danger ? " ctx-danger" : "");
    btn.setAttribute("role", "menuitem");
    btn.innerHTML = '<span class="ctx-icon">' + icon + "</span>" + label;
    btn.addEventListener("click", function (e) {
      e.stopPropagation();
      hideContextMenu();
      onClick();
    });
    return btn;
  }

  // === QUICK EDIT OVERLAY ===
  var qeOverlay = null;

  function hideQuickEdit() {
    if (qeOverlay) {
      qeOverlay.remove();
      qeOverlay = null;
    }
    hideContextMenu();
  }

  function showQuickEdit(card) {
    hideQuickEdit();

    var cardRect = card.getBoundingClientRect();
    var titleEl = card.querySelector(".card-header h4");
    var bodyEl = card.querySelector(".card-body");
    var tagsEl = card.querySelectorAll(".card-tags .tag");
    var colIdx = card.dataset.colIdx;
    var cardIdx = card.dataset.cardIdx;
    var slug = window.location.pathname.replace(/^\/board\//, "");

    var currentTitle = titleEl ? titleEl.textContent.trim() : "";
    var currentBody = bodyEl ? bodyEl.textContent.trim() : "";
    var currentTags = Array.from(tagsEl).map(function (t) { return t.textContent.trim(); }).join(", ");

    // Build overlay
    var overlay = document.createElement("div");
    overlay.className = "quick-edit-overlay";
    overlay.style.left = cardRect.left + "px";
    overlay.style.top = cardRect.top + "px";
    overlay.style.width = cardRect.width + "px";
    overlay.style.minHeight = cardRect.height + "px";

    var titleInput = document.createElement("textarea");
    titleInput.className = "qe-title";
    titleInput.value = currentTitle;
    titleInput.rows = 2;
    overlay.appendChild(titleInput);

    if (currentTags || true) {
      var tagsInput = document.createElement("input");
      tagsInput.className = "qe-tags";
      tagsInput.type = "text";
      tagsInput.placeholder = "Tags (comma-separated)";
      tagsInput.value = currentTags;
      overlay.appendChild(tagsInput);
    }

    var bodyInput = document.createElement("textarea");
    bodyInput.className = "qe-body";
    bodyInput.placeholder = "Description (optional)";
    bodyInput.value = currentBody;
    bodyInput.rows = 2;
    overlay.appendChild(bodyInput);

    var actions = document.createElement("div");
    actions.className = "qe-actions";

    var saveBtn = document.createElement("button");
    saveBtn.className = "btn-primary btn-small";
    saveBtn.textContent = "Save";
    saveBtn.addEventListener("click", function () {
      if (window.Live) {
        window.Live.send("edit-card", {
          col_idx: colIdx,
          card_idx: cardIdx,
          title: titleInput.value.trim(),
          body: bodyInput.value.trim(),
          tags: tagsInput.value.trim(),
          name: slug,
        });
      }
      hideQuickEdit();
    });

    var cancelBtn = document.createElement("button");
    cancelBtn.className = "btn-small";
    cancelBtn.textContent = "Cancel";
    cancelBtn.addEventListener("click", hideQuickEdit);

    actions.appendChild(saveBtn);
    actions.appendChild(cancelBtn);
    overlay.appendChild(actions);

    document.body.appendChild(overlay);
    qeOverlay = overlay;

    // Focus title
    titleInput.focus();
    titleInput.selectionStart = titleInput.value.length;

    // Build context menu to the right of the overlay
    buildContextMenuForCard(card, cardRect);
  }

  function buildContextMenuForCard(card, cardRect) {
    if (!ctxMenu) ctxMenu = buildContextMenu();
    ctxTargetCard = card;
    ctxMenu.innerHTML = "";

    var isCompleted = card.classList.contains("completed");

    var completeBtn = card.querySelector('[live-click="toggle-complete"]');
    if (completeBtn) {
      ctxMenu.appendChild(makeItem("✓", isCompleted ? "Mark Incomplete" : "Complete", false, function () {
        hideQuickEdit();
        completeBtn.click();
      }));
    }

    var moveTriggers = Array.from(card.querySelectorAll(".move-trigger[data-target]"));
    if (moveTriggers.length > 0) {
      var sep = document.createElement("div");
      sep.className = "ctx-separator";
      ctxMenu.appendChild(sep);

      var label = document.createElement("div");
      label.className = "ctx-submenu-label";
      label.textContent = "Move to";
      ctxMenu.appendChild(label);

      var sub = document.createElement("div");
      sub.className = "ctx-submenu";
      moveTriggers.forEach(function (trigger) {
        var target = trigger.dataset.target;
        sub.appendChild(makeItem("→", target, false, function () {
          hideQuickEdit();
          trigger.click();
        }));
      });
      ctxMenu.appendChild(sub);
    }

    var deleteBtn = card.querySelector('[live-click="delete-card"]');
    if (deleteBtn) {
      var sep2 = document.createElement("div");
      sep2.className = "ctx-separator";
      ctxMenu.appendChild(sep2);
      ctxMenu.appendChild(makeItem("🗑", "Delete", true, function () {
        hideQuickEdit();
        deleteBtn.click();
      }));
    }

    ctxMenu.classList.add("visible");

    // Position to the right of the card, aligned to its top
    var vw = window.innerWidth;
    var vh = window.innerHeight;
    var menuRect = ctxMenu.getBoundingClientRect();
    var left = cardRect.right + 8;
    if (left + menuRect.width > vw) left = cardRect.left - menuRect.width - 8;
    var top = cardRect.top;
    if (top + menuRect.height > vh) top = vh - menuRect.height - 8;
    ctxMenu.style.left = Math.max(0, left) + "px";
    ctxMenu.style.top = Math.max(0, top) + "px";
  }

  function showContextMenu(card, x, y) {
    if (!ctxMenu) ctxMenu = buildContextMenu();
    ctxTargetCard = card;
    ctxMenu.innerHTML = "";

    var isCompleted = card.classList.contains("completed");

    var completeBtn = card.querySelector('[live-click="toggle-complete"]');
    if (completeBtn) {
      ctxMenu.appendChild(makeItem("✓", isCompleted ? "Mark Incomplete" : "Complete", false, function () {
        completeBtn.click();
      }));
    }

    var moveTriggers = Array.from(card.querySelectorAll(".move-trigger[data-target]"));
    if (moveTriggers.length > 0) {
      var sep = document.createElement("div");
      sep.className = "ctx-separator";
      ctxMenu.appendChild(sep);
      var label = document.createElement("div");
      label.className = "ctx-submenu-label";
      label.textContent = "Move to";
      ctxMenu.appendChild(label);
      var sub = document.createElement("div");
      sub.className = "ctx-submenu";
      moveTriggers.forEach(function (trigger) {
        var target = trigger.dataset.target;
        sub.appendChild(makeItem("→", target, false, function () {
          trigger.click();
        }));
      });
      ctxMenu.appendChild(sub);
    }

    var deleteBtn = card.querySelector('[live-click="delete-card"]');
    if (deleteBtn) {
      var sep2 = document.createElement("div");
      sep2.className = "ctx-separator";
      ctxMenu.appendChild(sep2);
      ctxMenu.appendChild(makeItem("🗑", "Delete", true, function () {
        deleteBtn.click();
      }));
    }

    ctxMenu.classList.add("visible");
    var rect = ctxMenu.getBoundingClientRect();
    var vw = window.innerWidth;
    var vh = window.innerHeight;
    var left = x + rect.width > vw ? x - rect.width : x;
    var top = y + rect.height > vh ? y - rect.height : y;
    ctxMenu.style.left = Math.max(0, left) + "px";
    ctxMenu.style.top = Math.max(0, top) + "px";
  }

  // === COLUMN CONTEXT MENU ===
  var colCtxMenu = null;

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
    var slug = window.location.pathname.replace(/^\/board\//, "");

    colCtxMenu.appendChild(makeColItem("✏️", "Edit", false, function () {
      startColumnRename(btn, columnName, slug);
    }));

    colCtxMenu.appendChild(makeColItem("🗑", "Delete", true, function () {
      if (window.confirm('Delete column "' + columnName + '" and all its cards?')) {
        if (window.Live) {
          window.Live.send("delete-column", { column_name: columnName, name: slug });
        }
      }
    }));

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
    return makeItem(icon, label, danger, function () {
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
      if (save && newName && newName !== currentName && window.Live) {
        window.Live.send("rename-column", { old_name: currentName, new_name: newName, name: slug });
      }
    }

    input.addEventListener("blur", function () { finish(true); });
    input.addEventListener("keydown", function (e) {
      if (e.key === "Enter") { e.preventDefault(); finish(true); }
      if (e.key === "Escape") { finish(false); }
    });
  }

  var assistantModalBackdrop = null;

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

  // === BOARD EDIT MODAL ===
  function showBoardEditModal(name, description) {
    var slug = window.location.pathname.replace(/^\/board\//, "");

    var backdrop = document.createElement("div");
    backdrop.className = "card-modal-backdrop";
    backdrop.addEventListener("click", function (e) {
      if (e.target === backdrop) backdrop.remove();
    });

    var modal = document.createElement("div");
    modal.className = "card-modal";
    modal.style.maxWidth = "480px";

    var closeBtn = document.createElement("button");
    closeBtn.className = "card-modal-close";
    closeBtn.innerHTML = "&times;";
    closeBtn.addEventListener("click", function () { backdrop.remove(); });
    modal.appendChild(closeBtn);

    var main = document.createElement("div");
    main.className = "card-modal-main";

    var hdr = document.createElement("div");
    hdr.className = "card-modal-section-header";
    hdr.innerHTML = '<span class="card-modal-section-icon">&#9998;</span> Edit Board';
    main.appendChild(hdr);

    var nameLabel = document.createElement("label");
    nameLabel.style.cssText = "display:block;font-size:var(--font-size-sm);color:var(--color-text-secondary);margin-top:12px;margin-bottom:4px;";
    nameLabel.textContent = "Board name";
    main.appendChild(nameLabel);

    var nameInput = document.createElement("input");
    nameInput.type = "text";
    nameInput.className = "card-modal-tags-input";
    nameInput.value = name;
    nameInput.style.width = "100%";
    main.appendChild(nameInput);

    var descLabel = document.createElement("label");
    descLabel.style.cssText = "display:block;font-size:var(--font-size-sm);color:var(--color-text-secondary);margin-top:12px;margin-bottom:4px;";
    descLabel.textContent = "Description";
    main.appendChild(descLabel);

    var descInput = document.createElement("textarea");
    descInput.className = "card-modal-body";
    descInput.placeholder = "Board description (optional)";
    descInput.value = description;
    descInput.rows = 3;
    main.appendChild(descInput);

    var saveRow = document.createElement("div");
    saveRow.className = "card-modal-save-row";

    var saveBtn = document.createElement("button");
    saveBtn.className = "btn-primary btn-small";
    saveBtn.textContent = "Save";
    saveBtn.addEventListener("click", function () {
      var newName = nameInput.value.trim();
      if (!newName) return;
      if (window.Live) {
        window.Live.send("update-board-meta", {
          board_name: newName,
          description: descInput.value.trim(),
          name: slug,
        });
      }
      backdrop.remove();
    });
    saveRow.appendChild(saveBtn);

    var cancelBtn = document.createElement("button");
    cancelBtn.className = "btn-small";
    cancelBtn.style.marginLeft = "8px";
    cancelBtn.textContent = "Cancel";
    cancelBtn.addEventListener("click", function () { backdrop.remove(); });
    saveRow.appendChild(cancelBtn);

    main.appendChild(saveRow);
    modal.appendChild(main);
    backdrop.appendChild(modal);
    document.body.appendChild(backdrop);
    nameInput.focus();
    nameInput.select();
  }

  function attachBoardEdit() {
    var btn = document.querySelector(".board-edit-btn");
    if (btn && !btn.dataset.editWired) {
      btn.dataset.editWired = "1";
      btn.addEventListener("click", function (e) {
        e.stopPropagation();
        showBoardEditModal(btn.dataset.boardName, btn.dataset.boardDescription || "");
      });
    }

    var titleEl = document.querySelector(".board-title");
    if (titleEl && !titleEl.dataset.dblWired) {
      titleEl.dataset.dblWired = "1";
      titleEl.addEventListener("dblclick", function (e) {
        e.stopPropagation();
        var b = document.querySelector(".board-edit-btn");
        showBoardEditModal(b ? b.dataset.boardName : titleEl.textContent.trim(), b ? b.dataset.boardDescription || "" : "");
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
        var slug = window.location.pathname.replace(/^\/board\//, "");
        startColumnRename(btn, btn.dataset.columnName, slug);
      });
    });
  }

  document.addEventListener("click", function (e) {
    if (colCtxMenu && colCtxMenu.classList.contains("visible") && !colCtxMenu.contains(e.target)) {
      hideColumnMenu();
    }
    if (qeOverlay && !qeOverlay.contains(e.target) && ctxMenu && !ctxMenu.contains(e.target)) {
      hideQuickEdit();
    } else if (!qeOverlay && ctxMenu && !ctxMenu.contains(e.target)) {
      hideContextMenu();
    }
  });

  document.addEventListener("keydown", function (e) {
    if (e.key === "Escape") hideQuickEdit();
  });

  // Re-attach context menu listeners on cards
  function attachContextMenu() {
    document.querySelectorAll(".card[data-card-idx]").forEach(function (card) {
      if (card.dataset.ctxWired) return;
      card.dataset.ctxWired = "1";
      card.addEventListener("contextmenu", function (e) {
        e.preventDefault();
        showQuickEdit(card);
      });
    });
  }

  function clearDropIndicators() {
    document.querySelectorAll(".drop-indicator").forEach(function (el) {
      el.remove();
    });
  }

  function showDropIndicator(zone, beforeCard) {
    clearDropIndicators();
    var indicator = document.createElement("div");
    indicator.className = "drop-indicator";
    if (beforeCard) {
      zone.insertBefore(indicator, beforeCard);
    } else {
      // Append before the hidden end-trigger button
      var endTrigger = zone.querySelector(".reorder-end-trigger");
      if (endTrigger) {
        zone.insertBefore(indicator, endTrigger);
      } else {
        zone.appendChild(indicator);
      }
    }
  }

  // Find which card in the zone the cursor is above (for insertion point).
  // Returns the card element to insert before, or null to append at end.
  function getInsertionTarget(zone, clientY, excludeEl) {
    var cards = Array.from(zone.querySelectorAll(".card[data-card-idx]")).filter(
      function (c) { return c !== excludeEl; }
    );
    for (var i = 0; i < cards.length; i++) {
      var rect = cards[i].getBoundingClientRect();
      if (clientY < rect.top + rect.height / 2) {
        return cards[i];
      }
    }
    return null; // append to end
  }

  // === CARD DETAIL MODAL ===
  var cardModal = null;
  var cardModalBackdrop = null;
  var isDragging = false;

  function hideCardModal() {
    if (cardModalBackdrop) {
      cardModalBackdrop.remove();
      cardModalBackdrop = null;
      cardModal = null;
    }
  }

  function showCardModal(card) {
    hideCardModal();
    hideQuickEdit();

    var colIdx = card.dataset.colIdx;
    var cardIdx = card.dataset.cardIdx;
    var slug = window.location.pathname.replace(/^\/board\//, "");
    var title = card.dataset.cardTitle || "";
    var body = card.dataset.cardBody || "";
    var tags = card.dataset.cardTags || "";
    var assignee = card.dataset.cardAssignee || "";
    var priority = card.dataset.cardPriority || "";
    var due = card.dataset.cardDue || "";
    var completed = card.dataset.cardCompleted === "true";
    var columnName = card.dataset.cardColumn || "";

    // Backdrop
    var backdrop = document.createElement("div");
    backdrop.className = "card-modal-backdrop";
    backdrop.addEventListener("click", function (e) {
      if (e.target === backdrop) hideCardModal();
    });

    // Modal
    var modal = document.createElement("div");
    modal.className = "card-modal";

    // Close button
    var closeBtn = document.createElement("button");
    closeBtn.className = "card-modal-close";
    closeBtn.innerHTML = "&times;";
    closeBtn.addEventListener("click", hideCardModal);
    modal.appendChild(closeBtn);

    // Main content (left)
    var main = document.createElement("div");
    main.className = "card-modal-main";

    // Column pill
    var colPill = document.createElement("span");
    colPill.className = "card-modal-column-pill";
    colPill.textContent = columnName;
    main.appendChild(colPill);

    // Completed badge
    if (completed) {
      var doneBadge = document.createElement("span");
      doneBadge.className = "status-badge completed";
      doneBadge.innerHTML = "&#10003; Done";
      doneBadge.style.marginLeft = "8px";
      main.appendChild(doneBadge);
    }

    // Title
    var titleInput = document.createElement("textarea");
    titleInput.className = "card-modal-title";
    titleInput.value = title;
    titleInput.rows = 1;
    titleInput.addEventListener("input", function () {
      this.style.height = "auto";
      this.style.height = this.scrollHeight + "px";
    });
    main.appendChild(titleInput);

    // Action bar
    var actionBar = document.createElement("div");
    actionBar.className = "card-modal-action-bar";
    ["Labels", "Dates", "Checklist", "Members"].forEach(function (label) {
      var btn = document.createElement("button");
      btn.className = "card-modal-action-btn";
      var icons = { Labels: "🏷", Dates: "📅", Checklist: "☑", Members: "👤" };
      btn.innerHTML = '<span class="card-modal-action-icon">' + icons[label] + "</span> " + label;
      actionBar.appendChild(btn);
    });
    main.appendChild(actionBar);

    // Description section
    var descHeader = document.createElement("div");
    descHeader.className = "card-modal-section-header";
    descHeader.innerHTML = '<span class="card-modal-section-icon">≡</span> Description';
    main.appendChild(descHeader);

    var bodyInput = document.createElement("textarea");
    bodyInput.className = "card-modal-body";
    bodyInput.placeholder = "Add a more detailed description...";
    bodyInput.value = body;
    bodyInput.rows = 4;
    bodyInput.addEventListener("input", function () {
      this.style.height = "auto";
      this.style.height = this.scrollHeight + "px";
    });
    main.appendChild(bodyInput);

    // Tags section
    var tagsHeader = document.createElement("div");
    tagsHeader.className = "card-modal-section-header";
    tagsHeader.innerHTML = '<span class="card-modal-section-icon">🏷</span> Tags';
    main.appendChild(tagsHeader);

    var tagsInput = document.createElement("input");
    tagsInput.className = "card-modal-tags-input";
    tagsInput.type = "text";
    tagsInput.placeholder = "Tags (comma-separated)";
    tagsInput.value = tags;
    main.appendChild(tagsInput);

    // Save button
    var saveRow = document.createElement("div");
    saveRow.className = "card-modal-save-row";
    var saveBtn = document.createElement("button");
    saveBtn.className = "btn-primary btn-small";
    saveBtn.textContent = "Save";
    saveBtn.addEventListener("click", function () {
      if (window.Live) {
        window.Live.send("edit-card", {
          col_idx: colIdx,
          card_idx: cardIdx,
          title: titleInput.value.trim(),
          body: bodyInput.value.trim(),
          tags: tagsInput.value.trim(),
          name: slug,
        });
      }
      hideCardModal();
    });
    saveRow.appendChild(saveBtn);
    main.appendChild(saveRow);

    modal.appendChild(main);

    // Sidebar (right)
    var sidebar = document.createElement("div");
    sidebar.className = "card-modal-sidebar";

    var actionsLabel = document.createElement("div");
    actionsLabel.className = "card-modal-sidebar-label";
    actionsLabel.textContent = "Actions";
    sidebar.appendChild(actionsLabel);

    // Complete/Incomplete
    var completeBtn = card.querySelector('[live-click="toggle-complete"]');
    if (completeBtn || completed) {
      var toggleBtn = document.createElement("button");
      toggleBtn.className = "card-modal-sidebar-btn";
      toggleBtn.innerHTML = completed
        ? '<span class="card-modal-action-icon">↩</span> Mark Incomplete'
        : '<span class="card-modal-action-icon">✓</span> Complete';
      toggleBtn.addEventListener("click", function () {
        hideCardModal();
        var btn = card.querySelector('[live-click="toggle-complete"]');
        if (btn) btn.click();
        else if (window.Live) {
          window.Live.send("toggle-complete", { col_idx: colIdx, card_idx: cardIdx, name: slug });
        }
      });
      sidebar.appendChild(toggleBtn);
    }

    // Move to
    var moveTriggers = Array.from(card.querySelectorAll(".move-trigger[data-target]"));
    if (moveTriggers.length > 0) {
      var moveLabel = document.createElement("div");
      moveLabel.className = "card-modal-sidebar-sublabel";
      moveLabel.textContent = "Move to";
      sidebar.appendChild(moveLabel);

      moveTriggers.forEach(function (trigger) {
        var target = trigger.dataset.target;
        var moveBtn = document.createElement("button");
        moveBtn.className = "card-modal-sidebar-btn card-modal-sidebar-btn-move";
        moveBtn.innerHTML = '<span class="card-modal-action-icon">→</span> ' + target;
        moveBtn.addEventListener("click", function () {
          hideCardModal();
          trigger.click();
        });
        sidebar.appendChild(moveBtn);
      });
    }

    // Delete
    var deleteHidden = card.querySelector('[live-click="delete-card"]');
    if (deleteHidden) {
      var sep = document.createElement("div");
      sep.className = "card-modal-sidebar-sep";
      sidebar.appendChild(sep);

      var delBtn = document.createElement("button");
      delBtn.className = "card-modal-sidebar-btn card-modal-sidebar-btn-danger";
      delBtn.innerHTML = '<span class="card-modal-action-icon">🗑</span> Delete';
      delBtn.addEventListener("click", function () {
        hideCardModal();
        deleteHidden.click();
      });
      sidebar.appendChild(delBtn);
    }

    // Metadata display
    if (assignee || priority || due) {
      var metaSep = document.createElement("div");
      metaSep.className = "card-modal-sidebar-sep";
      sidebar.appendChild(metaSep);

      var metaLabel = document.createElement("div");
      metaLabel.className = "card-modal-sidebar-label";
      metaLabel.textContent = "Details";
      sidebar.appendChild(metaLabel);

      if (assignee) {
        var aEl = document.createElement("div");
        aEl.className = "card-modal-meta-item";
        aEl.innerHTML = "👤 " + assignee;
        sidebar.appendChild(aEl);
      }
      if (priority) {
        var pEl = document.createElement("div");
        pEl.className = "card-modal-meta-item";
        pEl.innerHTML = "⚡ " + priority;
        sidebar.appendChild(pEl);
      }
      if (due) {
        var dEl = document.createElement("div");
        dEl.className = "card-modal-meta-item";
        dEl.innerHTML = "📅 " + due;
        sidebar.appendChild(dEl);
      }
    }

    modal.appendChild(sidebar);
    backdrop.appendChild(modal);
    document.body.appendChild(backdrop);
    cardModalBackdrop = backdrop;
    cardModal = modal;

    // Auto-size title
    titleInput.style.height = "auto";
    titleInput.style.height = titleInput.scrollHeight + "px";
    titleInput.focus();
  }

  // Attach card click for modal (distinguish from drag)
  function attachCardClick() {
    document.querySelectorAll(".card[data-card-idx]").forEach(function (card) {
      if (card.dataset.modalWired) return;
      card.dataset.modalWired = "1";

      card.addEventListener("mousedown", function () {
        isDragging = false;
      });

      card.addEventListener("click", function (e) {
        if (isDragging) return;
        // Don't open modal if clicking a button or link
        if (e.target.closest("button, a, input, textarea, select")) return;
        showCardModal(card);
      });
    });
  }

  // Escape key for modal
  document.addEventListener("keydown", function (e) {
    if (e.key === "Escape" && cardModalBackdrop) {
      hideCardModal();
    }
  });

  var collapsedHeaderClickWired = false;
  function attachCollapsedColumnClick() {
    if (collapsedHeaderClickWired) return;
    collapsedHeaderClickWired = true;
    document.addEventListener("click", function (e) {
      var header = e.target.closest(".column-header");
      if (!header) return;
      var col = header.closest(".column");
      if (!col || !col.classList.contains("collapsed")) return;
      var btn = header.querySelector(".column-collapse-btn");
      if (btn && e.target !== btn) btn.click();
    });
  }

  function attach() {
    attachCollapsedColumnClick();
    attachContextMenu();
    attachCardClick();
    attachColumnMenus();
    attachBoardEdit();
    // Cards: draggable
    document.querySelectorAll(".card[draggable]").forEach(function (card) {
      if (card.dataset.dragWired) return;
      card.dataset.dragWired = "1";

      card.addEventListener("dragstart", function (e) {
        isDragging = true;
        draggingCard = card;
        var zone = card.closest(".cards[data-column]");
        draggingSourceColumn = zone ? zone.dataset.column : null;
        card.classList.add("dragging");
        e.dataTransfer.effectAllowed = "move";
        e.dataTransfer.setData("text/plain", card.dataset.colIdx + ":" + card.dataset.cardIdx);
      });

      card.addEventListener("dragend", function () {
        card.classList.remove("dragging");
        clearDropIndicators();
        document.querySelectorAll(".cards.drag-over").forEach(function (el) {
          el.classList.remove("drag-over");
        });
        draggingCard = null;
        draggingSourceColumn = null;
      });
    });

    // Drop zones: .cards containers
    document.querySelectorAll(".cards[data-column]").forEach(function (zone) {
      if (zone.dataset.dropWired) return;
      zone.dataset.dropWired = "1";

      zone.addEventListener("dragover", function (e) {
        e.preventDefault();
        e.dataTransfer.dropEffect = "move";

        var targetColumn = zone.dataset.column;

        if (draggingSourceColumn === targetColumn) {
          // Within-column: show insertion indicator
          zone.classList.remove("drag-over");
          var beforeCard = getInsertionTarget(zone, e.clientY, draggingCard);
          showDropIndicator(zone, beforeCard);
        } else {
          // Cross-column: highlight the whole zone
          clearDropIndicators();
          zone.classList.add("drag-over");
        }
      });

      zone.addEventListener("dragleave", function (e) {
        if (!zone.contains(e.relatedTarget)) {
          zone.classList.remove("drag-over");
          clearDropIndicators();
        }
      });

      zone.addEventListener("drop", function (e) {
        e.preventDefault();
        zone.classList.remove("drag-over");

        var data = e.dataTransfer.getData("text/plain");
        var targetColumn = zone.dataset.column;
        if (!data || !targetColumn) {
          clearDropIndicators();
          return;
        }

        var parts = data.split(":");
        var srcColIdx = parts[0];
        var srcCardIdx = parts[1];

        // Find the dragged card element
        var card = document.querySelector('.card[data-col-idx="' + srcColIdx + '"][data-card-idx="' + srcCardIdx + '"]');
        if (!card) {
          clearDropIndicators();
          return;
        }

        var sourceZone = card.closest(".cards[data-column]");
        var sourceColumn = sourceZone ? sourceZone.dataset.column : null;

        if (sourceColumn === targetColumn) {
          // Within-column reorder
          var indicator = zone.querySelector(".drop-indicator");
          var beforeCard = null;
          if (indicator) {
            var next = indicator.nextElementSibling;
            while (next && !next.classList.contains("card")) {
              next = next.nextElementSibling;
            }
            if (next && next.classList.contains("card")) {
              beforeCard = next;
            }
          }
          clearDropIndicators();

          var beforeIdx = beforeCard ? beforeCard.dataset.cardIdx : "-1";

          // Skip if card didn't actually move
          var prevSibling = card.previousElementSibling;
          while (prevSibling && !prevSibling.classList.contains("card")) {
            prevSibling = prevSibling.previousElementSibling;
          }
          var nextSibling = card.nextElementSibling;
          while (nextSibling && !nextSibling.classList.contains("card")) {
            nextSibling = nextSibling.nextElementSibling;
          }
          if (
            (beforeCard === null && nextSibling === null) ||
            (beforeCard && nextSibling && beforeCard.dataset.cardIdx === nextSibling.dataset.cardIdx)
          ) {
            return; // no-op
          }

          var slug = window.location.pathname.replace(/^\/board\//, "");
          if (window.Live) {
            window.Live.send("reorder-card", {
              col_idx: srcColIdx,
              card_idx: srcCardIdx,
              before_idx: beforeIdx,
              column: targetColumn,
              name: slug,
            });
          }
        } else {
          // Cross-column move
          clearDropIndicators();
          var btn = card.querySelector('.move-trigger[data-target="' + targetColumn + '"]');
          if (btn) btn.click();
        }
      });
    });
  }

  // Attach on load
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", attach);
  } else {
    attach();
  }

  // Re-attach after LiveView re-renders
  document.addEventListener("live:updated", attach);

  // Fallback: MutationObserver for DOM changes
  new MutationObserver(function () {
    requestAnimationFrame(attach);
  }).observe(document.body, { childList: true, subtree: true });
})();
