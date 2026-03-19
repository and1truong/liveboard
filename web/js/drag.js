// Drag-and-drop for board cards between columns.
// Works with jfyne/live by clicking hidden .move-trigger buttons
// that have live-click="move-card" wired by live.js.
(function () {
  function attach() {
    // Cards: draggable
    document.querySelectorAll(".card[draggable]").forEach(function (card) {
      if (card.dataset.dragWired) return;
      card.dataset.dragWired = "1";

      card.addEventListener("dragstart", function (e) {
        card.classList.add("dragging");
        e.dataTransfer.effectAllowed = "move";
        e.dataTransfer.setData("text/plain", card.dataset.cardId);
      });

      card.addEventListener("dragend", function () {
        card.classList.remove("dragging");
        document
          .querySelectorAll(".cards.drag-over")
          .forEach(function (el) { el.classList.remove("drag-over"); });
      });
    });

    // Drop zones: .cards containers
    document.querySelectorAll(".cards[data-column]").forEach(function (zone) {
      if (zone.dataset.dropWired) return;
      zone.dataset.dropWired = "1";

      zone.addEventListener("dragover", function (e) {
        e.preventDefault();
        e.dataTransfer.dropEffect = "move";
        zone.classList.add("drag-over");
      });

      zone.addEventListener("dragleave", function (e) {
        // Only remove if leaving the zone itself, not a child
        if (!zone.contains(e.relatedTarget)) {
          zone.classList.remove("drag-over");
        }
      });

      zone.addEventListener("drop", function (e) {
        e.preventDefault();
        zone.classList.remove("drag-over");

        var cardId = e.dataTransfer.getData("text/plain");
        var targetColumn = zone.dataset.column;
        if (!cardId || !targetColumn) return;

        // Find the card element and its hidden move-trigger button
        var card = document.querySelector(
          '.card[data-card-id="' + cardId + '"]'
        );
        if (!card) return;

        // Skip if dropped on same column
        var sourceZone = card.closest(".cards[data-column]");
        if (sourceZone && sourceZone.dataset.column === targetColumn) return;

        // Find the pre-wired trigger button for this target column
        var btn = card.querySelector(
          '.move-trigger[data-target="' + targetColumn + '"]'
        );
        if (!btn) return;

        btn.click();
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
