// LiveBoard: Card Modal Alpine component.
document.addEventListener('alpine:init', function () {
  Alpine.data('cardModal', function () {
    return {
      open: false,
      title: '',
      body: '',
      showBodyPreview: false,
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
        this.slug = Alpine.store('board').slug || decodeURIComponent(window.location.pathname.replace(/^\/board\//, ''));
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

        // Tags and members from board store
        this.tagSuggestions = Alpine.store('board').tags.slice();
        this.boardMembers = Alpine.store('board').members.slice();

        // Move triggers
        this.moveTriggers = [];
        Array.from(card.querySelectorAll('.move-trigger[data-target]')).forEach(function (t) {
          self.moveTriggers.push({ name: t.dataset.target, el: t });
        });

        this.hasCompleteBtn = !!card.querySelector('[hx-post$="/cards/complete"]');
        this.hasDeleteBtn = !!card.querySelector('[hx-post$="/cards/delete"]');

        this.showDatePicker = false;
        this.showMembersPicker = false;
        this.showBodyPreview = false;
        Alpine.store('ui').openModal('cardModal');
        this.open = true;

        this.$nextTick(function () {
          // Use rAF to ensure Safari has completed layout before measuring
          requestAnimationFrame(function () {
            var ta = document.querySelector('.card-modal-title');
            if (ta) {
              ta.style.height = 'auto';
              if (ta.scrollHeight > 0) {
                ta.style.height = ta.scrollHeight + 'px';
              }
              ta.focus();
            }
          });
        });
      },

      close: function () {
        this.open = false;
        this._cardEl = null;
        Alpine.store('ui').closeModal('cardModal');
      },

      autoResize: function (e) {
        var el = e.target;
        var minH = parseInt(el.style.minHeight) || 0;
        el.style.height = 'auto';
        el.style.height = Math.max(el.scrollHeight, minH) + 'px';
      },

      captureResize: function (e) {
        e.target.style.minHeight = e.target.offsetHeight + 'px';
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
          .replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" target="_blank" rel="noopener">$1</a>')
          .replace(/^[-*] (.+)$/gm, '<li>$1</li>')
          .replace(/(<li>.*<\/li>)/gs, '<ul>$1</ul>')
          .replace(/<\/ul>\s*<ul>/g, '');
        // Convert remaining double newlines to paragraphs
        var parts = s.split(/\n\n+/);
        return parts.map(function (p) {
          p = p.trim();
          if (!p) return '';
          if (/^<[hul]/.test(p)) return p;
          return '<p>' + p.replace(/\n/g, '<br>') + '</p>';
        }).join('\n');
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
});
