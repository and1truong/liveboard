// LiveBoard: Keyboard shortcuts (vim-style navigation).
// Opt-in via Settings → Keyboard shortcuts toggle.
(function () {
  var LB = window.LB || (window.LB = {});
  var focusCol = -1;
  var focusCard = -1;
  var pendingG = false;
  var pendingGTimer = null;
  var helpOverlay = null;

  // ── Helpers ──

  function isEnabled() {
    return localStorage.getItem('lb_keyboard_shortcuts') === '1';
  }

  function isOnBoard() {
    return !!document.querySelector('.board-view');
  }

  function isTableView() {
    return !!document.querySelector('.table-container');
  }

  function isModalOpen() {
    if (typeof Alpine !== 'undefined' && Alpine.store('ui')) {
      return Alpine.store('ui').isModalOpen();
    }
    return false;
  }

  function isInputFocused() {
    var el = document.activeElement;
    if (!el) return false;
    var tag = el.tagName;
    if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return true;
    if (el.isContentEditable) return true;
    return false;
  }

  // ── DOM queries ──

  function getColumns() {
    return Array.from(document.querySelectorAll('.column:not(.collapsed)'));
  }

  function getCardsInColumn(colEl) {
    return Array.from(colEl.querySelectorAll('.card[data-card-idx]'));
  }

  function getTableCards() {
    return Array.from(document.querySelectorAll('.table-row.card'));
  }

  function getAllBoardCards() {
    var cards = [];
    getColumns().forEach(function (col) {
      cards = cards.concat(getCardsInColumn(col));
    });
    return cards;
  }

  // ── Focus management ──

  function clearFocus() {
    var prev = document.querySelector('.card-focused');
    if (prev) prev.classList.remove('card-focused');
    focusCol = -1;
    focusCard = -1;
  }

  function applyFocus(el) {
    var prev = document.querySelector('.card-focused');
    if (prev) prev.classList.remove('card-focused');
    if (!el) return;
    el.classList.add('card-focused');
    el.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
  }

  function focusByIndices() {
    if (isTableView()) {
      var cards = getTableCards();
      if (focusCard < 0) focusCard = 0;
      if (focusCard >= cards.length) focusCard = cards.length - 1;
      if (cards[focusCard]) applyFocus(cards[focusCard]);
    } else {
      var cols = getColumns();
      if (cols.length === 0) return;
      if (focusCol < 0) focusCol = 0;
      if (focusCol >= cols.length) focusCol = cols.length - 1;
      var colCards = getCardsInColumn(cols[focusCol]);
      if (colCards.length === 0) return;
      if (focusCard < 0) focusCard = 0;
      if (focusCard >= colCards.length) focusCard = colCards.length - 1;
      applyFocus(colCards[focusCard]);
    }
  }

  // ── Actions ──

  function openFocusedCard() {
    var el = document.querySelector('.card-focused');
    if (!el) return;
    var modalComp = document.querySelector('[x-data^="cardModal"]');
    if (modalComp && modalComp._x_dataStack) {
      Alpine.$data(modalComp).show(el);
    }
  }

  function toggleFocusedComplete() {
    var el = document.querySelector('.card-focused');
    if (!el) return;
    var btn = el.querySelector('.card-checkbox');
    if (btn) btn.click();
  }

  function focusAddCard() {
    if (isTableView()) {
      var link = document.querySelector('.table-add-card-link');
      if (link) link.click();
    } else {
      var cols = getColumns();
      if (focusCol >= 0 && focusCol < cols.length) {
        var btn = cols[focusCol].querySelector('.btn-add-card');
        if (btn) {
          btn.click();
          var self = cols[focusCol];
          setTimeout(function () {
            var input = self.querySelector('.add-card-input');
            if (input) input.focus();
          }, 50);
        }
      }
    }
  }

  function toggleTheme() {
    var html = document.documentElement;
    var current = html.getAttribute('data-theme');
    var next;
    if (current === 'dark') {
      next = 'light';
    } else {
      next = 'dark';
    }
    html.setAttribute('data-theme', next);
    localStorage.setItem('lb_theme', next);
  }

  function openCmdPalette() {
    var cp = document.querySelector('[x-data^="cmdPalette"]');
    if (cp && cp._x_dataStack) {
      Alpine.$data(cp).toggle();
    }
  }

  // ── Help overlay ──

  var shortcuts = [
    ['Navigation', null],
    ['j / \u2193', 'Next card'],
    ['k / \u2191', 'Previous card'],
    ['h / \u2190', 'Previous column (board view)'],
    ['l / \u2192', 'Next column (board view)'],
    ['g g', 'First card'],
    ['G', 'Last card in column'],
    ['Actions', null],
    ['Enter / o', 'Open card'],
    ['x', 'Toggle card complete'],
    ['n', 'New card in column'],
    ['/ ', 'Open command palette'],
    ['Shift+D', 'Toggle dark/light theme'],
    ['Esc', 'Clear focus'],
    ['?', 'This help'],
  ];

  function createHelpOverlay() {
    if (helpOverlay) return helpOverlay;
    var overlay = document.createElement('div');
    overlay.className = 'keyboard-help-overlay';
    overlay.addEventListener('click', function (e) {
      if (e.target === overlay) closeHelp();
    });
    var panel = document.createElement('div');
    panel.className = 'keyboard-help-panel';
    var title = document.createElement('div');
    title.className = 'keyboard-help-title';
    title.textContent = 'Keyboard Shortcuts';
    panel.appendChild(title);
    var grid = document.createElement('div');
    grid.className = 'keyboard-help-grid';
    shortcuts.forEach(function (s) {
      if (s[1] === null) {
        var section = document.createElement('div');
        section.className = 'keyboard-help-section';
        section.textContent = s[0];
        grid.appendChild(section);
      } else {
        var key = document.createElement('span');
        key.className = 'keyboard-help-key';
        key.textContent = s[0];
        grid.appendChild(key);
        var desc = document.createElement('span');
        desc.className = 'keyboard-help-desc';
        desc.textContent = s[1];
        grid.appendChild(desc);
      }
    });
    panel.appendChild(grid);
    overlay.appendChild(panel);
    helpOverlay = overlay;
    return overlay;
  }

  function showHelp() {
    var el = createHelpOverlay();
    if (!el.parentNode) document.body.appendChild(el);
  }

  function closeHelp() {
    if (helpOverlay && helpOverlay.parentNode) {
      helpOverlay.parentNode.removeChild(helpOverlay);
    }
  }

  function isHelpOpen() {
    return helpOverlay && helpOverlay.parentNode;
  }

  // ── Keydown handler ──

  function handleKeydown(e) {
    if (!isEnabled()) return;
    if (!isOnBoard()) return;

    // Close help on Escape
    if (e.key === 'Escape' && isHelpOpen()) {
      e.preventDefault();
      closeHelp();
      return;
    }

    // Don't capture when typing in inputs or modals are open
    if (isInputFocused()) return;
    if (isModalOpen()) return;

    // Ignore when modifier keys are held (let Cmd+K etc. through)
    if (e.metaKey || e.ctrlKey || e.altKey) return;

    var key = e.key;

    // Shift+D → toggle theme
    if (key === 'D' && e.shiftKey) {
      e.preventDefault();
      toggleTheme();
      return;
    }

    // Shift+G → last card
    if (key === 'G' && e.shiftKey) {
      e.preventDefault();
      if (isTableView()) {
        var tc = getTableCards();
        if (tc.length > 0) { focusCard = tc.length - 1; focusByIndices(); }
      } else {
        var cols = getColumns();
        if (focusCol < 0 && cols.length > 0) focusCol = 0;
        if (focusCol >= 0 && focusCol < cols.length) {
          var cc = getCardsInColumn(cols[focusCol]);
          if (cc.length > 0) { focusCard = cc.length - 1; focusByIndices(); }
        }
      }
      return;
    }

    // Shift+/ (?) → show help
    if (key === '?' && e.shiftKey) {
      e.preventDefault();
      showHelp();
      return;
    }

    // Skip other shifted keys
    if (e.shiftKey) return;

    switch (key) {
      case 'ArrowDown':
      case 'j': // next card
        e.preventDefault();
        if (isTableView()) {
          focusCard++;
          focusByIndices();
        } else {
          var cols = getColumns();
          if (cols.length === 0) break;
          if (focusCol < 0) focusCol = 0;
          if (focusCol >= cols.length) focusCol = cols.length - 1;
          focusCard++;
          var cc = getCardsInColumn(cols[focusCol]);
          if (focusCard >= cc.length) focusCard = cc.length - 1;
          if (cc.length > 0) focusByIndices();
        }
        break;

      case 'ArrowUp':
      case 'k': // prev card
        e.preventDefault();
        if (isTableView()) {
          focusCard--;
          if (focusCard < 0) focusCard = 0;
          focusByIndices();
        } else {
          focusCard--;
          if (focusCard < 0) focusCard = 0;
          focusByIndices();
        }
        break;

      case 'ArrowLeft':
      case 'h': // prev column (board only)
        e.preventDefault();
        if (isTableView()) break;
        var cols = getColumns();
        if (cols.length === 0) break;
        focusCol--;
        if (focusCol < 0) focusCol = 0;
        // clamp card index to new column
        var cc = getCardsInColumn(cols[focusCol]);
        if (focusCard >= cc.length) focusCard = cc.length - 1;
        if (focusCard < 0) focusCard = 0;
        if (cc.length > 0) focusByIndices();
        break;

      case 'ArrowRight':
      case 'l': // next column (board only)
        e.preventDefault();
        if (isTableView()) break;
        var cols = getColumns();
        if (cols.length === 0) break;
        focusCol++;
        if (focusCol >= cols.length) focusCol = cols.length - 1;
        var cc = getCardsInColumn(cols[focusCol]);
        if (focusCard >= cc.length) focusCard = cc.length - 1;
        if (focusCard < 0) focusCard = 0;
        if (cc.length > 0) focusByIndices();
        break;

      case 'Enter':
      case 'o':
        e.preventDefault();
        openFocusedCard();
        break;

      case 'x':
        e.preventDefault();
        toggleFocusedComplete();
        break;

      case 'n':
        e.preventDefault();
        focusAddCard();
        break;

      case 'Escape':
        e.preventDefault();
        clearFocus();
        break;

      case '/':
        e.preventDefault();
        openCmdPalette();
        break;

      case 'g':
        e.preventDefault();
        if (pendingG) {
          clearTimeout(pendingGTimer);
          pendingG = false;
          // gg → go to first card
          focusCol = 0;
          focusCard = 0;
          focusByIndices();
        } else {
          pendingG = true;
          pendingGTimer = setTimeout(function () { pendingG = false; }, 500);
        }
        break;
    }
  }

  // ── Init ──

  document.addEventListener('DOMContentLoaded', function () {
    document.addEventListener('keydown', handleKeydown);

    // Re-apply focus after HTMX swaps
    document.body.addEventListener('htmx:afterSwap', function (e) {
      if (e.detail.target && e.detail.target.id === 'board-content') {
        if (focusCol >= 0 || focusCard >= 0) {
          setTimeout(function () { focusByIndices(); }, 50);
        }
      }
    });
  });

  // Cross-tab sync
  window.addEventListener('storage', function (e) {
    if (e.key === 'lb_keyboard_shortcuts' && e.newValue !== '1') {
      clearFocus();
      closeHelp();
    }
  });

  LB.keyboard = {
    clearFocus: clearFocus,
    isEnabled: isEnabled
  };
})();
