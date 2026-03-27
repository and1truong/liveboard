// LiveBoard Online: Alpine component templates for client-side rendering.
// Replaces server-rendered Go HTML templates with Alpine.js reactive rendering.

document.addEventListener('alpine:init', function () {

  // ── Main App Component ───────────────────────────────────────────
  Alpine.data('lbApp', function () {
    return {
      route: LBRouter.parse(),

      init: function () {
        var self = this;
        LBRouter.init();
        LBRouter.onChange(function (r) {
          self.route = r;
          // Update board store when navigating to a board
          if (r.view === 'board') {
            Alpine.store('lb')._currentSlug = r.slug;
            Alpine.store('board').refresh();
          }
          // Update page title
          self.updateTitle();
        });
        if (this.route.view === 'board') {
          Alpine.store('lb')._currentSlug = this.route.slug;
          Alpine.store('board').refresh();
        }
        this.updateTitle();
      },

      updateTitle: function () {
        var siteName = Alpine.store('lb').settings.site_name || 'LiveBoard';
        if (this.route.view === 'board') {
          var b = Alpine.store('lb').getBoard(this.route.slug);
          document.title = (b ? b.name : 'Board') + ' \u2014 ' + siteName;
        } else if (this.route.view === 'settings') {
          document.title = 'Settings \u2014 ' + siteName;
        } else {
          document.title = siteName;
        }
      },

      get isHome() { return this.route.view === 'home'; },
      get isBoard() { return this.route.view === 'board'; },
      get isSettings() { return this.route.view === 'settings'; },
      get currentBoard() { return this.route.view === 'board' ? Alpine.store('lb').getBoard(this.route.slug) : null; },
      get lb() { return Alpine.store('lb'); }
    };
  });

  // ── Board List Component ─────────────────────────────────────────
  Alpine.data('boardList', function () {
    return {
      showForm: false,
      newName: '',

      createBoard: function () {
        var name = this.newName.trim();
        if (!name) return;
        var board = Alpine.store('lb').createBoard(name);
        this.newName = '';
        this.showForm = false;
        LBRouter.navigate('#/board/' + encodeURIComponent(board.slug));
      },

      deleteBoard: function (slug, name) {
        if (confirm('Delete board "' + name + '"? This cannot be undone.')) {
          Alpine.store('lb').deleteBoard(slug);
        }
      },

      cardCount: function (slug) { return Alpine.store('lb').boardCardCount(slug); },
      get boards() { return Alpine.store('lb').sortedBoards(); }
    };
  });

  // ── Board View Component ─────────────────────────────────────────
  Alpine.data('boardView', function () {
    return {
      addCardCol: null,
      addCardTitle: '',
      addColName: '',
      showAddCol: false,

      get board() {
        return Alpine.store('lb').getBoard(Alpine.store('lb')._currentSlug || '');
      },

      get slug() { return this.board ? this.board.slug : ''; },

      get viewMode() {
        return Alpine.store('lb').effectiveSetting(this.slug, 'view_mode') || 'board';
      },

      get showCheckbox() {
        var v = Alpine.store('lb').effectiveSetting(this.slug, 'show_checkbox');
        return v === true || v === 'true';
      },

      get cardDisplayMode() {
        return Alpine.store('lb').effectiveSetting(this.slug, 'card_display_mode') || 'full';
      },

      get expandColumns() {
        var v = Alpine.store('lb').effectiveSetting(this.slug, 'expand_columns');
        return v === true || v === 'true';
      },

      get cardPosition() {
        return Alpine.store('lb').effectiveSetting(this.slug, 'card_position') || 'append';
      },

      showAddCard: function (colName) {
        this.addCardCol = colName;
        this.addCardTitle = '';
        this.$nextTick(function () {
          var col = document.querySelector('.cards[data-column="' + colName + '"]');
          var input = col ? col.closest('.column').querySelector('.online-add-card-input') : document.querySelector('.online-add-card-input');
          if (input) input.focus();
        });
      },

      submitAddCard: function () {
        var title = this.addCardTitle.trim();
        if (!title) return;
        Alpine.store('lb').addCard(this.slug, this.addCardCol, title, this.cardPosition);
        this.addCardTitle = '';
        Alpine.store('board').refresh();
      },

      cancelAddCard: function () { this.addCardCol = null; this.addCardTitle = ''; },

      submitAddColumn: function () {
        var name = this.addColName.trim();
        if (!name) return;
        Alpine.store('lb').addColumn(this.slug, name);
        this.addColName = '';
        this.showAddCol = false;
      },

      toggleComplete: function (colIdx, cardIdx) {
        Alpine.store('lb').toggleComplete(this.slug, colIdx, cardIdx);
        Alpine.store('board').refresh();
      },

      deleteCard: function (colIdx, cardIdx) {
        Alpine.store('lb').deleteCard(this.slug, colIdx, cardIdx);
        Alpine.store('board').refresh();
      },

      moveCard: function (colIdx, cardIdx, targetColumn) {
        Alpine.store('lb').moveCard(this.slug, colIdx, cardIdx, targetColumn);
        Alpine.store('board').refresh();
      },

      cardTagStyle: function (tag) {
        var b = this.board;
        if (!b || !b.tag_colors || !b.tag_colors[tag]) return '';
        var bg = b.tag_colors[tag];
        var lum = window.LB && window.LB.colorLuminance ? window.LB.colorLuminance(bg) : 0.5;
        return 'background:' + bg + ';color:' + (lum > 0.35 ? '#111' : '#fff') + ';border-color:transparent';
      }
    };
  });

  // ── Card Modal (online version) ──────────────────────────────────
  Alpine.data('onlineCardModal', function () {
    return {
      open: false,
      title: '', body: '', tags: [], priority: '', due: '', assignee: '',
      completed: false, columnName: '', colIdx: -1, cardIdx: -1,
      showDatePicker: false, showMembersPicker: false, showBodyPreview: false,

      show: function (colIdx, cardIdx) {
        var slug = Alpine.store('lb')._currentSlug;
        var b = Alpine.store('lb').getBoard(slug);
        if (!b || !b.columns[colIdx] || !b.columns[colIdx].cards[cardIdx]) return;
        var card = b.columns[colIdx].cards[cardIdx];
        this.colIdx = colIdx;
        this.cardIdx = cardIdx;
        this.title = card.title || '';
        this.body = card.body || '';
        this.tags = (card.tags || []).slice();
        this.priority = card.priority || '';
        this.due = card.due || '';
        this.assignee = card.assignee || '';
        this.completed = !!card.completed;
        this.columnName = b.columns[colIdx].name;
        this.showDatePicker = false;
        this.showMembersPicker = false;
        this.showBodyPreview = false;
        Alpine.store('ui').openModal('cardModal');
        this.open = true;
      },

      close: function () {
        this.open = false;
        Alpine.store('ui').closeModal('cardModal');
      },

      save: function () {
        var slug = Alpine.store('lb')._currentSlug;
        Alpine.store('lb').editCard(slug, this.colIdx, this.cardIdx, {
          title: this.title.trim(),
          body: this.body.trim(),
          tags: this.tags,
          priority: this.priority,
          due: this.due,
          assignee: this.assignee
        });
        Alpine.store('board').refresh();
        this.close();
      },

      toggleComplete: function () {
        var slug = Alpine.store('lb')._currentSlug;
        Alpine.store('lb').toggleComplete(slug, this.colIdx, this.cardIdx);
        this.completed = !this.completed;
        Alpine.store('board').refresh();
      },

      deleteCard: function () {
        var slug = Alpine.store('lb')._currentSlug;
        Alpine.store('lb').deleteCard(slug, this.colIdx, this.cardIdx);
        Alpine.store('board').refresh();
        this.close();
      },

      moveTo: function (targetCol) {
        var slug = Alpine.store('lb')._currentSlug;
        Alpine.store('lb').moveCard(slug, this.colIdx, this.cardIdx, targetCol);
        Alpine.store('board').refresh();
        this.close();
      },

      get otherColumns() {
        var slug = Alpine.store('lb')._currentSlug;
        var b = Alpine.store('lb').getBoard(slug);
        if (!b) return [];
        var self = this;
        return b.columns.filter(function (c) { return c.name !== self.columnName; }).map(function (c) { return c.name; });
      },

      renderMarkdown: function (text) {
        if (!text) return '';
        var s = text
          .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
          .replace(/^### (.+)$/gm, '<h3>$1</h3>')
          .replace(/^## (.+)$/gm, '<h2>$1</h2>')
          .replace(/^# (.+)$/gm, '<h1>$1</h1>')
          .replace(/`([^`]+)`/g, '<code>$1</code>')
          .replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
          .replace(/__([^_]+)__/g, '<strong>$1</strong>')
          .replace(/\*([^*]+)\*/g, '<em>$1</em>')
          .replace(/_([^_]+)_/g, '<em>$1</em>')
          .replace(/~~([^~]+)~~/g, '<del>$1</del>')
          .replace(/\[([^\]]+)\]\(([^)]+)\)/g, function (m, text, url) {
            if (/^\s*(javascript|data|vbscript):/i.test(url)) return text;
            return '<a href="' + url + '" target="_blank" rel="noopener">' + text + '</a>';
          })
          .replace(/^[-*] (.+)$/gm, '<li>$1</li>')
          .replace(/(<li>.*<\/li>)/gs, '<ul>$1</ul>')
          .replace(/<\/ul>\s*<ul>/g, '');
        var parts = s.split(/\n\n+/);
        return parts.map(function (p) {
          p = p.trim();
          if (!p) return '';
          if (/^<[hul]/.test(p)) return p;
          return '<p>' + p.replace(/\n/g, '<br>') + '</p>';
        }).join('\n');
      },

      autoResize: function (e) {
        var el = e.target;
        el.style.height = 'auto';
        el.style.height = el.scrollHeight + 'px';
      }
    };
  });

  // ── Online Column Menu ───────────────────────────────────────────
  Alpine.data('onlineColumnMenu', function () {
    return {
      open: false,
      left: 0, top: 0,
      columnName: '', colIdx: -1,

      show: function (e, colName, colIdx) {
        var btn = e.currentTarget;
        var rect = btn.getBoundingClientRect();
        this.columnName = colName;
        this.colIdx = colIdx;
        this.left = rect.left;
        this.top = rect.bottom + 4;
        Alpine.store('ui').openModal('columnMenu');
        this.open = true;
      },

      hide: function () {
        this.open = false;
        Alpine.store('ui').closeModal('columnMenu');
      },

      renameColumn: function () {
        this.hide();
        var newName = prompt('Rename column:', this.columnName);
        if (newName && newName.trim() && newName.trim() !== this.columnName) {
          var slug = Alpine.store('lb')._currentSlug;
          Alpine.store('lb').renameColumn(slug, this.columnName, newName.trim());
        }
      },

      deleteColumn: function () {
        this.hide();
        if (confirm('Delete column "' + this.columnName + '" and all its cards?')) {
          var slug = Alpine.store('lb')._currentSlug;
          Alpine.store('lb').deleteColumn(slug, this.columnName);
        }
      },

      sortBy: function (field) {
        this.hide();
        var slug = Alpine.store('lb')._currentSlug;
        Alpine.store('lb').sortColumn(slug, this.colIdx, field);
      },

      focusColumn: function () {
        var name = this.columnName;
        this.hide();
        Alpine.store('ui').focusedColumn = name;
      }
    };
  });

  // ── Online Board Settings Panel ──────────────────────────────────
  Alpine.data('onlineBoardSettings', function () {
    return {
      open: false,
      boardName: '', boardDescription: '', tags: [], tagColors: {},
      showCheckbox: '', cardPosition: '', expandColumns: 'false',
      viewMode: 'board', cardDisplayMode: '', weekStart: '',
      TAG_PALETTE: ['#e05252','#d4722c','#c9a227','#4caf76','#45aab5','#4080c4','#8060c4','#c060a0','#607080','#a07040'],

      toggle: function () {
        if (this.open) { this.close(); return; }
        this.populate();
        Alpine.store('ui').openModal('boardSettings');
        this.open = true;
      },

      close: function () { this.open = false; Alpine.store('ui').closeModal('boardSettings'); },

      populate: function () {
        var slug = Alpine.store('lb')._currentSlug;
        var b = Alpine.store('lb').getBoard(slug);
        if (!b) return;
        this.boardName = b.name || '';
        this.boardDescription = b.description || '';
        this.tags = (b.tags || []).slice();
        this.tagColors = Object.assign({}, b.tag_colors || {});
        var s = b.settings || {};
        this.showCheckbox = s.show_checkbox !== undefined ? String(s.show_checkbox) : '';
        this.cardPosition = s.card_position || '';
        this.expandColumns = s.expand_columns !== undefined ? String(s.expand_columns) : 'false';
        this.viewMode = s.view_mode || 'board';
        this.cardDisplayMode = s.card_display_mode || '';
        this.weekStart = s.week_start || '';
      },

      applySettings: function () {
        var slug = Alpine.store('lb')._currentSlug;
        Alpine.store('lb').updateBoardMeta(slug, {
          name: this.boardName.trim(),
          description: this.boardDescription.trim(),
          tags: this.tags,
          tag_colors: this.tagColors
        });
        var s = {};
        if (this.showCheckbox !== '') s.show_checkbox = this.showCheckbox;
        if (this.cardPosition !== '') s.card_position = this.cardPosition;
        s.expand_columns = this.expandColumns;
        s.view_mode = this.viewMode;
        if (this.cardDisplayMode !== '') s.card_display_mode = this.cardDisplayMode;
        if (this.weekStart !== '') s.week_start = this.weekStart;
        Alpine.store('lb').updateBoardSettings(slug, s);
        Alpine.store('board').refresh();
        this.close();
      },

      get tagSuggestions() {
        var slug = Alpine.store('lb')._currentSlug;
        var all = Alpine.store('lb').boardTags(slug).slice();
        this.tags.forEach(function (t) { if (all.indexOf(t) === -1) all.push(t); });
        return all.sort();
      },

      getTagPreviewStyle: function (tag) {
        var bg = this.tagColors[tag];
        if (!bg) return '';
        var lum = window.LB && window.LB.colorLuminance ? window.LB.colorLuminance(bg) : 0.5;
        return 'background:' + bg + ';color:' + (lum > 0.35 ? '#111' : '#fff') + ';border-color:transparent';
      },

      toggleTagColor: function (tag, color) {
        if (this.tagColors[tag] === color) delete this.tagColors[tag];
        else this.tagColors[tag] = color;
      },

      clearTagColor: function (tag) { delete this.tagColors[tag]; },

      addTag: function (tag) {
        tag = tag.trim();
        if (tag && this.tags.indexOf(tag) === -1) this.tags.push(tag);
      },

      removeTag: function (idx) { this.tags.splice(idx, 1); }
    };
  });

  // ── Online Global Settings ───────────────────────────────────────
  Alpine.data('onlineGlobalSettings', function () {
    return {
      siteName: '',
      theme: 'system',
      colorTheme: 'aqua',
      fontFamily: 'system',
      columnWidth: 280,
      sidebarPosition: 'left',
      showCheckbox: true,
      newLineTrigger: 'shift-enter',
      cardPosition: 'append',
      cardDisplayMode: 'full',
      keyboardShortcuts: false,
      defaultColumns: [],

      fontMap: {
        'system': { css: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif", gf: '' },
        'inter': { css: "'Inter', sans-serif", gf: 'Inter' },
        'ibm-plex-sans': { css: "'IBM Plex Sans', sans-serif", gf: 'IBM+Plex+Sans' },
        'source-sans-3': { css: "'Source Sans 3', sans-serif", gf: 'Source+Sans+3' },
        'nunito-sans': { css: "'Nunito Sans', sans-serif", gf: 'Nunito+Sans' },
        'dm-sans': { css: "'DM Sans', sans-serif", gf: 'DM+Sans' },
        'rubik': { css: "'Rubik', sans-serif", gf: 'Rubik' }
      },

      init: function () {
        var s = Alpine.store('lb').settings;
        this.siteName = s.site_name || 'LiveBoard';
        this.theme = s.theme || 'system';
        this.colorTheme = s.color_theme || 'aqua';
        this.fontFamily = s.font_family || 'system';
        this.columnWidth = s.column_width || 280;
        this.sidebarPosition = s.sidebar_position || 'left';
        this.showCheckbox = s.show_checkbox !== false;
        this.newLineTrigger = s.newline_trigger || 'shift-enter';
        this.cardPosition = s.card_position || 'append';
        this.cardDisplayMode = s.card_display_mode || 'full';
        this.keyboardShortcuts = !!s.keyboard_shortcuts;
        this.defaultColumns = (s.default_columns || ['To Do', 'In Progress', 'Done']).slice();
      },

      save: function () {
        var payload = {
          site_name: this.siteName.trim() || 'LiveBoard',
          theme: this.theme,
          color_theme: this.colorTheme,
          font_family: this.fontFamily,
          column_width: parseInt(this.columnWidth, 10) || 280,
          sidebar_position: this.sidebarPosition,
          show_checkbox: !!this.showCheckbox,
          newline_trigger: this.newLineTrigger,
          card_position: this.cardPosition,
          card_display_mode: this.cardDisplayMode,
          keyboard_shortcuts: !!this.keyboardShortcuts,
          default_columns: this.defaultColumns.length ? this.defaultColumns : ['To Do', 'In Progress', 'Done']
        };
        Alpine.store('lb').updateSettings(payload);
        this.applyVisuals(payload);
      },

      applyVisuals: function (s) {
        var el = document.documentElement;
        if (s.theme === 'system') el.removeAttribute('data-theme');
        else el.setAttribute('data-theme', s.theme);
        el.setAttribute('data-color-theme', s.color_theme || 'aqua');
        el.style.setProperty('--column-width', s.column_width + 'px');
        if (s.sidebar_position === 'right') el.setAttribute('data-sidebar-position', 'right');
        else el.removeAttribute('data-sidebar-position');

        // Font
        var f = this.fontMap[s.font_family] || this.fontMap['system'];
        el.style.setProperty('--font-sans', f.css);
        var existing = document.getElementById('lb-google-font');
        if (existing) existing.remove();
        if (f.gf) {
          var link = document.createElement('link');
          link.id = 'lb-google-font';
          link.rel = 'stylesheet';
          link.href = 'https://fonts.googleapis.com/css2?family=' + f.gf + ':wght@400;500;600;700&display=swap';
          document.head.appendChild(link);
        }

        // Brand name
        var brandEl = document.querySelector('.brand-name');
        if (brandEl) brandEl.textContent = s.site_name;
      }
    };
  });

  // ── Online Emoji Picker ──────────────────────────────────────────
  Alpine.data('onlineEmojiPicker', function () {
    return {
      open: false,
      top: 0, left: 0,
      targetSlug: '',
      emojis: ['\u{1F680}','\u{1F4CB}','\u{2B50}','\u{1F525}','\u{1F4A1}','\u{1F3AF}','\u{1F527}','\u{1F4DA}','\u{1F381}','\u{1F30D}',
               '\u{2764}\uFE0F','\u{1F4E6}','\u{1F389}','\u{1F6A7}','\u{1F3C6}','\u{1F4DD}','\u{1F512}','\u{26A1}','\u{1F331}','\u{1F41B}',
               '\u{1F504}','\u{1F4AC}','\u{1F3E0}','\u{1F4CA}','\u{1F4C5}','\u{1F50D}','\u{2705}','\u{274C}','\u{1F514}','\u{2699}\uFE0F'],

      show: function (el, slug) {
        var rect = el.getBoundingClientRect();
        this.top = rect.bottom + 4;
        this.left = rect.left;
        this.targetSlug = slug;
        this.open = true;
      },

      pick: function (emoji) {
        Alpine.store('lb').setIcon(this.targetSlug, emoji);
        this.open = false;
      },

      clear: function () {
        Alpine.store('lb').setIcon(this.targetSlug, '');
        this.open = false;
      }
    };
  });
});
