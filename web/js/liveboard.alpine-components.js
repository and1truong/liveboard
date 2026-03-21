// LiveBoard: Alpine.js reusable components.
document.addEventListener('alpine:init', function () {

  // ── Tag Chips ────────────────────────────────────────────────────────
  // Usage: x-data="tagChips()"
  // Parent scope MUST provide: `tags` (array) and `tagSuggestions` (array)
  // tagChips does NOT own `tags` — it inherits from the parent via Alpine scope chain.
  Alpine.data('tagChips', function () {
    return {
      // `tags` and `tagSuggestions` inherited from parent x-data scope
      tcInput: '',
      tcDropdownOpen: false,
      tcActiveIdx: -1,

      get filtered() {
        var self = this;
        var f = this.tcInput.toLowerCase();
        var suggestions = this.tagSuggestions || [];
        return suggestions.filter(function (t) {
          return self.tags.indexOf(t) === -1 && (!f || t.toLowerCase().indexOf(f) !== -1);
        });
      },

      addTag: function (tag) {
        tag = tag.trim();
        if (!tag || this.tags.indexOf(tag) !== -1) return;
        this.tags.push(tag);
        this.tcInput = '';
        this.tcDropdownOpen = false;
        this.tcActiveIdx = -1;
      },

      removeTag: function (idx) {
        this.tags.splice(idx, 1);
      },

      showDropdown: function () {
        this.tcDropdownOpen = true;
        this.tcActiveIdx = -1;
      },

      hideDropdown: function () {
        var self = this;
        setTimeout(function () { self.tcDropdownOpen = false; self.tcActiveIdx = -1; }, 150);
      },

      handleKeydown: function (e) {
        if (e.key === 'ArrowDown') {
          e.preventDefault();
          this.tcActiveIdx = Math.min(this.tcActiveIdx + 1, this.filtered.length - 1);
        } else if (e.key === 'ArrowUp') {
          e.preventDefault();
          this.tcActiveIdx = Math.max(this.tcActiveIdx - 1, 0);
        } else if (e.key === 'Enter') {
          e.preventDefault();
          if (this.tcActiveIdx >= 0 && this.filtered[this.tcActiveIdx]) {
            this.addTag(this.filtered[this.tcActiveIdx]);
          } else if (this.tcInput.trim()) {
            this.addTag(this.tcInput);
          }
        } else if (e.key === 'Backspace' && !this.tcInput && this.tags.length) {
          this.tags.pop();
        } else if (e.key === 'Escape') {
          this.tcDropdownOpen = false;
          this.tcActiveIdx = -1;
        }
      },

      getTagsValue: function () {
        return this.tags.join(', ');
      }
    };
  });

  // ── Column Chips (settings page, no dropdown) ────────────────────────
  Alpine.data('columnChips', function (initial) {
    return {
      cols: initial || [],
      input: '',

      addCol: function (val) {
        val = (val || '').replace(/,/g, '').trim();
        if (!val) return;
        this.cols.push(val);
        this.input = '';
      },

      removeCol: function (idx) {
        this.cols.splice(idx, 1);
      },

      handleKeydown: function (e) {
        if (e.key === 'Enter' || e.key === ',') {
          e.preventDefault();
          this.addCol(this.input);
        } else if (e.key === 'Backspace' && !this.input && this.cols.length) {
          this.cols.pop();
        }
      }
    };
  });

  // ── Command Palette ──────────────────────────────────────────────────
  Alpine.data('cmdPalette', function (boards, activeSlug) {
    return {
      open: false,
      query: '',
      activeIdx: 0,

      get items() {
        var self = this;
        var path = window.location.pathname;
        var items = [];

        boards.forEach(function (b) {
          var url = '/board/' + b.slug;
          if (b.slug === activeSlug && path === url) return;
          items.push({ icon: b.icon || '\u2630', name: b.name, url: url });
        });

        var hasFixed = false;
        if (path !== '/') {
          if (!hasFixed) { items.push({ separator: true }); hasFixed = true; }
          items.push({ icon: '\uD83C\uDFE0', name: 'All Boards', url: '/' });
        }
        if (path !== '/settings') {
          if (!hasFixed) { items.push({ separator: true }); hasFixed = true; }
          items.push({ icon: '\u2699\uFE0F', name: 'Settings', url: '/settings' });
        }

        if (self.query) {
          var q = self.query.toLowerCase();
          items = items.filter(function (it) {
            return !it.separator && it.name.toLowerCase().indexOf(q) !== -1;
          });
        }
        return items;
      },

      get selectableItems() {
        return this.items.filter(function (it) { return !it.separator; });
      },

      toggle: function () {
        this.open = !this.open;
        this.query = '';
        this.activeIdx = 0;
        if (this.open) {
          var self = this;
          this.$nextTick(function () {
            var inp = self.$refs.input;
            if (inp) inp.focus();
          });
        }
      },

      handleKeydown: function (e) {
        if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
          e.preventDefault();
          this.toggle();
          return;
        }
        if (!this.open) return;
        if (e.key === 'Escape') {
          e.preventDefault();
          this.open = false;
          return;
        }
        var count = this.selectableItems.length;
        if (e.key === 'ArrowDown') {
          e.preventDefault();
          if (count > 0) this.activeIdx = (this.activeIdx + 1) % count;
          return;
        }
        if (e.key === 'ArrowUp') {
          e.preventDefault();
          if (count > 0) this.activeIdx = (this.activeIdx - 1 + count) % count;
          return;
        }
        if (e.key === 'Enter') {
          e.preventDefault();
          var sel = this.selectableItems[this.activeIdx];
          if (sel) window.location.href = sel.url;
        }
      },

      onInput: function () {
        this.activeIdx = 0;
      }
    };
  });

  // ── Priority Selector ────────────────────────────────────────────────
  Alpine.data('prioritySelector', function (initial) {
    return {
      value: initial || '',
      options: [
        { val: '', label: '\u2014', title: 'None' },
        { val: 'low', label: 'L', title: 'Low' },
        { val: 'medium', label: 'M', title: 'Medium' },
        { val: 'high', label: 'H', title: 'High' },
        { val: 'critical', label: '!', title: 'Critical' }
      ],
      select: function (val) { this.value = val; }
    };
  });

  // ── Date Picker ──────────────────────────────────────────────────────
  Alpine.data('datePicker', function (currentDue, onSelect) {
    var now = new Date();
    var vy = now.getFullYear();
    var vm = now.getMonth();
    if (currentDue) {
      var parts = currentDue.split('-');
      if (parts.length === 3) {
        vy = parseInt(parts[0], 10);
        vm = parseInt(parts[1], 10) - 1;
      }
    }
    return {
      viewYear: vy,
      viewMonth: vm,
      selected: currentDue || '',
      monthNames: ['January','February','March','April','May','June',
        'July','August','September','October','November','December'],
      weekdays: ['Su','Mo','Tu','We','Th','Fr','Sa'],
      _onSelect: onSelect,

      get monthLabel() { return this.monthNames[this.viewMonth] + ' ' + this.viewYear; },

      get days() {
        var firstDay = new Date(this.viewYear, this.viewMonth, 1).getDay();
        var daysInMonth = new Date(this.viewYear, this.viewMonth + 1, 0).getDate();
        var result = [];
        for (var i = 0; i < firstDay; i++) result.push(null);
        for (var d = 1; d <= daysInMonth; d++) result.push(d);
        return result;
      },

      pad: function (n) { return n < 10 ? '0' + n : '' + n; },

      dateStr: function (d) {
        return this.viewYear + '-' + this.pad(this.viewMonth + 1) + '-' + this.pad(d);
      },

      get todayStr() {
        var t = new Date();
        return t.getFullYear() + '-' + this.pad(t.getMonth() + 1) + '-' + this.pad(t.getDate());
      },

      prevMonth: function () {
        this.viewMonth--;
        if (this.viewMonth < 0) { this.viewMonth = 11; this.viewYear--; }
      },

      nextMonth: function () {
        this.viewMonth++;
        if (this.viewMonth > 11) { this.viewMonth = 0; this.viewYear++; }
      },

      selectDate: function (d) {
        var val = this.dateStr(d);
        this.selected = val;
        if (this._onSelect) this._onSelect(val);
      },

      removeDate: function () {
        this.selected = '';
        if (this._onSelect) this._onSelect('');
      }
    };
  });

  // ── Members Picker ───────────────────────────────────────────────────
  Alpine.data('membersPicker', function (currentAssignee, boardMembers, onSelect) {
    return {
      members: (boardMembers || []).slice().sort(function (a, b) { return a.toLowerCase().localeCompare(b.toLowerCase()); }),
      current: currentAssignee || '',
      newMember: '',
      _onSelect: onSelect,

      addMember: function () {
        var name = this.newMember.trim();
        if (!name) return;
        if (this.members.indexOf(name) === -1) {
          this.members.push(name);
          this.members.sort(function (a, b) { return a.toLowerCase().localeCompare(b.toLowerCase()); });
        }
        this.newMember = '';
      },

      selectMember: function (m) {
        this.current = m;
        if (this._onSelect) this._onSelect(m);
      },

      clear: function () {
        this.current = '';
        if (this._onSelect) this._onSelect('');
      }
    };
  });

  // ── Card Modal ───────────────────────────────────────────────────────
  Alpine.data('cardModal', function () {
    return {
      open: false,
      title: '',
      body: '',
      tags: [],
      tagSuggestions: [],
      priority: '',
      due: '',
      assignee: '',
      completed: false,
      columnName: '',
      colIdx: '',
      cardIdx: '',
      slug: '',
      moveTriggers: [],
      hasCompleteBtn: false,
      hasDeleteBtn: false,
      showDatePicker: false,
      showMembersPicker: false,
      boardMembers: [],
      _cardEl: null,

      show: function (card) {
        this.slug = decodeURIComponent(window.location.pathname.replace(/^\/board\//, ''));
        this.colIdx = card.dataset.colIdx;
        this.cardIdx = card.dataset.cardIdx;
        this.title = card.dataset.cardTitle || '';
        this.body = card.dataset.cardBody || '';
        this.priority = card.dataset.cardPriority || '';
        this.due = card.dataset.cardDue || '';
        this.assignee = card.dataset.cardAssignee || '';
        this.completed = card.dataset.cardCompleted === 'true';
        this.columnName = card.dataset.cardColumn || '';
        this._cardEl = card;

        // Collect tags
        var rawTags = card.dataset.cardTags || '';
        this.tags = [];
        if (rawTags) {
          var self = this;
          rawTags.split(',').forEach(function (s) {
            s = s.trim();
            if (s && self.tags.indexOf(s) === -1) self.tags.push(s);
          });
        }

        // Collect all board tags
        this.tagSuggestions = [];
        var self = this;
        document.querySelectorAll('.card[data-card-tags]').forEach(function (c) {
          (c.dataset.cardTags || '').split(',').forEach(function (s) {
            s = s.trim();
            if (s && self.tagSuggestions.indexOf(s) === -1) self.tagSuggestions.push(s);
          });
        });
        this.tagSuggestions.sort(function (a, b) { return a.toLowerCase().localeCompare(b.toLowerCase()); });

        // Board members
        var boardView = document.querySelector('.board-view');
        var membersRaw = boardView ? (boardView.dataset.boardMembers || '') : '';
        this.boardMembers = membersRaw ? membersRaw.split(',').map(function (s) { return s.trim(); }).filter(Boolean) : [];
        document.querySelectorAll('[data-card-assignee]').forEach(function (c) {
          var a = c.dataset.cardAssignee;
          if (a && self.boardMembers.indexOf(a) === -1) self.boardMembers.push(a);
        });

        // Move triggers
        this.moveTriggers = [];
        Array.from(card.querySelectorAll('.move-trigger[data-target]')).forEach(function (t) {
          self.moveTriggers.push({ name: t.dataset.target, el: t });
        });

        this.hasCompleteBtn = !!card.querySelector('[hx-post$="/cards/complete"]');
        this.hasDeleteBtn = !!card.querySelector('[hx-post$="/cards/delete"]');

        this.showDatePicker = false;
        this.showMembersPicker = false;
        this.open = true;

        var modalSelf = this;
        this.$nextTick(function () {
          var ta = document.querySelector('.card-modal-title');
          if (ta) {
            ta.style.height = 'auto';
            ta.style.height = ta.scrollHeight + 'px';
            ta.focus();
          }
        });
      },

      close: function () {
        this.open = false;
        this._cardEl = null;
      },

      autoResize: function (e) {
        e.target.style.height = 'auto';
        e.target.style.height = e.target.scrollHeight + 'px';
      },

      save: function () {
        htmx.ajax('POST', '/board/' + encodeURIComponent(this.slug) + '/cards/edit', {
          values: {
            col_idx: this.colIdx,
            card_idx: this.cardIdx,
            title: this.title.trim(),
            body: this.body.trim(),
            tags: this.tags.join(', '),
            priority: this.priority,
            due: this.due,
            assignee: this.assignee,
            name: this.slug,
            version: window.LB.getBoardVersion()
          },
          target: '#board-content',
          swap: 'innerHTML'
        });
        this.close();
      },


      toggleComplete: function () {
        this.close();
        htmx.ajax('POST', '/board/' + encodeURIComponent(this.slug) + '/cards/complete', {
          values: { col_idx: this.colIdx, card_idx: this.cardIdx, name: this.slug, version: window.LB.getBoardVersion() },
          target: '#board-content',
          swap: 'innerHTML'
        });
      },

      moveTo: function (trigger) {
        this.close();
        trigger.el.click();
      },

      deleteCard: function () {
        if (!this._cardEl) return;
        var btn = this._cardEl.querySelector('[hx-post$="/cards/delete"]');
        this.close();
        if (btn) btn.click();
      },

      onDateSelect: function (val) {
        this.due = val;
        this.showDatePicker = false;
      },

      onMemberSelect: function (val) {
        this.assignee = val;
        this.showMembersPicker = false;
      }
    };
  });

  // ── Quick Edit + Context Menu ────────────────────────────────────────
  Alpine.data('quickEdit', function () {
    return {
      open: false,
      title: '',
      body: '',
      tags: [],
      tagSuggestions: [],
      priority: '',
      colIdx: '',
      cardIdx: '',
      slug: '',
      left: 0,
      top: 0,
      width: 0,
      minHeight: 0,
      _cardEl: null,

      // Context menu state
      ctxOpen: false,
      ctxLeft: 0,
      ctxTop: 0,
      ctxCompleted: false,
      ctxMoveTriggers: [],
      ctxHasComplete: false,
      ctxHasDelete: false,
      ctxDeleteArmed: false,
      ctxDeleteLabel: 'Delete',

      show: function (card) {
        this.hide();
        var cardRect = card.getBoundingClientRect();
        var posRect = cardRect;
        var cardCell = card.querySelector('.table-cell-card');
        if (cardCell) posRect = cardCell.getBoundingClientRect();

        this.slug = decodeURIComponent(window.location.pathname.replace(/^\/board\//, ''));
        this.colIdx = card.dataset.colIdx;
        this.cardIdx = card.dataset.cardIdx;
        this.title = card.dataset.cardTitle || '';
        this.body = card.dataset.cardBody || '';
        this.priority = card.dataset.cardPriority || '';
        this._cardEl = card;

        // Tags
        var rawTags = card.dataset.cardTags || '';
        this.tags = [];
        if (rawTags) {
          var self = this;
          rawTags.split(',').forEach(function (s) { s = s.trim(); if (s && self.tags.indexOf(s) === -1) self.tags.push(s); });
        }
        this.tagSuggestions = [];
        var self = this;
        document.querySelectorAll('.card[data-card-tags]').forEach(function (c) {
          (c.dataset.cardTags || '').split(',').forEach(function (s) { s = s.trim(); if (s && self.tagSuggestions.indexOf(s) === -1) self.tagSuggestions.push(s); });
        });
        this.tagSuggestions.sort(function (a, b) { return a.toLowerCase().localeCompare(b.toLowerCase()); });

        // Position
        this.left = posRect.left;
        this.top = cardRect.top;
        this.width = posRect.width;
        this.minHeight = cardRect.height;
        this.open = true;

        // Context menu
        this.ctxCompleted = card.classList.contains('completed');
        this.ctxHasComplete = !!card.querySelector('[hx-post$="/cards/complete"]');
        this.ctxHasDelete = !!card.querySelector('[hx-post$="/cards/delete"]');
        this.ctxDeleteArmed = false;
        this.ctxDeleteLabel = 'Delete';
        this.ctxMoveTriggers = [];
        Array.from(card.querySelectorAll('.move-trigger[data-target]')).forEach(function (t) {
          self.ctxMoveTriggers.push({ name: t.dataset.target, el: t });
        });

        // Position context menu to right of card
        var vw = window.innerWidth;
        var menuWidth = 180;
        var ctxLeft = cardRect.right + 8;
        if (ctxLeft + menuWidth > vw) ctxLeft = cardRect.left - menuWidth - 8;
        this.ctxLeft = Math.max(0, ctxLeft);
        this.ctxTop = Math.max(0, cardRect.top);
        this.ctxOpen = true;

        var qeSelf = this;
        this.$nextTick(function () {
          var ta = document.querySelector('.qe-title');
          if (ta) { ta.focus(); ta.selectionStart = ta.value.length; }
        });
      },

      hide: function () {
        this.open = false;
        this.ctxOpen = false;
        this._cardEl = null;
      },

      save: function () {
        htmx.ajax('POST', '/board/' + encodeURIComponent(this.slug) + '/cards/edit', {
          values: {
            col_idx: this.colIdx,
            card_idx: this.cardIdx,
            title: this.title.trim(),
            body: this.body.trim(),
            tags: this.tags.join(', '),
            priority: this.priority,
            name: this.slug,
            version: window.LB.getBoardVersion()
          },
          target: '#board-content',
          swap: 'innerHTML'
        });
        this.hide();
      },


      handleTitleKeydown: function (e) {
        var trigger = window.__lbNewLineTrigger ? window.__lbNewLineTrigger() : 'shift-enter';
        if (trigger === 'shift-enter') {
          if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); this.save(); }
        } else {
          if (e.key === 'Enter' && e.shiftKey) { e.preventDefault(); this.save(); }
        }
        if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) { e.preventDefault(); this.save(); return; }
        if (e.key === 'Escape') this.hide();
      },

      handleBodyKeydown: function (e) {
        this.handleTitleKeydown(e);
      },

      ctxComplete: function () {
        if (!this._cardEl) return;
        var btn = this._cardEl.querySelector('[hx-post$="/cards/complete"]');
        this.hide();
        if (btn) btn.click();
      },

      ctxMoveTo: function (trigger) {
        this.hide();
        trigger.el.click();
      },

      ctxDelete: function () {
        if (!this.ctxDeleteArmed) {
          this.ctxDeleteLabel = 'Deleting\u2026';
          var self = this;
          setTimeout(function () {
            self.ctxDeleteArmed = true;
            self.ctxDeleteLabel = 'Confirm delete';
          }, 1000);
          return;
        }
        if (!this._cardEl) return;
        var btn = this._cardEl.querySelector('[hx-post$="/cards/delete"]');
        this.hide();
        if (btn) btn.click();
      }
    };
  });

  // ── Column Menu ──────────────────────────────────────────────────────
  Alpine.data('columnMenu', function () {
    return {
      open: false,
      left: 0,
      top: 0,
      columnName: '',
      colIdx: -1,
      slug: '',
      _btn: null,
      // Inline rename
      renaming: false,
      renameValue: '',
      // Assistant modal
      assistantOpen: false,
      assistantMode: 'summary',
      assistantPrompt: '',

      show: function (btn) {
        if (this.open && this._btn === btn) { this.hide(); return; }
        this.hide();
        this._btn = btn;
        this.columnName = btn.dataset.columnName;
        this.slug = decodeURIComponent(window.location.pathname.replace(/^\/board\//, ''));

        var colEl = btn.closest('.column');
        this.colIdx = colEl ? Array.from(colEl.parentNode.children).filter(function (c) { return c.classList.contains('column') && !c.classList.contains('add-column'); }).indexOf(colEl) : -1;

        var rect = btn.getBoundingClientRect();
        var vw = window.innerWidth;
        var vh = window.innerHeight;
        this.left = rect.left;
        this.top = rect.bottom + 4;

        this.open = true;

        // Adjust after render
        var self = this;
        this.$nextTick(function () {
          var menu = document.getElementById('alpine-column-menu');
          if (!menu) return;
          var mr = menu.getBoundingClientRect();
          if (self.left + mr.width > vw) self.left = rect.right - mr.width;
          if (self.top + mr.height > vh) self.top = rect.top - mr.height - 4;
        });
      },

      hide: function () {
        this.open = false;
        this._btn = null;
      },

      editColumn: function () {
        this.hide();
        var col = document.querySelector('.column-menu-btn[data-column-name="' + this.columnName + '"]');
        if (!col) return;
        var column = col.closest('.column');
        if (!column) return;
        var h3 = column.querySelector('.column-header h3');
        if (!h3) return;

        var input = document.createElement('input');
        input.type = 'text';
        input.value = this.columnName;
        input.style.cssText = 'flex:1;min-width:0;padding:1px 4px;font-size:var(--font-size-xs);font-weight:700;text-transform:uppercase;letter-spacing:0.06em;color:var(--color-text-secondary);background:var(--color-surface);border:1px solid var(--color-accent);border-radius:var(--radius-sm);font-family:var(--font-sans)';

        var currentName = this.columnName;
        var slug = this.slug;
        h3.replaceWith(input);
        input.focus();
        input.select();

        var saved = false;
        function finish(save) {
          if (saved) return;
          saved = true;
          var newName = input.value.trim();
          input.replaceWith(h3);
          if (save && newName && newName !== currentName) {
            htmx.ajax('POST', '/board/' + encodeURIComponent(slug) + '/columns/rename', {
              values: { old_name: currentName, new_name: newName, name: slug, version: window.LB.getBoardVersion() },
              target: '#board-content', swap: 'innerHTML'
            });
          }
        }
        input.addEventListener('blur', function () { finish(true); });
        input.addEventListener('keydown', function (e) {
          if (e.key === 'Enter') { e.preventDefault(); finish(true); }
          if (e.key === 'Escape') { finish(false); }
        });
      },

      deleteColumn: function () {
        this.hide();
        if (window.confirm('Delete column "' + this.columnName + '" and all its cards?')) {
          htmx.ajax('POST', '/board/' + encodeURIComponent(this.slug) + '/columns/delete', {
            values: { column_name: this.columnName, name: this.slug, version: window.LB.getBoardVersion() },
            target: '#board-content', swap: 'innerHTML'
          });
        }
      },

      sortBy: function (field) {
        this.hide();
        htmx.ajax('POST', '/board/' + encodeURIComponent(this.slug) + '/columns/sort', {
          values: { col_idx: String(this.colIdx), sort_by: field, name: this.slug, version: window.LB.getBoardVersion() },
          target: '#board-content', swap: 'innerHTML'
        });
      },

      showAssistant: function (mode) {
        this.hide();
        this.assistantMode = mode;
        this.assistantPrompt = '';
        this.assistantOpen = true;
        var self = this;
        this.$nextTick(function () {
          var ta = document.querySelector('.assistant-textarea');
          if (ta) ta.focus();
        });
      },

      closeAssistant: function () {
        this.assistantOpen = false;
      },

      runAssistant: function () {
        // TODO: wire to AI backend
        this.assistantOpen = false;
      }
    };
  });

  // ── Board Settings Panel ─────────────────────────────────────────────
  Alpine.data('boardSettings', function () {
    return {
      open: false,
      boardName: '',
      boardDescription: '',
      tags: [],
      tagSuggestions: [],
      showCheckbox: '',
      cardPosition: '',
      expandColumns: 'false',
      viewMode: 'board',
      cardDisplayMode: '',
      slug: '',
      savedVisible: false,
      _savedTimer: null,

      toggle: function () {
        if (this.open) { this.open = false; return; }
        this.populate();
        this.open = true;
      },

      close: function () { this.open = false; },

      populate: function () {
        var bv = document.querySelector('.board-view');
        if (!bv) return;
        this.slug = bv.dataset.boardSlug;
        this.boardName = bv.dataset.boardName || '';
        this.boardDescription = bv.dataset.boardDescription || '';

        // Tags
        this.tags = [];
        var tagsRaw = bv.dataset.boardTags || '';
        if (tagsRaw) {
          var self = this;
          tagsRaw.split(',').forEach(function (s) { s = s.trim(); if (s && self.tags.indexOf(s) === -1) self.tags.push(s); });
        }

        // Suggestions from card tags
        this.tagSuggestions = [];
        var self = this;
        document.querySelectorAll('.card[data-card-tags]').forEach(function (c) {
          (c.dataset.cardTags || '').split(',').forEach(function (s) { s = s.trim(); if (s && self.tagSuggestions.indexOf(s) === -1) self.tagSuggestions.push(s); });
        });
        this.tags.forEach(function (t) { if (self.tagSuggestions.indexOf(t) === -1) self.tagSuggestions.push(t); });
        this.tagSuggestions.sort(function (a, b) { return a.toLowerCase().localeCompare(b.toLowerCase()); });

        // Display settings
        this.showCheckbox = bv.dataset.bsShowCheckbox || '';
        this.cardPosition = bv.dataset.bsCardPosition || '';
        this.expandColumns = bv.dataset.bsExpandColumns || 'false';
        this.viewMode = bv.dataset.bsViewMode || bv.dataset.viewMode || 'board';
        this.cardDisplayMode = bv.dataset.bsCardDisplayMode || '';
      },

      saveMeta: function () {
        htmx.ajax('POST', '/board/' + encodeURIComponent(this.slug) + '/meta', {
          values: {
            board_name: this.boardName.trim(),
            description: this.boardDescription.trim(),
            tags: this.tags.join(', '),
            name: this.slug,
            version: window.LB.getBoardVersion()
          },
          target: '#board-content',
          swap: 'innerHTML'
        });
        this.showSaved();
      },


      showSaved: function () {
        var self = this;
        this.savedVisible = true;
        clearTimeout(this._savedTimer);
        this._savedTimer = setTimeout(function () { self.savedVisible = false; }, 2000);
      },

      sendSettings: function () {
        var params = { name: this.slug, version: window.LB.getBoardVersion() };
        if (this.showCheckbox !== '') params.show_checkbox = this.showCheckbox;
        if (this.cardPosition !== '') params.card_position = this.cardPosition;
        params.expand_columns = this.expandColumns;
        params.view_mode = this.viewMode;
        if (this.cardDisplayMode !== '') params.card_display_mode = this.cardDisplayMode;
        htmx.ajax('POST', '/board/' + encodeURIComponent(this.slug) + '/settings', {
          values: params,
          target: '#board-content',
          swap: 'innerHTML'
        });
      },

      resetSetting: function (setting) {
        if (setting === 'show_checkbox') this.showCheckbox = '';
        if (setting === 'card_position') this.cardPosition = '';
        if (setting === 'card_display_mode') this.cardDisplayMode = '';
        this.sendSettings();
      }
    };
  });

  // ── Emoji Picker ─────────────────────────────────────────────────────
  Alpine.data('emojiPicker', function () {
    return {
      open: false,
      top: 0,
      left: 0,
      slug: '',
      emojis: ['📋','📌','📝','📊','📈','🎯','🚀','💡','🔥','⭐','❤️','💼','🏠','🎨','🎵','📚','🔧','⚡','🌟','🎮','🧪','📦','🔔','💬','🌈','🍀','🦊','🐱','🐶','🌻','🌙','☀️','🏔️','🌊','🎪','🏆','💎','🔑','🎁','🧩'],

      show: function (trigger, slug) {
        if (this.open && this.slug === slug) { this.open = false; return; }
        var rect = trigger.getBoundingClientRect();
        this.top = rect.top;
        this.left = rect.right + 8;
        this.slug = slug;
        this.open = true;
      },

      pick: function (emoji) {
        var isOnBoard = window.location.pathname.indexOf('/board/') === 0;
        var url = isOnBoard ? '/board/' + this.slug + '/icon' : '/boards/' + this.slug + '/icon';
        htmx.ajax('POST', url, { values: { name: this.slug, icon: emoji }, target: '#board-content', swap: 'innerHTML' });
        this.open = false;
      },

      clear: function () {
        this.pick('');
      }
    };
  });

  // ── Global Settings Page ─────────────────────────────────────────────
  Alpine.data('globalSettings', function () {
    return {
      siteName: 'LiveBoard',
      theme: 'system',
      colorTheme: 'default',
      fontFamily: 'system',
      columnWidth: 280,
      sidebarPosition: 'left',
      showCheckbox: 'true',
      newLineTrigger: 'shift-enter',
      cardPosition: 'append',
      cardDisplayMode: 'full',
      defaultColumns: [],
      savedVisible: false,
      _savedTimer: null,

      fontMap: {
        'system': { css: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif", gf: '' },
        'inter': { css: "'Inter', sans-serif", gf: 'Inter' },
        'ibm-plex-sans': { css: "'IBM Plex Sans', sans-serif", gf: 'IBM+Plex+Sans' },
        'source-sans-3': { css: "'Source Sans 3', sans-serif", gf: 'Source+Sans+3' },
        'nunito-sans': { css: "'Nunito Sans', sans-serif", gf: 'Nunito+Sans' },
        'dm-sans': { css: "'DM Sans', sans-serif", gf: 'DM+Sans' },
        'rubik': { css: "'Rubik', sans-serif", gf: 'Rubik' }
      },

      init: function () {
        var self = this;
        fetch('/api/settings')
          .then(function (r) { return r.json(); })
          .then(function (s) {
            self.siteName = s.site_name || 'LiveBoard';
            self.theme = s.theme || 'system';
            self.colorTheme = s.color_theme || 'default';
            self.fontFamily = s.font_family || 'system';
            self.columnWidth = s.column_width || 280;
            self.sidebarPosition = s.sidebar_position || 'left';
            self.showCheckbox = s.show_checkbox === false ? 'false' : 'true';
            self.newLineTrigger = s.newline_trigger || 'shift-enter';
            self.cardPosition = s.card_position || 'append';
            self.cardDisplayMode = s.card_display_mode || 'full';
            self.defaultColumns = s.default_columns || ['not now', 'maybe?', 'done'];
            // Update child columnChips component after async fetch
            self.$nextTick(function () {
              var colsComp = self.getColumnsComponent();
              if (colsComp) colsComp.cols = self.defaultColumns.slice();
            });
          });
      },

      applyFont: function (key) {
        var f = this.fontMap[key] || this.fontMap['system'];
        document.documentElement.style.setProperty('--font-sans', f.css);
        var existing = document.getElementById('lb-google-font');
        if (existing) existing.remove();
        if (f.gf) {
          var link = document.createElement('link');
          link.id = 'lb-google-font';
          link.rel = 'stylesheet';
          link.href = 'https://fonts.googleapis.com/css2?family=' + f.gf + ':wght@400;500;600;700&display=swap';
          document.head.appendChild(link);
        }
      },

      save: function () {
        var colsComp = this.getColumnsComponent();
        var rawCols = colsComp ? colsComp.cols : this.defaultColumns;
        if (rawCols.length === 0) rawCols = ['not now', 'maybe?', 'done'];

        var payload = {
          site_name: this.siteName.trim() || 'LiveBoard',
          theme: this.theme,
          color_theme: this.colorTheme,
          font_family: this.fontFamily,
          column_width: parseInt(this.columnWidth, 10) || 280,
          sidebar_position: this.sidebarPosition,
          show_checkbox: this.showCheckbox === 'true',
          newline_trigger: this.newLineTrigger,
          card_position: this.cardPosition,
          card_display_mode: this.cardDisplayMode,
          default_columns: rawCols
        };

        var self = this;
        fetch('/api/settings', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload)
        })
          .then(function (r) { return r.json(); })
          .then(function (s) {
            localStorage.setItem('lb_site_name', s.site_name);
            localStorage.setItem('lb_theme', s.theme);
            localStorage.setItem('lb_color_theme', s.color_theme);
            localStorage.setItem('lb_column_width', String(s.column_width));
            localStorage.setItem('lb_sidebar_position', s.sidebar_position);
            localStorage.setItem('lb_font_family', s.font_family);
            localStorage.setItem('lb_newline_trigger', s.newline_trigger);

            if (s.theme === 'system') { document.documentElement.removeAttribute('data-theme'); }
            else { document.documentElement.setAttribute('data-theme', s.theme); }
            if (s.color_theme && s.color_theme !== 'default') { document.documentElement.setAttribute('data-color-theme', s.color_theme); }
            else { document.documentElement.removeAttribute('data-color-theme'); }
            self.applyFont(s.font_family || 'system');
            document.documentElement.style.setProperty('--column-width', s.column_width + 'px');
            if (s.sidebar_position === 'right') { document.documentElement.setAttribute('data-sidebar-position', 'right'); }
            else { document.documentElement.removeAttribute('data-sidebar-position'); }

            var brandEl = document.querySelector('.brand-name');
            if (brandEl) brandEl.textContent = s.site_name;
            document.title = 'Settings \u2014 ' + s.site_name;

            self.savedVisible = true;
            clearTimeout(self._savedTimer);
            self._savedTimer = setTimeout(function () { self.savedVisible = false; }, 2000);
          });
      },

      getColumnsComponent: function () {
        var el = document.querySelector('[data-settings-cols]');
        return el ? Alpine.$data(el) : null;
      }
    };
  });

});
