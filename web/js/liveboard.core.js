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

    var cfg = e.detail.requestConfig;
    if (!cfg) return;

    // Mark POST mutations so SSE echo can be suppressed.
    if (cfg.verb === "post") {
      var tgt = e.detail.target || e.detail.elt;
      if (tgt && tgt.id === "board-content") {
        _lastMutationSwapAt = Date.now();
      }
    }

    // Suppress SSE echo: our POST already handled the update.
    if (_lastMutationSwapAt && cfg.verb === "get" && Date.now() - _lastMutationSwapAt < 2000) {
      var isSSE = cfg.triggeringEvent && cfg.triggeringEvent.type === "sse:board-update";
      if (isSSE) {
        e.preventDefault();
        return;
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

  // Minimal toggle: update card UI client-side when server returns no body.
  document.body.addEventListener("boardToggled", function (e) {
    var d = e.detail;
    var card = document.querySelector(
      '.card[data-col-idx="' + d.colIdx + '"][data-card-idx="' + d.cardIdx + '"]'
    );
    if (!card) return;
    card.classList.toggle("completed", d.completed);
    card.dataset.cardCompleted = String(d.completed);
    var cb = card.querySelector(".card-checkbox");
    if (cb) cb.classList.toggle("checked", d.completed);
  });

  // Minimal card edit: update card UI client-side when server returns no body.
  document.body.addEventListener("cardEdited", function (e) {
    var d = e.detail;
    var card = document.querySelector(
      '.card[data-col-idx="' + d.colIdx + '"][data-card-idx="' + d.cardIdx + '"]'
    );
    if (!card) return;

    // Update data attributes so quick-edit reads fresh values.
    card.dataset.cardTitle = d.title || '';
    card.dataset.cardBody = d.body || '';
    card.dataset.cardTags = (d.tags || []).join(', ');
    card.dataset.cardPriority = d.priority || '';
    card.dataset.cardDue = d.due || '';
    card.dataset.cardAssignee = d.assignee || '';

    var isTable = card.classList.contains('table-row');

    // --- Title ---
    var titleEl = isTable
      ? card.querySelector('.table-card-title span')
      : card.querySelector('.card-title');
    if (titleEl) titleEl.textContent = d.title || '';

    // --- Tags ---
    var tagsHTML = '';
    (d.tags || []).forEach(function (t) {
      tagsHTML += '<span class="tag" data-tag="' + t.replace(/"/g, '&quot;') + '">' + t + '</span>';
    });
    if (isTable) {
      var tc = card.querySelector('.table-cell-tags');
      if (tc) tc.innerHTML = tagsHTML;
    } else {
      var tagsDiv = card.querySelector('.card-tags');
      if (d.tags && d.tags.length) {
        if (!tagsDiv) {
          tagsDiv = document.createElement('div');
          tagsDiv.className = 'card-tags';
          var content = card.querySelector('.card-content');
          if (content) {
            var titleH = content.querySelector('.card-title');
            var bodyDiv = content.querySelector('.card-body');
            var after = bodyDiv || titleH;
            if (after && after.nextSibling) content.insertBefore(tagsDiv, after.nextSibling);
            else content.appendChild(tagsDiv);
          }
        }
        tagsDiv.innerHTML = tagsHTML;
      } else if (tagsDiv) {
        tagsDiv.remove();
      }
    }

    // --- Priority + Assignee ---
    if (isTable) {
      var pc = card.querySelector('.table-cell-priority');
      if (pc) pc.innerHTML = d.priority ? '<span class="priority-indicator priority-' + d.priority + '">' + d.priority + '</span>' : '';
      var ac = card.querySelector('.table-cell-assignee');
      if (ac) ac.textContent = d.assignee || '';
    } else {
      var metaRow = card.querySelector('.card-meta-row');
      if (d.assignee || d.priority) {
        var metaHTML = '';
        if (d.assignee) metaHTML += '<span class="card-meta">&#128100; ' + d.assignee + '</span>';
        if (d.priority) metaHTML += '<span class="card-meta">&#9889; ' + d.priority + '</span>';
        if (!metaRow) {
          metaRow = document.createElement('div');
          metaRow.className = 'card-meta-row';
          var content = card.querySelector('.card-content');
          if (content) content.appendChild(metaRow);
        }
        metaRow.innerHTML = metaHTML;
      } else if (metaRow) {
        metaRow.remove();
      }
    }

    // --- Due date ---
    if (isTable) {
      var dc = card.querySelector('.table-cell-due');
      if (dc) dc.textContent = d.due || '';
    } else {
      // Due is a <p class="card-meta"> after .card-meta-row; find by content pattern.
      var content = card.querySelector('.card-content');
      if (content) {
        var dueParagraphs = content.querySelectorAll('p.card-meta');
        dueParagraphs.forEach(function (p) { p.remove(); });
        if (d.due) {
          var p = document.createElement('p');
          p.className = 'card-meta';
          p.innerHTML = '&#128197; ' + d.due;
          content.appendChild(p);
        }
      }
    }

    LB.applyTagColors();
  });

  // Pick up X-Board-Version header from minimal (no-body) responses.
  document.body.addEventListener("htmx:afterRequest", function (e) {
    var xhr = e.detail.xhr;
    if (!xhr) return;
    var v = xhr.getResponseHeader("X-Board-Version");
    if (!v) return;
    var el = document.getElementById("board-version");
    if (el) el.value = v;
    var bv = document.querySelector(".board-view");
    if (bv) bv.dataset.boardVersion = v;
    if (typeof Alpine !== 'undefined' && Alpine.store('board')) {
      Alpine.store('board').refresh();
    }
  });

  document.addEventListener('DOMContentLoaded', LB.applyTagColors);
  document.addEventListener('htmx:afterSwap', LB.applyTagColors);
})();
