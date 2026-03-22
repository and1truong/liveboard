// LiveBoard: Board Settings Panel Alpine component.
document.addEventListener('alpine:init', function () {
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
          bv.dataset.viewMode = this.viewMode || bv.dataset.viewMode || 'board';
          bv.dataset.showCheckbox = this.showCheckbox || bv.dataset.globalShowCheckbox || 'true';
          bv.dataset.cardPosition = this.cardPosition || bv.dataset.globalCardPosition || 'append';
          bv.dataset.expandColumns = this.expandColumns;
          bv.dataset.cardDisplayMode = this.cardDisplayMode || bv.dataset.globalCardDisplayMode || 'full';
        }

        var self = this;
        var version = window.LB.getBoardVersion();

        // Save meta
        htmx.ajax('POST', '/board/' + encodeURIComponent(this.slug) + '/meta', {
          values: {
            board_name: this.boardName.trim(),
            description: this.boardDescription.trim(),
            tags: this.tags.join(', '),
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

      resetSetting: function (setting) {
        if (setting === 'show_checkbox') this.showCheckbox = '';
        if (setting === 'card_position') this.cardPosition = '';
        if (setting === 'card_display_mode') this.cardDisplayMode = '';
      }
    };
  });
});
