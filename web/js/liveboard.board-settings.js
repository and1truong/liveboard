// LiveBoard: Board Settings Panel Alpine component.
document.addEventListener('alpine:init', function () {
  Alpine.data('boardSettings', function () {
    return {
      open: false,
      boardName: '',
      boardDescription: '',
      tags: [],
      tagSuggestions: [],
      tagColors: {},
      TAG_PALETTE: ['#e05252','#d4722c','#c9a227','#4caf76','#45aab5','#4080c4','#8060c4','#c060a0','#607080','#a07040'],
      showCheckbox: '',
      cardPosition: '',
      expandColumns: 'false',
      viewMode: 'board',
      cardDisplayMode: '',
      weekStart: '',
      slug: '',

      toggle: function () {
        if (this.open) { this.open = false; Alpine.store('ui').closeModal('boardSettings'); return; }
        this.populate();
        Alpine.store('ui').openModal('boardSettings');
        this.open = true;
      },

      close: function () { this.open = false; Alpine.store('ui').closeModal('boardSettings'); },

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

        // Tag colors
        var tcRaw = bv.dataset.boardTagColors || '{}';
        try { this.tagColors = JSON.parse(tcRaw) || {}; } catch(e) { this.tagColors = {}; }

        // Suggestions from board store + board-level tags
        this.tagSuggestions = Alpine.store('board').tags.slice();
        var self = this;
        this.tags.forEach(function (t) { if (self.tagSuggestions.indexOf(t) === -1) self.tagSuggestions.push(t); });
        this.tagSuggestions.sort(function (a, b) { return a.toLowerCase().localeCompare(b.toLowerCase()); });

        // Display settings
        this.showCheckbox = bv.dataset.bsShowCheckbox || '';
        this.cardPosition = bv.dataset.bsCardPosition || '';
        this.expandColumns = bv.dataset.bsExpandColumns || 'false';
        this.viewMode = bv.dataset.bsViewMode || bv.dataset.viewMode || 'board';
        this.cardDisplayMode = bv.dataset.bsCardDisplayMode || '';
        this.weekStart = bv.dataset.bsWeekStart || bv.dataset.weekStart || 'sunday';
      },

      applySettings: function () {
        var bv = document.querySelector('.board-view');
        if (bv) {
          bv.dataset.boardName = this.boardName.trim();
          bv.dataset.boardDescription = this.boardDescription.trim();
          bv.dataset.boardTags = this.tags.join(',');
          bv.dataset.bsShowCheckbox = this.showCheckbox;
          bv.dataset.bsCardPosition = this.cardPosition;
          bv.dataset.bsExpandColumns = this.expandColumns;
          bv.dataset.bsViewMode = this.viewMode;
          bv.dataset.bsCardDisplayMode = this.cardDisplayMode;
          bv.dataset.bsWeekStart = this.weekStart;
          bv.dataset.weekStart = this.weekStart || bv.dataset.weekStart || 'sunday';
          bv.dataset.viewMode = this.viewMode || bv.dataset.viewMode || 'board';
          bv.dataset.showCheckbox = this.showCheckbox || bv.dataset.globalShowCheckbox || 'true';
          bv.dataset.cardPosition = this.cardPosition || bv.dataset.globalCardPosition || 'append';
          bv.dataset.expandColumns = this.expandColumns;
          bv.dataset.cardDisplayMode = this.cardDisplayMode || bv.dataset.globalCardDisplayMode || 'full';
        }

        // Update the data attribute immediately so applyTagColors can run
        if (bv) { bv.dataset.boardTagColors = JSON.stringify(this.tagColors); }
        if (window.LB && window.LB.applyTagColors) { window.LB.applyTagColors(); }

        var self = this;
        var version = window.LB.getBoardVersion();

        // Save meta
        htmx.ajax('POST', '/board/' + encodeURIComponent(this.slug) + '/meta', {
          values: {
            board_name: this.boardName.trim(),
            description: this.boardDescription.trim(),
            tags: this.tags.join(', '),
            tag_colors: JSON.stringify(this.tagColors),
            name: this.slug,
            version: version
          },
          target: '#board-content',
          swap: 'innerHTML'
        }).then(function () {
          // Save display settings with updated version
          var params = { name: self.slug, version: window.LB.getBoardVersion() };
          if (self.showCheckbox !== '') params.show_checkbox = self.showCheckbox;
          if (self.cardPosition !== '') params.card_position = self.cardPosition;
          params.expand_columns = self.expandColumns;
          params.view_mode = self.viewMode;
          if (self.cardDisplayMode !== '') params.card_display_mode = self.cardDisplayMode;
          if (self.weekStart !== '') params.week_start = self.weekStart;
          htmx.ajax('POST', '/board/' + encodeURIComponent(self.slug) + '/settings', {
            values: params,
            target: '#board-content',
            swap: 'innerHTML'
          });

          // Refresh sidebar navigation to reflect name/tag changes
          htmx.ajax('GET', '/api/boards/sidebar?slug=' + encodeURIComponent(self.slug), {
            target: '#sidebar-board-list',
            swap: 'innerHTML'
          });

          // Update mobile dropdown label
          var ddLabel = document.querySelector('.dropdown-trigger-label');
          if (ddLabel && self.boardName.trim()) ddLabel.textContent = self.boardName.trim();
        });
      },

      getTagPreviewStyle: function (tag) {
        var bg = this.tagColors[tag];
        if (!bg) return '';
        var lum = window.LB && window.LB.colorLuminance ? window.LB.colorLuminance(bg) : 0.5;
        return 'background:' + bg + ';color:' + (lum > 0.35 ? '#111' : '#fff') + ';border-color:transparent';
      },

      toggleTagColor: function (tag, color) {
        if (this.tagColors[tag] === color) {
          delete this.tagColors[tag];
        } else {
          this.tagColors[tag] = color;
        }
      },

      clearTagColor: function (tag) {
        delete this.tagColors[tag];
      },

      resetSetting: function (setting) {
        if (setting === 'show_checkbox') this.showCheckbox = '';
        if (setting === 'card_position') this.cardPosition = '';
        if (setting === 'card_display_mode') this.cardDisplayMode = '';
      }
    };
  });
});
