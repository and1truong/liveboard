// LiveBoard: card context menu and quick-edit overlay.
(function () {
  var LB = window.LB;
  var ctxMenu = null;
  var ctxTargetCard = null;
  var qeOverlay = null;

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

  function makeDeleteItem(triggerBtn, beforeDelete) {
    var btn = document.createElement("button");
    btn.className = "ctx-item ctx-danger";
    btn.setAttribute("role", "menuitem");
    btn.innerHTML = '<span class="ctx-icon">🗑</span>Delete';
    var armed = false;
    btn.addEventListener("click", function (e) {
      e.stopPropagation();
      if (armed) {
        hideContextMenu();
        if (beforeDelete) beforeDelete();
        triggerBtn.click();
        return;
      }
      btn.disabled = true;
      btn.innerHTML = '<span class="ctx-icon">⏳</span>Deleting…';
      setTimeout(function () {
        armed = true;
        btn.disabled = false;
        btn.innerHTML = '<span class="ctx-icon">🗑</span>Confirm delete';
      }, 1000);
    });
    return btn;
  }

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
    var colIdx = card.dataset.colIdx;
    var cardIdx = card.dataset.cardIdx;
    var slug = decodeURIComponent(window.location.pathname.replace(/^\/board\//, ""));

    var currentTitle = card.dataset.cardTitle || "";
    var currentBody = card.dataset.cardBody || "";
    var currentTags = card.dataset.cardTags || "";
    var currentPriority = card.dataset.cardPriority || "";

    // Build overlay — in table mode, align to the card-content cell, not the full row
    var posRect = cardRect;
    var cardCell = card.querySelector(".table-cell-card");
    if (cardCell) {
      posRect = cardCell.getBoundingClientRect();
    }

    var overlay = document.createElement("div");
    overlay.className = "quick-edit-overlay";
    overlay.style.left = posRect.left + "px";
    overlay.style.top = cardRect.top + "px";
    overlay.style.width = posRect.width + "px";
    overlay.style.minHeight = cardRect.height + "px";

    var titleInput = document.createElement("textarea");
    titleInput.className = "qe-title";
    titleInput.value = currentTitle;
    titleInput.rows = 2;
    // New line trigger: determine if Enter submits or inserts newline
    titleInput.addEventListener("keydown", function (e) {
      var trigger = window.__lbNewLineTrigger ? window.__lbNewLineTrigger() : "shift-enter";
      if (trigger === "shift-enter") {
        // Enter submits (default)
        if (e.key === "Enter" && !e.shiftKey) {
          e.preventDefault();
          var sb = overlay.querySelector(".btn-primary");
          if (sb) sb.click();
        }
      } else {
        // Shift+Enter submits
        if (e.key === "Enter" && e.shiftKey) {
          e.preventDefault();
          var sb = overlay.querySelector(".btn-primary");
          if (sb) sb.click();
        }
      }
      if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        var sb = overlay.querySelector(".btn-primary");
        if (sb) sb.click();
        return;
      }
      if (e.key === "Escape") hideQuickEdit();
    });
    overlay.appendChild(titleInput);

    // Collect all unique tags from board
    var qeAllTags = [];
    document.querySelectorAll(".card[data-card-tags]").forEach(function (c) {
      (c.dataset.cardTags || "").split(",").forEach(function (s) {
        s = s.trim();
        if (s && qeAllTags.indexOf(s) === -1) qeAllTags.push(s);
      });
    });
    qeAllTags.sort(function (a, b) { return a.toLowerCase().localeCompare(b.toLowerCase()); });

    var qeTags = [];
    if (currentTags) {
      currentTags.split(",").forEach(function (s) {
        s = s.trim();
        if (s && qeTags.indexOf(s) === -1) qeTags.push(s);
      });
    }

    var qeTagsContainer = document.createElement("div");
    qeTagsContainer.className = "card-modal-tags-container qe-tags-container";

    var qeTagsInput = document.createElement("input");
    qeTagsInput.className = "card-modal-tags-input";
    qeTagsInput.type = "text";
    qeTagsInput.placeholder = qeTags.length ? "" : "Add tags...";

    var qeDropdown = document.createElement("div");
    qeDropdown.className = "card-modal-tags-dropdown";
    var qeDropIdx = -1;

    function qeGetTagsValue() { return qeTags.join(", "); }

    function qeRenderChips() {
      Array.from(qeTagsContainer.querySelectorAll(".card-modal-tag-chip")).forEach(function (el) { el.remove(); });
      qeTags.forEach(function (tag, idx) {
        var chip = document.createElement("span");
        chip.className = "card-modal-tag-chip";
        chip.textContent = tag;
        var rm = document.createElement("button");
        rm.className = "card-modal-tag-chip-remove";
        rm.type = "button";
        rm.innerHTML = "&times;";
        rm.addEventListener("click", function (e) {
          e.stopPropagation();
          qeTags.splice(idx, 1);
          qeRenderChips();
          qeTagsInput.placeholder = qeTags.length ? "" : "Add tags...";
        });
        chip.appendChild(rm);
        qeTagsContainer.insertBefore(chip, qeTagsInput);
      });
    }

    function qeAddTag(tag) {
      tag = tag.trim();
      if (!tag || qeTags.indexOf(tag) !== -1) return;
      qeTags.push(tag);
      qeRenderChips();
      qeTagsInput.value = "";
      qeTagsInput.placeholder = "";
      qeHideDropdown();
    }

    function qeShowDropdown(filter) {
      qeDropdown.innerHTML = "";
      qeDropIdx = -1;
      var f = (filter || "").toLowerCase();
      var suggestions = qeAllTags.filter(function (t) {
        return qeTags.indexOf(t) === -1 && (!f || t.toLowerCase().indexOf(f) !== -1);
      });
      if (suggestions.length === 0) {
        if (f) {
          var hint = document.createElement("div");
          hint.className = "card-modal-tags-dropdown-empty";
          hint.textContent = 'Press Enter to add "' + filter + '"';
          qeDropdown.appendChild(hint);
        }
        qeDropdown.classList.toggle("open", !!f);
        return;
      }
      suggestions.forEach(function (t) {
        var item = document.createElement("div");
        item.className = "card-modal-tags-dropdown-item";
        item.textContent = t;
        item.addEventListener("mousedown", function (e) {
          e.preventDefault();
          qeAddTag(t);
          qeTagsInput.focus();
        });
        qeDropdown.appendChild(item);
      });
      qeDropdown.classList.add("open");
    }

    function qeHideDropdown() {
      qeDropdown.classList.remove("open");
      qeDropIdx = -1;
    }

    qeTagsInput.addEventListener("input", function () { qeShowDropdown(qeTagsInput.value); });
    qeTagsInput.addEventListener("focus", function () { qeShowDropdown(qeTagsInput.value); });
    qeTagsInput.addEventListener("click", function () { qeShowDropdown(qeTagsInput.value); });
    qeTagsInput.addEventListener("blur", function () { setTimeout(qeHideDropdown, 150); });

    qeTagsInput.addEventListener("keydown", function (e) {
      var items = qeDropdown.querySelectorAll(".card-modal-tags-dropdown-item");
      if (e.key === "ArrowDown") {
        e.preventDefault();
        qeDropIdx = Math.min(qeDropIdx + 1, items.length - 1);
        items.forEach(function (it, i) { it.classList.toggle("active", i === qeDropIdx); });
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        qeDropIdx = Math.max(qeDropIdx - 1, 0);
        items.forEach(function (it, i) { it.classList.toggle("active", i === qeDropIdx); });
      } else if (e.key === "Enter") {
        e.preventDefault();
        if (qeDropIdx >= 0 && items[qeDropIdx]) {
          qeAddTag(items[qeDropIdx].textContent);
        } else if (qeTagsInput.value.trim()) {
          qeAddTag(qeTagsInput.value);
        }
        qeTagsInput.focus();
      } else if (e.key === "Backspace" && !qeTagsInput.value && qeTags.length) {
        qeTags.pop();
        qeRenderChips();
        qeTagsInput.placeholder = qeTags.length ? "" : "Add tags...";
      } else if (e.key === "Escape") {
        qeHideDropdown();
      }
    });

    qeTagsContainer.addEventListener("click", function () { qeTagsInput.focus(); });
    qeTagsContainer.appendChild(qeTagsInput);
    qeTagsContainer.appendChild(qeDropdown);
    qeRenderChips();
    overlay.appendChild(qeTagsContainer);

    var bodyInput = document.createElement("textarea");
    bodyInput.className = "qe-body";
    bodyInput.placeholder = "Description (optional)";
    bodyInput.value = currentBody;
    bodyInput.rows = 2;
    bodyInput.addEventListener("keydown", function (e) {
      var trigger = window.__lbNewLineTrigger ? window.__lbNewLineTrigger() : "shift-enter";
      if (trigger === "shift-enter") {
        if (e.key === "Enter" && !e.shiftKey) {
          e.preventDefault();
          var sb = overlay.querySelector(".btn-primary");
          if (sb) sb.click();
        }
      } else {
        if (e.key === "Enter" && e.shiftKey) {
          e.preventDefault();
          var sb = overlay.querySelector(".btn-primary");
          if (sb) sb.click();
        }
      }
      if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        var sb = overlay.querySelector(".btn-primary");
        if (sb) sb.click();
        return;
      }
      if (e.key === "Escape") hideQuickEdit();
    });
    overlay.appendChild(bodyInput);

    // Priority value — updated by context menu
    var qePriorityValue = { current: currentPriority };

    var actions = document.createElement("div");
    actions.className = "qe-actions";

    var saveBtn = document.createElement("button");
    saveBtn.className = "btn-primary btn-small";
    saveBtn.textContent = "Save";
    saveBtn.addEventListener("click", function () {
      htmx.ajax('POST', '/board/' + encodeURIComponent(slug) + '/cards/edit', {
        values: {
          col_idx: colIdx,
          card_idx: cardIdx,
          title: titleInput.value.trim(),
          body: bodyInput.value.trim(),
          tags: qeGetTagsValue(),
          priority: qePriorityValue.current,
          name: slug,
          version: LB.getBoardVersion(),
        },
        target: '#board-content',
        swap: 'innerHTML'
      });
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
    buildContextMenuForCard(card, cardRect, qePriorityValue);
  }

  function buildContextMenuForCard(card, cardRect, priorityRef) {
    if (!ctxMenu) ctxMenu = buildContextMenu();
    ctxTargetCard = card;
    ctxMenu.innerHTML = "";

    var isCompleted = card.classList.contains("completed");

    var completeBtn = card.querySelector('[hx-post$="/cards/complete"]');
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

    // Priority selector
    if (priorityRef) {
      var psep = document.createElement("div");
      psep.className = "ctx-separator";
      ctxMenu.appendChild(psep);

      var plabel = document.createElement("div");
      plabel.className = "ctx-submenu-label";
      plabel.textContent = "Priority";
      ctxMenu.appendChild(plabel);

      var pgroup = document.createElement("div");
      pgroup.className = "card-modal-priority-group ctx-priority-group";
      var pbtns = [];
      [
        { val: "", label: "—", title: "None" },
        { val: "low", label: "L", title: "Low" },
        { val: "medium", label: "M", title: "Medium" },
        { val: "high", label: "H", title: "High" },
        { val: "critical", label: "!", title: "Critical" },
      ].forEach(function (item) {
        var btn = document.createElement("button");
        btn.className =
          "card-modal-priority-btn" +
          (item.val === priorityRef.current ? " card-modal-priority-btn--active" : "") +
          (item.val ? " card-modal-priority-btn--" + item.val : "");
        btn.textContent = item.label;
        btn.title = item.title;
        btn.addEventListener("click", function () {
          priorityRef.current = item.val;
          pbtns.forEach(function (b) {
            b.className = b.className.replace(" card-modal-priority-btn--active", "");
          });
          btn.className += " card-modal-priority-btn--active";
        });
        pgroup.appendChild(btn);
        pbtns.push(btn);
      });
      ctxMenu.appendChild(pgroup);
    }

    var deleteBtn = card.querySelector('[hx-post$="/cards/delete"]');
    if (deleteBtn) {
      var sep2 = document.createElement("div");
      sep2.className = "ctx-separator";
      ctxMenu.appendChild(sep2);
      ctxMenu.appendChild(makeDeleteItem(deleteBtn, function () { hideQuickEdit(); }));
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

    var completeBtn = card.querySelector('[hx-post$="/cards/complete"]');
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

    var deleteBtn = card.querySelector('[hx-post$="/cards/delete"]');
    if (deleteBtn) {
      var sep2 = document.createElement("div");
      sep2.className = "ctx-separator";
      ctxMenu.appendChild(sep2);
      ctxMenu.appendChild(makeDeleteItem(deleteBtn, null));
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

  document.addEventListener("keydown", function (e) {
    if (e.key === "Escape") hideQuickEdit();
  });

  // Expose for cross-module use
  LB.hideContextMenu = hideContextMenu;
  LB.hideQuickEdit = hideQuickEdit;
  LB.makeItem = makeItem;
  LB.attachContextMenu = attachContextMenu;
  LB.showQuickEdit = showQuickEdit;

  // Expose internals needed by global dismiss handler
  LB._ctxMenuContains = function (el) { return ctxMenu && ctxMenu.contains(el); };
  LB._qeOverlayContains = function (el) { return qeOverlay && qeOverlay.contains(el); };
  LB._hasQeOverlay = function () { return !!qeOverlay; };
  LB._hasCtxMenu = function () { return ctxMenu && ctxMenu.classList.contains("visible"); };
})();
