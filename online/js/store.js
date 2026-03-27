// LiveBoard Online: Alpine.js data store backed by localStorage.
// Provides full CRUD for boards, columns, and cards.

(function () {
  var STORAGE_KEY = 'liveboard_data';
  var SETTINGS_KEY = 'liveboard_settings';

  function generateSlug(name) {
    var base = name.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '') || 'board';
    return base + '-' + Date.now().toString(36);
  }

  function now() { return new Date().toISOString(); }

  function loadData() {
    try {
      var raw = localStorage.getItem(STORAGE_KEY);
      if (raw) return JSON.parse(raw);
    } catch (e) {}
    return null;
  }

  function saveData(data) {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(data));
  }

  function loadSettings() {
    try {
      var raw = localStorage.getItem(SETTINGS_KEY);
      if (raw) return JSON.parse(raw);
    } catch (e) {}
    return null;
  }

  function saveSettings(settings) {
    localStorage.setItem(SETTINGS_KEY, JSON.stringify(settings));
  }

  var defaultSettings = {
    site_name: 'LiveBoard',
    theme: 'system',
    color_theme: 'aqua',
    font_family: 'system',
    column_width: 280,
    sidebar_position: 'left',
    show_checkbox: true,
    newline_trigger: 'shift-enter',
    card_position: 'append',
    card_display_mode: 'full',
    keyboard_shortcuts: false,
    default_columns: ['To Do', 'In Progress', 'Done']
  };

  var sampleBoard = {
    slug: 'getting-started',
    name: 'Getting Started',
    icon: '\u{1F680}',
    description: 'Welcome to LiveBoard! This is a sample board.',
    tags: ['sample'],
    tag_colors: {},
    members: ['you'],
    list_collapse: [],
    pinned: false,
    settings: {},
    columns: [
      {
        name: 'To Do',
        cards: [
          { title: 'Try dragging this card', completed: false, tags: ['sample'], priority: '', due: '', assignee: '', body: 'Drag me to another column!', metadata: {} },
          { title: 'Right-click for quick edit', completed: false, tags: [], priority: 'medium', due: '', assignee: '', body: '', metadata: {} },
          { title: 'Click to open detail modal', completed: false, tags: [], priority: '', due: '', assignee: 'you', body: 'You can edit title, body, tags, priority, and more.', metadata: {} }
        ]
      },
      {
        name: 'In Progress',
        cards: [
          { title: 'Explore board settings', completed: false, tags: [], priority: 'high', due: '', assignee: '', body: 'Click the gear icon next to the board title.', metadata: {} }
        ]
      },
      {
        name: 'Done',
        cards: [
          { title: 'Open LiveBoard', completed: true, tags: ['sample'], priority: '', due: '', assignee: '', body: '', metadata: {} }
        ]
      }
    ],
    created_at: now(),
    updated_at: now()
  };

  function getDefaultData() {
    return { boards: [JSON.parse(JSON.stringify(sampleBoard))] };
  }

  // ── Alpine Store Registration ──────────────────────────────────────
  document.addEventListener('alpine:init', function () {

    var data = loadData() || getDefaultData();
    var settings = loadSettings() || JSON.parse(JSON.stringify(defaultSettings));

    Alpine.store('lb', {
      boards: data.boards,
      settings: settings,

      // ── Persistence ──────────────────────────────────────────────
      _save: function () {
        saveData({ boards: this.boards });
      },
      _saveSettings: function () {
        saveSettings(this.settings);
      },

      // ── Board CRUD ───────────────────────────────────────────────
      getBoard: function (slug) {
        for (var i = 0; i < this.boards.length; i++) {
          if (this.boards[i].slug === slug) return this.boards[i];
        }
        return null;
      },

      createBoard: function (name) {
        var cols = (this.settings.default_columns || ['To Do', 'In Progress', 'Done']).map(function (n) {
          return { name: n, cards: [] };
        });
        var board = {
          slug: generateSlug(name),
          name: name,
          icon: '',
          description: '',
          tags: [],
          tag_colors: {},
          members: [],
          list_collapse: [],
          pinned: false,
          settings: {},
          columns: cols,
          created_at: now(),
          updated_at: now()
        };
        this.boards.push(board);
        this._save();
        return board;
      },

      deleteBoard: function (slug) {
        this.boards = this.boards.filter(function (b) { return b.slug !== slug; });
        this._save();
      },

      updateBoardMeta: function (slug, meta) {
        var b = this.getBoard(slug);
        if (!b) return;
        if (meta.name !== undefined) b.name = meta.name;
        if (meta.description !== undefined) b.description = meta.description;
        if (meta.tags !== undefined) b.tags = meta.tags;
        if (meta.tag_colors !== undefined) b.tag_colors = meta.tag_colors;
        if (meta.icon !== undefined) b.icon = meta.icon;
        b.updated_at = now();
        this._save();
      },

      updateBoardSettings: function (slug, s) {
        var b = this.getBoard(slug);
        if (!b) return;
        b.settings = Object.assign(b.settings || {}, s);
        b.updated_at = now();
        this._save();
      },

      togglePin: function (slug) {
        var b = this.getBoard(slug);
        if (!b) return;
        b.pinned = !b.pinned;
        this._save();
      },

      setIcon: function (slug, icon) {
        var b = this.getBoard(slug);
        if (!b) return;
        b.icon = icon;
        b.updated_at = now();
        this._save();
      },

      // ── Column CRUD ──────────────────────────────────────────────
      addColumn: function (slug, colName) {
        var b = this.getBoard(slug);
        if (!b) return;
        b.columns.push({ name: colName, cards: [] });
        b.updated_at = now();
        this._save();
      },

      renameColumn: function (slug, oldName, newName) {
        var b = this.getBoard(slug);
        if (!b) return;
        for (var i = 0; i < b.columns.length; i++) {
          if (b.columns[i].name === oldName) {
            b.columns[i].name = newName;
            break;
          }
        }
        b.updated_at = now();
        this._save();
      },

      deleteColumn: function (slug, colName) {
        var b = this.getBoard(slug);
        if (!b) return;
        b.columns = b.columns.filter(function (c) { return c.name !== colName; });
        b.updated_at = now();
        this._save();
      },

      moveColumn: function (slug, colName, afterColName) {
        var b = this.getBoard(slug);
        if (!b) return;
        var idx = -1;
        for (var i = 0; i < b.columns.length; i++) {
          if (b.columns[i].name === colName) { idx = i; break; }
        }
        if (idx < 0) return;
        var col = b.columns.splice(idx, 1)[0];
        if (!afterColName) {
          b.columns.unshift(col);
        } else {
          var afterIdx = -1;
          for (var i = 0; i < b.columns.length; i++) {
            if (b.columns[i].name === afterColName) { afterIdx = i; break; }
          }
          b.columns.splice(afterIdx + 1, 0, col);
        }
        b.updated_at = now();
        this._save();
      },

      sortColumn: function (slug, colIdx, sortBy) {
        var b = this.getBoard(slug);
        if (!b || colIdx < 0 || colIdx >= b.columns.length) return;
        var cards = b.columns[colIdx].cards;
        var priorityOrder = { critical: 0, high: 1, medium: 2, low: 3, '': 4 };
        if (sortBy === 'name') {
          cards.sort(function (a, b) { return a.title.toLowerCase().localeCompare(b.title.toLowerCase()); });
        } else if (sortBy === 'priority') {
          cards.sort(function (a, b) { return (priorityOrder[a.priority] || 4) - (priorityOrder[b.priority] || 4); });
        } else if (sortBy === 'due') {
          cards.sort(function (a, b) { return (a.due || '9999') < (b.due || '9999') ? -1 : 1; });
        }
        b.updated_at = now();
        this._save();
      },

      // ── Card CRUD ────────────────────────────────────────────────
      addCard: function (slug, colName, title, position) {
        var b = this.getBoard(slug);
        if (!b) return;
        for (var i = 0; i < b.columns.length; i++) {
          if (b.columns[i].name === colName) {
            var card = { title: title, completed: false, tags: [], priority: '', due: '', assignee: '', body: '', metadata: {} };
            if (position === 'prepend') {
              b.columns[i].cards.unshift(card);
            } else {
              b.columns[i].cards.push(card);
            }
            break;
          }
        }
        b.updated_at = now();
        this._save();
      },

      editCard: function (slug, colIdx, cardIdx, updates) {
        var b = this.getBoard(slug);
        if (!b) return;
        var card = b.columns[colIdx] && b.columns[colIdx].cards[cardIdx];
        if (!card) return;
        if (updates.title !== undefined) card.title = updates.title;
        if (updates.body !== undefined) card.body = updates.body;
        if (updates.tags !== undefined) card.tags = updates.tags;
        if (updates.priority !== undefined) card.priority = updates.priority;
        if (updates.due !== undefined) card.due = updates.due;
        if (updates.assignee !== undefined) card.assignee = updates.assignee;
        b.updated_at = now();
        this._save();
      },

      deleteCard: function (slug, colIdx, cardIdx) {
        var b = this.getBoard(slug);
        if (!b) return;
        if (b.columns[colIdx]) {
          b.columns[colIdx].cards.splice(cardIdx, 1);
        }
        b.updated_at = now();
        this._save();
      },

      toggleComplete: function (slug, colIdx, cardIdx) {
        var b = this.getBoard(slug);
        if (!b) return;
        var card = b.columns[colIdx] && b.columns[colIdx].cards[cardIdx];
        if (!card) return;
        card.completed = !card.completed;
        b.updated_at = now();
        this._save();
      },

      moveCard: function (slug, colIdx, cardIdx, targetColumn) {
        var b = this.getBoard(slug);
        if (!b) return;
        var card = b.columns[colIdx] && b.columns[colIdx].cards[cardIdx];
        if (!card) return;
        b.columns[colIdx].cards.splice(cardIdx, 1);
        for (var i = 0; i < b.columns.length; i++) {
          if (b.columns[i].name === targetColumn) {
            b.columns[i].cards.push(card);
            break;
          }
        }
        b.updated_at = now();
        this._save();
      },

      reorderCard: function (slug, srcColIdx, srcCardIdx, targetColumn, beforeIdx) {
        var b = this.getBoard(slug);
        if (!b) return;
        var card = b.columns[srcColIdx] && b.columns[srcColIdx].cards[srcCardIdx];
        if (!card) return;
        b.columns[srcColIdx].cards.splice(srcCardIdx, 1);
        var targetColIdx = -1;
        for (var i = 0; i < b.columns.length; i++) {
          if (b.columns[i].name === targetColumn) { targetColIdx = i; break; }
        }
        if (targetColIdx < 0) return;
        if (beforeIdx < 0 || beforeIdx >= b.columns[targetColIdx].cards.length) {
          b.columns[targetColIdx].cards.push(card);
        } else {
          b.columns[targetColIdx].cards.splice(beforeIdx, 0, card);
        }
        b.updated_at = now();
        this._save();
      },

      // ── Settings ─────────────────────────────────────────────────
      updateSettings: function (s) {
        Object.assign(this.settings, s);
        this._saveSettings();
      },

      // ── Reset ────────────────────────────────────────────────────
      reset: function () {
        var d = getDefaultData();
        this.boards = d.boards;
        this.settings = JSON.parse(JSON.stringify(defaultSettings));
        this._save();
        this._saveSettings();
      },

      // ── Helpers ──────────────────────────────────────────────────
      allTags: function () {
        var tags = [];
        this.boards.forEach(function (b) {
          b.tags.forEach(function (t) { if (tags.indexOf(t) === -1) tags.push(t); });
          b.columns.forEach(function (col) {
            col.cards.forEach(function (card) {
              (card.tags || []).forEach(function (t) { if (tags.indexOf(t) === -1) tags.push(t); });
            });
          });
        });
        return tags.sort();
      },

      boardTags: function (slug) {
        var b = this.getBoard(slug);
        if (!b) return [];
        var tags = [];
        b.tags.forEach(function (t) { if (tags.indexOf(t) === -1) tags.push(t); });
        b.columns.forEach(function (col) {
          col.cards.forEach(function (card) {
            (card.tags || []).forEach(function (t) { if (tags.indexOf(t) === -1) tags.push(t); });
          });
        });
        return tags.sort();
      },

      boardMembers: function (slug) {
        var b = this.getBoard(slug);
        if (!b) return [];
        var members = b.members.slice();
        b.columns.forEach(function (col) {
          col.cards.forEach(function (card) {
            if (card.assignee && members.indexOf(card.assignee) === -1) members.push(card.assignee);
          });
        });
        return members;
      },

      boardCardCount: function (slug) {
        var b = this.getBoard(slug);
        if (!b) return { total: 0, done: 0 };
        var total = 0, done = 0;
        b.columns.forEach(function (col) {
          col.cards.forEach(function (card) {
            total++;
            if (card.completed) done++;
          });
        });
        return { total: total, done: done };
      },

      sortedBoards: function () {
        return this.boards.slice().sort(function (a, b) {
          if (a.pinned && !b.pinned) return -1;
          if (!a.pinned && b.pinned) return 1;
          return (b.updated_at || '') < (a.updated_at || '') ? -1 : 1;
        });
      },

      // Effective setting: board-level overrides global
      effectiveSetting: function (slug, key) {
        var b = this.getBoard(slug);
        if (b && b.settings && b.settings[key] !== undefined && b.settings[key] !== '') {
          return b.settings[key];
        }
        return this.settings[key];
      }
    });

    // Also register the UI store (same as server version)
    Alpine.store('ui', {
      activeModal: null,
      isDragging: false,
      sidebarCollapsed: localStorage.getItem('sidebarCollapsed') === 'true',
      activeTag: null,
      focusedColumn: '',
      openModal: function (name) { this.activeModal = name; },
      closeModal: function (name) { if (this.activeModal === name) this.activeModal = null; },
      isModalOpen: function () { return this.activeModal !== null; },
      toggleSidebar: function () {
        this.sidebarCollapsed = !this.sidebarCollapsed;
        localStorage.setItem('sidebarCollapsed', this.sidebarCollapsed);
      }
    });

    // Board store compatibility shim — some existing JS components reference Alpine.store('board')
    Alpine.store('board', {
      slug: '',
      version: '0',
      tags: [],
      members: [],
      refresh: function () {
        var slug = Alpine.store('lb')._currentSlug || '';
        this.slug = slug;
        this.tags = Alpine.store('lb').boardTags(slug);
        this.members = Alpine.store('lb').boardMembers(slug);
      }
    });
  });
})();
