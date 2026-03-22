// LiveBoard: Command Palette Alpine component.
document.addEventListener('alpine:init', function () {
  Alpine.data('cmdPalette', function (boards, activeSlug) {
    return {
      open: false,
      query: '',
      activeIdx: 0,

      escapeHtml: function (str) {
        var d = document.createElement('div');
        d.appendChild(document.createTextNode(str));
        return d.innerHTML;
      },

      fuzzyMatch: function (query, text) {
        var qi = 0, indices = [];
        var qLower = query.toLowerCase(), tLower = text.toLowerCase();
        for (var ti = 0; ti < tLower.length && qi < qLower.length; ti++) {
          if (tLower[ti] === qLower[qi]) {
            indices.push(ti);
            qi++;
          }
        }
        return qi === qLower.length ? indices : null;
      },

      highlightName: function (name, indices) {
        if (!indices || !indices.length) return this.escapeHtml(name);
        var set = {};
        indices.forEach(function (i) { set[i] = true; });
        var out = '';
        for (var i = 0; i < name.length; i++) {
          var ch = this.escapeHtml(name[i]);
          out += set[i] ? '<mark class="cmd-palette-highlight">' + ch + '</mark>' : ch;
        }
        return out;
      },

      get items() {
        var self = this;
        var path = window.location.pathname;
        var boardItems = [];
        var navItems = [];
        var counter = 0;

        boards.forEach(function (b) {
          var url = '/board/' + b.slug;
          if (b.slug === activeSlug && path === url) return;
          var item = { icon: b.icon || '\u2630', name: b.name, url: url, category: 'Boards', matchIndices: null, _idx: counter++ };
          boardItems.push(item);
        });

        if (path !== '/') {
          navItems.push({ icon: '\uD83C\uDFE0', name: 'All Boards', url: '/', category: 'Navigation', matchIndices: null, _idx: counter++ });
        }
        if (path !== '/settings') {
          navItems.push({ icon: '\u2699\uFE0F', name: 'Settings', url: '/settings', category: 'Navigation', matchIndices: null, _idx: counter++ });
        }

        var all = boardItems.concat(navItems);

        if (self.query) {
          var q = self.query;
          var filtered = [];
          var reIdx = 0;
          all.forEach(function (it) {
            var indices = self.fuzzyMatch(q, it.name);
            if (indices) {
              it.matchIndices = indices;
              it._idx = reIdx++;
              filtered.push(it);
            }
          });
          return filtered;
        }
        return all;
      },

      get groupedItems() {
        var items = this.items;
        var groups = [];
        var boardGroup = { label: 'Boards', items: [] };
        var navGroup = { label: 'Navigation', items: [] };
        items.forEach(function (it) {
          if (it.category === 'Navigation') navGroup.items.push(it);
          else boardGroup.items.push(it);
        });
        if (boardGroup.items.length) groups.push(boardGroup);
        if (navGroup.items.length) groups.push(navGroup);
        return groups;
      },

      get selectableItems() {
        return this.items;
      },

      toggle: function () {
        this.open = !this.open;
        if (this.open) { Alpine.store('ui').openModal('cmdPalette'); }
        else { Alpine.store('ui').closeModal('cmdPalette'); }
        this.query = '';
        this.activeIdx = 0;
        if (this.open) {
          var self = this;
          this.$nextTick(function () {
            requestAnimationFrame(function () {
              var inp = self.$refs.input;
              if (inp) inp.focus();
            });
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
          Alpine.store('ui').closeModal('cmdPalette');
          return;
        }
        var count = this.selectableItems.length;
        if (e.key === 'ArrowDown') {
          e.preventDefault();
          if (count > 0) {
            var wrapped = this.activeIdx === count - 1;
            this.activeIdx = (this.activeIdx + 1) % count;
            this.$nextTick(function () {
              if (wrapped) {
                var list = document.querySelector('.cmd-palette-list');
                if (list) list.scrollTop = 0;
              }
              var el = document.querySelector('[data-cmd-active="true"]');
              if (el) el.scrollIntoView({ block: 'nearest' });
            });
          }
          return;
        }
        if (e.key === 'ArrowUp') {
          e.preventDefault();
          if (count > 0) {
            var wrapped = this.activeIdx === 0;
            this.activeIdx = (this.activeIdx - 1 + count) % count;
            this.$nextTick(function () {
              if (wrapped) {
                var list = document.querySelector('.cmd-palette-list');
                if (list) list.scrollTop = list.scrollHeight;
              }
              var el = document.querySelector('[data-cmd-active="true"]');
              if (el) el.scrollIntoView({ block: 'nearest' });
            });
          }
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
});
