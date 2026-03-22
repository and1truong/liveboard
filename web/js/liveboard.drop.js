// LiveBoard: drop indicator helpers for card drag-and-drop.
(function () {
  var LB = window.LB;

  function clearDropIndicators() {
    document.querySelectorAll(".drop-indicator").forEach(function (el) {
      el.remove();
    });
  }

  function showDropIndicator(zone, beforeCard) {
    // Check if indicator is already in the correct position to avoid DOM thrashing.
    // Removing and re-inserting on every dragover causes layout shifts that trigger
    // dragleave/dragover loops (flickering placeholder).
    var existing = zone.querySelector(".drop-indicator");
    if (existing) {
      if (beforeCard) {
        if (existing.nextElementSibling === beforeCard) return;
      } else {
        var endTrigger = zone.querySelector(".reorder-end-trigger");
        if (endTrigger) {
          if (existing.nextElementSibling === endTrigger) return;
        } else {
          if (!existing.nextElementSibling) return;
        }
      }
    }
    clearDropIndicators();
    var indicator = document.createElement("div");
    indicator.className = "drop-indicator";
    if (beforeCard) {
      zone.insertBefore(indicator, beforeCard);
    } else {
      // Append before the hidden end-trigger button
      var endTrigger2 = zone.querySelector(".reorder-end-trigger");
      if (endTrigger2) {
        zone.insertBefore(indicator, endTrigger2);
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

  LB.clearDropIndicators = clearDropIndicators;
  LB.showDropIndicator = showDropIndicator;
  LB.getInsertionTarget = getInsertionTarget;
})();
