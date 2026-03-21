// LiveBoard: Command Palette Alpine component.
document.addEventListener('alpine:init', function () {
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
});
