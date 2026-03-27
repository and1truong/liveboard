// LiveBoard Online: hash-based client-side router.
// Routes: #/ (board list), #/board/<slug>, #/settings
(function () {
  window.LBRouter = {
    current: { view: 'home', slug: '' },

    parse: function () {
      var hash = window.location.hash || '#/';
      if (hash.indexOf('#/board/') === 0) {
        return { view: 'board', slug: decodeURIComponent(hash.slice(8)) };
      }
      if (hash === '#/settings') {
        return { view: 'settings', slug: '' };
      }
      return { view: 'home', slug: '' };
    },

    navigate: function (hash) {
      window.location.hash = hash;
    },

    init: function () {
      var self = this;
      window.addEventListener('hashchange', function () {
        self.current = self.parse();
        self._notify();
      });
      this.current = this.parse();
    },

    _listeners: [],
    onChange: function (fn) { this._listeners.push(fn); },
    _notify: function () {
      var c = this.current;
      this._listeners.forEach(function (fn) { fn(c); });
    }
  };
})();
