// LiveBoard core: namespace, version control, conflict handling.
window.LB = window.LB || {};

(function () {
  var LB = window.LB;
  var _conflictRetrying = false;
  var _savedScrollLeft = 0;
  var _lastMutationSwapAt = 0;

  LB.isDragging = false;

  function getBoardVersion() {
    var el = document.getElementById("board-version");
    return el ? el.value : "0";
  }

  function extractVersionFromResponse(html) {
    var doc = new DOMParser().parseFromString(html, "text/html");
    var el = doc.getElementById("board-version");
    return el ? el.value || el.getAttribute("value") : null;
  }

  // Save horizontal scroll position before HTMX replaces board content.
  document.body.addEventListener("htmx:beforeRequest", function (e) {
    var c = document.querySelector(".columns-container");
    if (c) _savedScrollLeft = c.scrollLeft;

    // Suppress SSE echo: our POST already swapped fresh HTML.
    if (_lastMutationSwapAt && e.detail.requestConfig) {
      var cfg = e.detail.requestConfig;
      if (cfg.verb === "get" && Date.now() - _lastMutationSwapAt < 2000) {
        var isSSE = cfg.triggeringEvent && cfg.triggeringEvent.type === "sse:board-update";
        if (isSSE) {
          e.preventDefault();
          return;
        }
      }
    }
  });

  // Auto-inject version into all HTMX form submissions (hx-post, hx-vals).
  document.body.addEventListener("htmx:configRequest", function (e) {
    var el = document.getElementById("board-version");
    if (el) {
      e.detail.parameters["version"] = el.value;
    }
  });

  // Handle 409 Conflict: retry once with fresh version, then fall back to swap + toast.
  document.body.addEventListener("htmx:beforeSwap", function (e) {
    if (!e.detail.xhr || e.detail.xhr.status !== 409) return;

    // Retry also got 409 — give up, swap fresh HTML and notify user.
    if (_conflictRetrying) {
      _conflictRetrying = false;
      e.detail.shouldSwap = true;
      e.detail.isError = false;
      showConflictToast();
      return;
    }

    var newVersion = extractVersionFromResponse(e.detail.xhr.responseText);
    if (!newVersion) {
      e.detail.shouldSwap = true;
      e.detail.isError = false;
      showConflictToast();
      return;
    }

    // Suppress swap — we will retry the request.
    e.detail.shouldSwap = false;
    e.detail.isError = false;

    // Update stored version so htmx:configRequest picks it up.
    var versionEl = document.getElementById("board-version");
    if (versionEl) versionEl.value = newVersion;

    var cfg = e.detail.requestConfig;
    var params = Object.assign({}, cfg.parameters);
    params.version = newVersion;

    _conflictRetrying = true;
    htmx.ajax(cfg.verb, cfg.path, {
      values: params,
      target: "#board-content",
      swap: "innerHTML",
    });
  });

  // Clear focus mode on boosted navigation (board switch).
  document.body.addEventListener("htmx:pushedIntoHistory", function () {
    if (typeof Alpine !== 'undefined' && Alpine.store('ui')) {
      Alpine.store('ui').focusedColumn = '';
    }
  });

  // After any swap, sync the board version from the hidden input and refresh store.
  document.body.addEventListener("htmx:afterSwap", function (e) {
    _conflictRetrying = false;

    // Track POST swaps so we can suppress the redundant SSE echo.
    if (e.detail.requestConfig && e.detail.requestConfig.verb === "post") {
      var tgt = e.detail.target || e.detail.elt;
      if (tgt && tgt.id === "board-content") {
        _lastMutationSwapAt = Date.now();
      }
    }

    var versionEl = document.getElementById("board-version");
    var boardView = document.querySelector(".board-view");
    if (versionEl && boardView) {
      boardView.dataset.boardVersion = versionEl.value;
    }
    // Restore horizontal scroll position after board content is replaced.
    var cols = document.querySelector(".columns-container");
    if (cols && _savedScrollLeft) {
      requestAnimationFrame(function () {
        cols.scrollLeft = _savedScrollLeft;
      });
    }
    // Refresh board store with fresh DOM data
    if (typeof Alpine !== 'undefined' && Alpine.store('board')) {
      Alpine.store('board').refresh();
    }
    // Re-apply client-side filters after content swap
    LB.applyFilters();
  });

  function showConflictToast() {
    var existing = document.getElementById("conflict-toast");
    if (existing) existing.remove();
    var toast = document.createElement("div");
    toast.id = "conflict-toast";
    toast.className = "conflict-toast";
    toast.textContent = "Board was updated. Refreshed to latest.";
    document.body.appendChild(toast);
    setTimeout(function () {
      toast.classList.add("conflict-toast-hide");
      setTimeout(function () { toast.remove(); }, 300);
    }, 2000);
  }

  // New Line Trigger helper
  function getNewLineTrigger() {
    var boardView = document.querySelector(".board-view");
    if (boardView) return boardView.dataset.newlineTrigger || "shift-enter";
    return localStorage.getItem("lb_newline_trigger") || "shift-enter";
  }

  window.__lbNewLineTrigger = getNewLineTrigger;

  // Apply client-side filters (search + hide-completed) and update visible counts.
  LB.applyFilters = function () {
    var ui = (typeof Alpine !== 'undefined' && Alpine.store('ui')) ? Alpine.store('ui') : null;
    var query = ui ? (ui.searchQuery || '').toLowerCase().trim() : '';

    // Re-sync hide-completed attr on stable parent
    var bv = document.querySelector('.board-view');
    if (bv && ui) bv.setAttribute('data-hide-completed', ui.hideCompleted);

    // Search filter: toggle .search-hidden on cards
    var cards = document.querySelectorAll('.card[data-card-title]');
    cards.forEach(function (c) {
      if (!query) {
        c.classList.remove('search-hidden');
        return;
      }
      var haystack = [
        c.dataset.cardTitle || '',
        c.dataset.cardBody || '',
        c.dataset.cardTags || '',
        c.dataset.cardAssignee || '',
        c.dataset.cardColumn || ''
      ].join(' ').toLowerCase();
      c.classList.toggle('search-hidden', haystack.indexOf(query) === -1);
    });

    // Update column card counts (board view)
    document.querySelectorAll('.column').forEach(function (col) {
      var countEl = col.querySelector('.column-card-count');
      if (!countEl) return;
      var visible = col.querySelectorAll('.card[data-card-title]:not(.search-hidden):not([style*="display: none"])');
      // Also exclude hide-completed cards via data attr
      var hideCompleted = bv && bv.getAttribute('data-hide-completed') === 'true';
      var count = 0;
      visible.forEach(function (c) {
        if (hideCompleted && c.classList.contains('completed')) return;
        count++;
      });
      countEl.textContent = count;
    });

    // Update section counts (list view)
    document.querySelectorAll('.list-section').forEach(function (sec) {
      var countEl = sec.querySelector('.section-count');
      if (!countEl) return;
      var hideCompleted = bv && bv.getAttribute('data-hide-completed') === 'true';
      var count = 0;
      sec.querySelectorAll('.list-item.card').forEach(function (c) {
        if (c.classList.contains('search-hidden')) return;
        if (hideCompleted && c.classList.contains('completed')) return;
        count++;
      });
      countEl.textContent = count;
    });
  };

  LB.getBoardVersion = getBoardVersion;

  LB.colorLuminance = function (hex) {
    var r = parseInt(hex.slice(1, 3), 16) / 255;
    var g = parseInt(hex.slice(3, 5), 16) / 255;
    var b = parseInt(hex.slice(5, 7), 16) / 255;
    var toLinear = function (c) { return c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4); };
    return 0.2126 * toLinear(r) + 0.7152 * toLinear(g) + 0.0722 * toLinear(b);
  };

  LB.applyTagColors = function () {
    var bv = document.querySelector('.board-view');
    if (!bv) return;
    var map = {};
    try { map = JSON.parse(bv.dataset.boardTagColors || '{}'); } catch (e) {}
    document.querySelectorAll('.tag[data-tag]').forEach(function (el) {
      var color = map[el.dataset.tag];
      if (color) {
        el.style.background = color;
        el.style.color = LB.colorLuminance(color) > 0.35 ? '#111' : '#fff';
      } else {
        el.style.background = '';
        el.style.color = '';
      }
    });
  };

  document.addEventListener('DOMContentLoaded', LB.applyTagColors);
  document.addEventListener('htmx:afterSwap', LB.applyTagColors);
})();
