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
});
