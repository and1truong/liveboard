// LiveBoard: card + column drag-and-drop, attach() orchestrator.
(function () {
  var LB = window.LB;
  var draggingCard = null;
  var draggingSourceColumn = null;
  var draggingColumnEl = null;

  var isReadOnly = document.documentElement.hasAttribute('data-readonly');

  function attach() {
    // Card click → open Alpine card modal
    document.querySelectorAll(".card[data-card-idx]").forEach(function (card) {
      if (card.dataset.modalWired) return;
      card.dataset.modalWired = "1";

      card.addEventListener("mousedown", function () {
        LB.isDragging = false;
      });

      card.addEventListener("click", function (e) {
        if (LB.isDragging) return;
        if (e.target.closest("button, a, input, textarea, select")) return;
        // Open Alpine card modal
        var modalComp = document.querySelector('[x-data^="cardModal"]');
        if (modalComp && modalComp._x_dataStack) {
          Alpine.$data(modalComp).show(card);
        }
      });
    });

    if (!isReadOnly) {
    // Card right-click → open Alpine quick edit
    document.querySelectorAll(".card[data-card-idx]").forEach(function (card) {
      if (card.dataset.ctxWired) return;
      card.dataset.ctxWired = "1";
      card.addEventListener("contextmenu", function (e) {
        e.preventDefault();
        var qeComp = document.querySelector('[x-data^="quickEdit"]');
        if (qeComp && qeComp._x_dataStack) {
          Alpine.$data(qeComp).show(card);
        }
      });
    });

    // Column menu buttons → open Alpine column menu
    document.querySelectorAll(".column-menu-btn").forEach(function (btn) {
      if (btn.dataset.colMenuWired) return;
      btn.dataset.colMenuWired = "1";
      btn.addEventListener("click", function (e) {
        e.stopPropagation();
        var menuComp = document.querySelector('[x-data^="columnMenu"]');
        if (menuComp && menuComp._x_dataStack) {
          Alpine.$data(menuComp).show(btn);
        }
      });
    });

    // Column header double-click → rename
    document.querySelectorAll(".column-header h3").forEach(function (h3) {
      if (h3.dataset.dblWired) return;
      h3.dataset.dblWired = "1";
      h3.addEventListener("dblclick", function (e) {
        e.stopPropagation();
        var btn = h3.closest(".column-header").querySelector(".column-menu-btn");
        if (!btn) return;
        var menuComp = document.querySelector('[x-data^="columnMenu"]');
        if (menuComp && menuComp._x_dataStack) {
          var data = Alpine.$data(menuComp);
          data.columnName = btn.dataset.columnName;
          data.slug = decodeURIComponent(window.location.pathname.replace(/^\/board\//, ""));
          data.editColumn();
        }
      });
    });

    // Board title double-click → open settings
    var titleEl = document.querySelector(".board-title");
    if (titleEl && !titleEl.dataset.dblWired) {
      titleEl.dataset.dblWired = "1";
      titleEl.addEventListener("dblclick", function (e) {
        e.stopPropagation();
        var bsComp = document.querySelector('[x-data^="boardSettings"]');
        if (bsComp && bsComp._x_dataStack) {
          Alpine.$data(bsComp).toggle();
        }
      });
    }

    // Board settings gear button
    document.querySelectorAll(".board-settings-btn").forEach(function (btn) {
      if (btn.dataset.bsWired) return;
      btn.dataset.bsWired = "1";
      btn.addEventListener("click", function () {
        var bsComp = document.querySelector('[x-data^="boardSettings"]');
        if (bsComp && bsComp._x_dataStack) {
          Alpine.$data(bsComp).toggle();
        }
      });
    });
    } // end !isReadOnly

    // Collapsed column click → expand
    document.addEventListener("click", function (e) {
      var header = e.target.closest(".column-header");
      if (!header) return;
      var col = header.closest(".column");
      if (!col || !col.classList.contains("collapsed")) return;
      var btn = header.querySelector(".column-collapse-btn");
      if (btn && e.target !== btn) btn.click();
    });

    // Cards: draggable
    document.querySelectorAll(".card[draggable]").forEach(function (card) {
      if (card.dataset.dragWired) return;
      card.dataset.dragWired = "1";

      card.addEventListener("dragstart", function (e) {
        LB.isDragging = true;
        if (typeof Alpine !== 'undefined') Alpine.store('ui').isDragging = true;
        draggingCard = card;
        var zone = card.closest(".cards[data-column], .table-group-cards[data-column]");
        draggingSourceColumn = zone ? zone.dataset.column : null;
        card.classList.add("dragging");
        e.dataTransfer.effectAllowed = "move";
        e.dataTransfer.setData("text/plain", card.dataset.colIdx + ":" + card.dataset.cardIdx);
        e.stopPropagation();
      });

      card.addEventListener("dragend", function () {
        LB.isDragging = false;
        if (typeof Alpine !== 'undefined') Alpine.store('ui').isDragging = false;
        card.classList.remove("dragging");
        LB.clearDropIndicators();
        document.querySelectorAll(".cards.drag-over, .table-group-cards.drag-over").forEach(function (el) {
          el.classList.remove("drag-over");
        });
        draggingCard = null;
        draggingSourceColumn = null;
      });
    });

    // Drop zones: .cards containers (board view) and .table-group-cards (table view)
    document.querySelectorAll(".cards[data-column], .table-group-cards[data-column]").forEach(function (zone) {
      if (zone.dataset.dropWired) return;
      zone.dataset.dropWired = "1";

      zone.addEventListener("dragover", function (e) {
        if (draggingColumnEl) return;
        e.preventDefault();
        e.dataTransfer.dropEffect = "move";
        // Cancel any pending dragleave cleanup — we're still over the zone
        if (zone._dragLeaveTimer) {
          clearTimeout(zone._dragLeaveTimer);
          zone._dragLeaveTimer = null;
        }
        zone.classList.remove("drag-over");
        var beforeCard = LB.getInsertionTarget(zone, e.clientY, draggingCard);
        LB.showDropIndicator(zone, beforeCard);
      });

      zone.addEventListener("dragleave", function (e) {
        if (!zone.contains(e.relatedTarget)) {
          // Defer cleanup so a rapid dragover (from DOM shifts caused by
          // indicator insertion) can cancel the stale leave. Use 100ms to
          // outlast the browser's ~50ms dragover throttle interval.
          zone._dragLeaveTimer = setTimeout(function () {
            zone.classList.remove("drag-over");
            LB.clearDropIndicators();
          }, 100);
        }
      });

      zone.addEventListener("drop", function (e) {
        if (draggingColumnEl) return;
        e.preventDefault();
        zone.classList.remove("drag-over");

        var data = e.dataTransfer.getData("text/plain");
        var targetColumn = zone.dataset.column;
        if (!data || !targetColumn) {
          LB.clearDropIndicators();
          return;
        }

        var parts = data.split(":");
        var srcColIdx = parts[0];
        var srcCardIdx = parts[1];

        var card = document.querySelector('.card[data-col-idx="' + srcColIdx + '"][data-card-idx="' + srcCardIdx + '"]');
        if (!card) {
          LB.clearDropIndicators();
          return;
        }

        var sourceZone = card.closest(".cards[data-column], .table-group-cards[data-column]");
        var sourceColumn = sourceZone ? sourceZone.dataset.column : null;

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
        LB.clearDropIndicators();

        var beforeIdx = beforeCard ? beforeCard.dataset.cardIdx : "-1";

        if (sourceColumn === targetColumn) {
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
            return;
          }
        }

        var slug = decodeURIComponent(window.location.pathname.replace(/^\/board\//, ""));
        htmx.ajax('POST', '/board/' + encodeURIComponent(slug) + '/cards/reorder', {
          values: {
            col_idx: srcColIdx,
            card_idx: srcCardIdx,
            before_idx: beforeIdx,
            column: targetColumn,
            name: slug,
            version: LB.getBoardVersion(),
          },
          target: '#board-content',
          swap: 'innerHTML'
        });
      });
    });

    // ── Column drag-and-drop ──────────────────────────────────────────
    function clearColumnDropIndicators() {
      document.querySelectorAll(".column-drop-indicator").forEach(function (el) {
        el.remove();
      });
    }

    function getColumnInsertionTarget(container, clientX, excludeEl, isTable) {
      var selector = isTable ? ".table-group-cards[data-column]" : ".column[data-column-name]";
      var columns = Array.from(container.querySelectorAll(selector)).filter(
        function (c) { return c !== excludeEl; }
      );
      for (var i = 0; i < columns.length; i++) {
        var rect = columns[i].getBoundingClientRect();
        var mid = isTable ? rect.top + rect.height / 2 : rect.left + rect.width / 2;
        var pos = isTable ? clientX : clientX;
        if (pos < mid) {
          return columns[i];
        }
      }
      return null;
    }

    function showColumnDropIndicator(container, beforeCol, isTable) {
      clearColumnDropIndicators();
      var indicator = document.createElement("div");
      indicator.className = "column-drop-indicator" + (isTable ? " column-drop-indicator-horizontal" : "");
      if (beforeCol) {
        container.insertBefore(indicator, beforeCol);
      } else {
        var addCol = container.querySelector(".add-column-bar, .table-add-column");
        if (addCol) {
          container.insertBefore(indicator, addCol);
        } else {
          container.appendChild(indicator);
        }
      }
    }

    // Board mode: columns
    document.querySelectorAll(".column[draggable][data-column-name]").forEach(function (col) {
      if (col.dataset.colDragWired) return;
      col.dataset.colDragWired = "1";

      col.addEventListener("mousedown", function (e) {
        var header = col.querySelector(".column-header");
        col._dragFromHeader = header && header.contains(e.target);
      });

      col.addEventListener("dragstart", function (e) {
        if (!col._dragFromHeader) {
          e.preventDefault();
          return;
        }
        if (draggingCard) {
          e.preventDefault();
          return;
        }
        draggingColumnEl = col;
        col.classList.add("column-dragging");
        e.dataTransfer.effectAllowed = "move";
        e.dataTransfer.setData("application/x-column", col.dataset.columnName);
      });

      col.addEventListener("dragend", function () {
        col.classList.remove("column-dragging");
        clearColumnDropIndicators();
        draggingColumnEl = null;
      });
    });

    // Board mode: drop zone is .columns-container
    document.querySelectorAll(".columns-container").forEach(function (container) {
      if (container.dataset.colDropWired) return;
      container.dataset.colDropWired = "1";

      container.addEventListener("dragover", function (e) {
        if (!draggingColumnEl) return;
        e.preventDefault();
        e.dataTransfer.dropEffect = "move";
        var beforeCol = getColumnInsertionTarget(container, e.clientX, draggingColumnEl, false);
        showColumnDropIndicator(container, beforeCol, false);
      });

      container.addEventListener("dragleave", function (e) {
        if (!draggingColumnEl) return;
        if (!container.contains(e.relatedTarget)) {
          clearColumnDropIndicators();
        }
      });

      container.addEventListener("drop", function (e) {
        if (!draggingColumnEl) return;
        e.preventDefault();
        var colName = draggingColumnEl.dataset.columnName;

        var indicator = container.querySelector(".column-drop-indicator");
        var afterCol = "";
        if (indicator) {
          var prev = indicator.previousElementSibling;
          while (prev && !prev.classList.contains("column")) {
            prev = prev.previousElementSibling;
          }
          if (prev && prev.dataset.columnName) {
            afterCol = prev.dataset.columnName;
          }
        }
        clearColumnDropIndicators();

        if (afterCol === colName) return;
        var prevSib = draggingColumnEl.previousElementSibling;
        while (prevSib && !prevSib.classList.contains("column")) {
          prevSib = prevSib.previousElementSibling;
        }
        if (afterCol === "" && !prevSib) return;
        if (prevSib && prevSib.dataset.columnName === afterCol) return;

        var slug = decodeURIComponent(window.location.pathname.replace(/^\/board\//, ""));
        htmx.ajax('POST', '/board/' + encodeURIComponent(slug) + '/columns/move', {
          values: {
            column: colName,
            after_column: afterCol,
            version: LB.getBoardVersion(),
          },
          target: '#board-content',
          swap: 'innerHTML'
        });
      });
    });

    // Table mode: column groups
    document.querySelectorAll(".table-group-cards[draggable][data-column]").forEach(function (group) {
      if (group.dataset.colDragWired) return;
      group.dataset.colDragWired = "1";

      group.addEventListener("dragstart", function (e) {
        if (draggingCard) {
          e.preventDefault();
          return;
        }
        draggingColumnEl = group;
        group.classList.add("column-dragging");
        e.dataTransfer.effectAllowed = "move";
        e.dataTransfer.setData("application/x-column", group.dataset.column);
      });

      group.addEventListener("dragend", function () {
        group.classList.remove("column-dragging");
        clearColumnDropIndicators();
        draggingColumnEl = null;
      });
    });

    // Table mode: drop zone is .table-container
    document.querySelectorAll(".table-container").forEach(function (container) {
      if (container.dataset.colDropWired) return;
      container.dataset.colDropWired = "1";

      container.addEventListener("dragover", function (e) {
        if (!draggingColumnEl) return;
        e.preventDefault();
        e.dataTransfer.dropEffect = "move";
        var groups = Array.from(container.querySelectorAll(".table-group-cards[data-column]")).filter(
          function (g) { return g !== draggingColumnEl; }
        );
        var beforeGroup = null;
        for (var i = 0; i < groups.length; i++) {
          var rect = groups[i].getBoundingClientRect();
          if (e.clientY < rect.top + rect.height / 2) {
            beforeGroup = groups[i];
            break;
          }
        }
        showColumnDropIndicator(container, beforeGroup, true);
      });

      container.addEventListener("dragleave", function (e) {
        if (!draggingColumnEl) return;
        if (!container.contains(e.relatedTarget)) {
          clearColumnDropIndicators();
        }
      });

      container.addEventListener("drop", function (e) {
        if (!draggingColumnEl) return;
        e.preventDefault();
        var colName = draggingColumnEl.dataset.column;

        var indicator = container.querySelector(".column-drop-indicator");
        var afterCol = "";
        if (indicator) {
          var prev = indicator.previousElementSibling;
          while (prev && !prev.classList.contains("table-group-cards")) {
            prev = prev.previousElementSibling;
          }
          if (prev && prev.dataset.column) {
            afterCol = prev.dataset.column;
          }
        }
        clearColumnDropIndicators();

        if (afterCol === colName) return;
        var prevSib = draggingColumnEl.previousElementSibling;
        while (prevSib && !prevSib.classList.contains("table-group-cards")) {
          prevSib = prevSib.previousElementSibling;
        }
        if (afterCol === "" && !prevSib) return;
        if (prevSib && prevSib.dataset.column === afterCol) return;

        var slug = decodeURIComponent(window.location.pathname.replace(/^\/board\//, ""));
        htmx.ajax('POST', '/board/' + encodeURIComponent(slug) + '/columns/move', {
          values: {
            column: colName,
            after_column: afterCol,
            version: LB.getBoardVersion(),
          },
          target: '#board-content',
          swap: 'innerHTML'
        });
      });
    });
  }

  // Attach on load
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", attach);
  } else {
    attach();
  }

  // Re-attach after HTMX swaps
  document.addEventListener("htmx:afterSettle", function () {
    requestAnimationFrame(attach);
  });
})();
