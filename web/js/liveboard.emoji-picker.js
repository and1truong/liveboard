// LiveBoard: Emoji Picker Alpine component.
document.addEventListener('alpine:init', function () {
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
        var slug = this.slug;
        htmx.ajax('POST', url, { values: { name: slug, icon: emoji }, target: '#board-content', swap: 'innerHTML' }).then(function () {
          if (isOnBoard) {
            htmx.ajax('GET', '/api/boards/sidebar?slug=' + encodeURIComponent(slug), { target: '#sidebar-board-list', swap: 'innerHTML' });
          }
        });
        this.open = false;
      },

      clear: function () {
        this.pick('');
      }
    };
  });
});
