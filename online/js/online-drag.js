// LiveBoard Online: drag-and-drop for cards and columns.
// Wired to Alpine store instead of htmx POST endpoints.
(function () {
  var LB = window.LB;
  var draggingCard = null;
  var draggingColumnEl = null;

  function getSlug() {
    return (typeof Alpine !== 'undefined' && Alpine.store('lb')) ? Alpine.store('lb')._currentSlug || '' : '';
  }

  function clearDropIndicators() {
    document.querySelectorAll('.drop-indicator').forEach(function (el) { el.remove(); });
  }

  function showDropIndicator(zone, beforeCard) {
    var existing = zone.querySelector('.drop-indicator');
    if (existing) {
      if (beforeCard) { if (existing.nextElementSibling === beforeCard) return; }
      else { if (!existing.nextElementSibling) return; }
    }
    clearDropIndicators();
    var indicator = document.createElement('div');
    indicator.className = 'drop-indicator';
    if (beforeCard) zone.insertBefore(indicator, beforeCard);
    else zone.appendChild(indicator);
  }

  function getInsertionTarget(zone, clientY, excludeEl) {
    var cards = Array.from(zone.querySelectorAll('.card[data-card-idx]')).filter(function (c) { return c !== excludeEl; });
    for (var i = 0; i < cards.length; i++) {
      var rect = cards[i].getBoundingClientRect();
      if (clientY < rect.top + rect.height / 2) return cards[i];
    }
    return null;
  }

  LB.clearDropIndicators = clearDropIndicators;
  LB.showDropIndicator = showDropIndicator;
  LB.getInsertionTarget = getInsertionTarget;

  function attach() {
    // Card drag
    document.querySelectorAll('.card[draggable]:not(.calendar-card-chip)').forEach(function (card) {
      if (card.dataset.dragWired) return;
      card.dataset.dragWired = '1';

      card.addEventListener('dragstart', function (e) {
        LB.isDragging = true;
        if (typeof Alpine !== 'undefined') Alpine.store('ui').isDragging = true;
        draggingCard = card;
        card.classList.add('dragging');
        e.dataTransfer.effectAllowed = 'move';
        e.dataTransfer.setData('text/plain', card.dataset.colIdx + ':' + card.dataset.cardIdx);
        e.stopPropagation();
      });

      card.addEventListener('dragend', function () {
        LB.isDragging = false;
        if (typeof Alpine !== 'undefined') Alpine.store('ui').isDragging = false;
        card.classList.remove('dragging');
        clearDropIndicators();
        draggingCard = null;
      });
    });

    // Drop zones
    document.querySelectorAll('.cards[data-column]').forEach(function (zone) {
      if (zone.dataset.dropWired) return;
      zone.dataset.dropWired = '1';

      zone.addEventListener('dragover', function (e) {
        if (draggingColumnEl) return;
        e.preventDefault();
        e.dataTransfer.dropEffect = 'move';
        var beforeCard = getInsertionTarget(zone, e.clientY, draggingCard);
        showDropIndicator(zone, beforeCard);
      });

      zone.addEventListener('dragleave', function (e) {
        if (!zone.contains(e.relatedTarget)) {
          clearDropIndicators();
        }
      });

      zone.addEventListener('drop', function (e) {
        if (draggingColumnEl) return;
        e.preventDefault();
        var data = e.dataTransfer.getData('text/plain');
        var targetColumn = zone.dataset.column;
        if (!data || !targetColumn) { clearDropIndicators(); return; }

        var parts = data.split(':');
        var srcColIdx = parseInt(parts[0], 10);
        var srcCardIdx = parseInt(parts[1], 10);

        // Find insertion index
        var indicator = zone.querySelector('.drop-indicator');
        var beforeCard = null;
        if (indicator) {
          var next = indicator.nextElementSibling;
          while (next && !next.classList.contains('card')) next = next.nextElementSibling;
          if (next && next.classList.contains('card')) beforeCard = next;
        }
        clearDropIndicators();

        var beforeIdx = beforeCard ? parseInt(beforeCard.dataset.cardIdx, 10) : -1;
        var slug = getSlug();
        Alpine.store('lb').reorderCard(slug, srcColIdx, srcCardIdx, targetColumn, beforeIdx);
        Alpine.store('board').refresh();
      });
    });

    // Column drag
    function clearColumnDropIndicators() {
      document.querySelectorAll('.column-drop-indicator').forEach(function (el) { el.remove(); });
    }

    document.querySelectorAll('.column[draggable][data-column-name]').forEach(function (col) {
      if (col.dataset.colDragWired) return;
      col.dataset.colDragWired = '1';

      col.addEventListener('mousedown', function (e) {
        var header = col.querySelector('.column-header');
        col._dragFromHeader = header && header.contains(e.target);
      });

      col.addEventListener('dragstart', function (e) {
        if (!col._dragFromHeader || draggingCard) { e.preventDefault(); return; }
        draggingColumnEl = col;
        col.classList.add('column-dragging');
        e.dataTransfer.effectAllowed = 'move';
        e.dataTransfer.setData('application/x-column', col.dataset.columnName);
      });

      col.addEventListener('dragend', function () {
        col.classList.remove('column-dragging');
        clearColumnDropIndicators();
        draggingColumnEl = null;
      });
    });

    document.querySelectorAll('.columns-container').forEach(function (container) {
      if (container.dataset.colDropWired) return;
      container.dataset.colDropWired = '1';

      container.addEventListener('dragover', function (e) {
        if (!draggingColumnEl) return;
        e.preventDefault();
        e.dataTransfer.dropEffect = 'move';
        clearColumnDropIndicators();
        var columns = Array.from(container.querySelectorAll('.column[data-column-name]')).filter(function (c) { return c !== draggingColumnEl; });
        var beforeCol = null;
        for (var i = 0; i < columns.length; i++) {
          var rect = columns[i].getBoundingClientRect();
          if (e.clientX < rect.left + rect.width / 2) { beforeCol = columns[i]; break; }
        }
        var indicator = document.createElement('div');
        indicator.className = 'column-drop-indicator';
        if (beforeCol) container.insertBefore(indicator, beforeCol);
        else {
          var addCol = container.querySelector('.add-column-bar');
          if (addCol) container.insertBefore(indicator, addCol);
          else container.appendChild(indicator);
        }
      });

      container.addEventListener('dragleave', function (e) {
        if (!draggingColumnEl || container.contains(e.relatedTarget)) return;
        clearColumnDropIndicators();
      });

      container.addEventListener('drop', function (e) {
        if (!draggingColumnEl) return;
        e.preventDefault();
        var colName = draggingColumnEl.dataset.columnName;
        var indicator = container.querySelector('.column-drop-indicator');
        var afterCol = '';
        if (indicator) {
          var prev = indicator.previousElementSibling;
          while (prev && !prev.classList.contains('column')) prev = prev.previousElementSibling;
          if (prev && prev.dataset.columnName) afterCol = prev.dataset.columnName;
        }
        clearColumnDropIndicators();
        if (afterCol === colName) return;
        var slug = getSlug();
        Alpine.store('lb').moveColumn(slug, colName, afterCol);
        Alpine.store('board').refresh();
      });
    });
  }

  // MutationObserver to re-attach after Alpine re-renders (debounced)
  var _attachTimer = null;
  var observer = new MutationObserver(function () {
    if (_attachTimer) return;
    _attachTimer = requestAnimationFrame(function () {
      _attachTimer = null;
      attach();
    });
  });

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', function () {
      attach();
      observer.observe(document.body, { childList: true, subtree: true });
    });
  } else {
    attach();
    observer.observe(document.body, { childList: true, subtree: true });
  }
})();
