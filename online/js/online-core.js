// LiveBoard Online: core namespace and helpers (replaces liveboard.core.js).
// No HTMX, no SSE — just the LB namespace for compatibility with shared components.
window.LB = window.LB || {};

(function () {
  var LB = window.LB;
  LB.isDragging = false;

  LB.getBoardVersion = function () { return '0'; };

  LB.colorLuminance = function (hex) {
    if (!hex || hex.length < 7) return 0.5;
    var r = parseInt(hex.slice(1, 3), 16) / 255;
    var g = parseInt(hex.slice(3, 5), 16) / 255;
    var b = parseInt(hex.slice(5, 7), 16) / 255;
    var toLinear = function (c) { return c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4); };
    return 0.2126 * toLinear(r) + 0.7152 * toLinear(g) + 0.0722 * toLinear(b);
  };

  LB.applyTagColors = function () {};

  // New line trigger helper
  window.__lbNewLineTrigger = function () {
    try {
      var s = JSON.parse(localStorage.getItem('liveboard_settings') || '{}');
      return s.newline_trigger || 'shift-enter';
    } catch (e) { return 'shift-enter'; }
  };
})();
