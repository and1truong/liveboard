// LiveBoard: Global Settings Page Alpine component.
document.addEventListener('alpine:init', function () {
  Alpine.data('globalSettings', function () {
    return {
      siteName: 'LiveBoard',
      theme: 'system',
      colorTheme: 'default',
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
            self.colorTheme = s.color_theme || 'aqua';
            self.fontFamily = s.font_family || 'system';
            self.columnWidth = s.column_width || 280;
            self.sidebarPosition = s.sidebar_position || 'left';
            self.showCheckbox = s.show_checkbox !== false;
            self.newLineTrigger = s.newline_trigger || 'shift-enter';
            self.cardPosition = s.card_position || 'append';
            self.cardDisplayMode = s.card_display_mode || 'full';
            self.keyboardShortcuts = !!s.keyboard_shortcuts;
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
          show_checkbox: !!this.showCheckbox,
          newline_trigger: this.newLineTrigger,
          card_position: this.cardPosition,
          card_display_mode: this.cardDisplayMode,
          keyboard_shortcuts: !!this.keyboardShortcuts,
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
            var el = document.documentElement;

            // Update server-rendered data attributes so they stay current
            el.setAttribute('data-settings-theme', s.theme);
            el.setAttribute('data-settings-color-theme', s.color_theme);
            el.setAttribute('data-settings-column-width', String(s.column_width));
            el.setAttribute('data-settings-sidebar-position', s.sidebar_position);
            el.setAttribute('data-settings-font-family', s.font_family);
            el.setAttribute('data-settings-keyboard-shortcuts', s.keyboard_shortcuts ? '1' : '0');

            // Apply visual changes immediately
            if (s.theme === 'system') { el.removeAttribute('data-theme'); }
            else { el.setAttribute('data-theme', s.theme); }
            el.setAttribute('data-color-theme', s.color_theme || 'aqua');
            self.applyFont(s.font_family || 'system');
            el.style.setProperty('--column-width', s.column_width + 'px');
            if (s.sidebar_position === 'right') { el.setAttribute('data-sidebar-position', 'right'); }
            else { el.removeAttribute('data-sidebar-position'); }

            var brandEl = document.querySelector('.brand-name');
            if (brandEl) brandEl.textContent = s.site_name;
            document.title = 'Settings \u2014 ' + s.site_name;

            // auto-save: no flash message needed
          });
      },

      getColumnsComponent: function () {
        var el = document.querySelector('[data-settings-cols]');
        return el ? Alpine.$data(el) : null;
      }
    };
  });
});
