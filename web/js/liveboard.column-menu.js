// LiveBoard: Column Menu Alpine component.
document.addEventListener('alpine:init', function () {
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

        Alpine.store('ui').openModal('columnMenu');
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
        Alpine.store('ui').closeModal('columnMenu');
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

      focusColumn: function () {
        var name = this.columnName;
        this.hide();
        Alpine.store('ui').focusedColumn = name;
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
});
