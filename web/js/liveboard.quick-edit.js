// LiveBoard: Quick Edit + Context Menu Alpine component.
document.addEventListener('alpine:init', function () {
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

        this.slug = Alpine.store('board').slug || decodeURIComponent(window.location.pathname.replace(/^\/board\//, ''));
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
        this.tagSuggestions = Alpine.store('board').tags.slice();

        // Position
        this.left = posRect.left;
        this.top = cardRect.top;
        this.width = posRect.width;
        this.minHeight = cardRect.height;
        Alpine.store('ui').openModal('quickEdit');
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
        Alpine.store('ui').closeModal('quickEdit');
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
});
