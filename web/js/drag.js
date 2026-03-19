// Drag-and-drop for board cards: between columns and within-column reordering.
// Works with jfyne/live by clicking hidden trigger buttons wired by live.js.
(function () {
  var draggingCardId = null;
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
    var cardId = card.dataset.cardId;
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
          card_id: cardId,
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

  document.addEventListener("click", function (e) {
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
    document.querySelectorAll(".card[data-card-id]").forEach(function (card) {
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
  function getInsertionTarget(zone, clientY, excludeCardId) {
    var cards = Array.from(zone.querySelectorAll(".card[data-card-id]")).filter(
      function (c) { return c.dataset.cardId !== excludeCardId; }
    );
    for (var i = 0; i < cards.length; i++) {
      var rect = cards[i].getBoundingClientRect();
      if (clientY < rect.top + rect.height / 2) {
        return cards[i];
      }
    }
    return null; // append to end
  }

  function attach() {
    attachContextMenu();
    // Cards: draggable
    document.querySelectorAll(".card[draggable]").forEach(function (card) {
      if (card.dataset.dragWired) return;
      card.dataset.dragWired = "1";

      card.addEventListener("dragstart", function (e) {
        draggingCardId = card.dataset.cardId;
        var zone = card.closest(".cards[data-column]");
        draggingSourceColumn = zone ? zone.dataset.column : null;
        card.classList.add("dragging");
        e.dataTransfer.effectAllowed = "move";
        e.dataTransfer.setData("text/plain", card.dataset.cardId);
      });

      card.addEventListener("dragend", function () {
        card.classList.remove("dragging");
        clearDropIndicators();
        document.querySelectorAll(".cards.drag-over").forEach(function (el) {
          el.classList.remove("drag-over");
        });
        draggingCardId = null;
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
          var beforeCard = getInsertionTarget(zone, e.clientY, draggingCardId);
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

        var cardId = e.dataTransfer.getData("text/plain");
        var targetColumn = zone.dataset.column;
        if (!cardId || !targetColumn) {
          clearDropIndicators();
          return;
        }

        var card = document.querySelector('.card[data-card-id="' + cardId + '"]');
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

          var beforeCardId = beforeCard ? beforeCard.dataset.cardId : "";

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
            (beforeCard && nextSibling && beforeCard.dataset.cardId === nextSibling.dataset.cardId)
          ) {
            return; // no-op
          }

          var slug = window.location.pathname.replace(/^\/board\//, "");
          if (window.Live) {
            window.Live.send("reorder-card", {
              card_id: cardId,
              before_card_id: beforeCardId,
              column: targetColumn,
              name: slug,
            });
          }
        } else {
          // Cross-column move
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
