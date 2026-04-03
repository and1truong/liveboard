// LiveBoard: Reminder notifications via global SSE.
(function () {
  'use strict';

  var evtSource = null;

  function connect() {
    evtSource = new EventSource('/events/global');

    evtSource.addEventListener('reminder-fire', function (e) {
      var data;
      try { data = JSON.parse(e.data); } catch (_) { return; }
      showReminderToast(data);
      requestBrowserNotification(data);
    });

    evtSource.addEventListener('connected', function () {
      console.log('[reminder] global SSE connected');
    });

    evtSource.onerror = function () {
      evtSource.close();
      evtSource = null;
      setTimeout(connect, 10000);
    };
  }

  function showReminderToast(data) {
    var toast = document.createElement('div');
    toast.className = 'reminder-toast';
    toast.innerHTML =
      '<div class="reminder-toast-content">' +
        '<span class="reminder-toast-icon">&#128276;</span>' +
        '<div class="reminder-toast-body">' +
          '<strong>' + escapeHtml(data.card_title || data.board_slug) + '</strong>' +
          (data.message ? '<div class="reminder-toast-message">' + escapeHtml(data.message) + '</div>' : '') +
        '</div>' +
        '<button class="reminder-toast-close" onclick="this.parentElement.parentElement.remove()">&times;</button>' +
      '</div>' +
      '<div class="reminder-toast-actions">' +
        '<a href="/board/' + encodeURIComponent(data.board_slug) + (data.card_id ? '?highlight=' + data.card_id : '') + '" class="btn-text btn-sm">View</a>' +
        '<a href="/reminders" class="btn-text btn-sm">Reminders</a>' +
      '</div>';

    document.body.appendChild(toast);

    // Auto-remove after 15 seconds
    setTimeout(function () {
      if (toast.parentElement) toast.remove();
    }, 15000);
  }

  function requestBrowserNotification(data) {
    if (!('Notification' in window)) return;
    if (Notification.permission === 'granted') {
      sendBrowserNotification(data);
    } else if (Notification.permission !== 'denied') {
      Notification.requestPermission().then(function (perm) {
        if (perm === 'granted') sendBrowserNotification(data);
      });
    }
  }

  function sendBrowserNotification(data) {
    var title = data.type === 'board' ? 'Board Reminder' : 'Reminder';
    var body = data.card_title || data.board_slug;
    if (data.message) body += '\n' + data.message;

    var n = new Notification(title, { body: body, icon: '/static/img/liveboard-icon.svg' });
    n.onclick = function () {
      window.focus();
      var url = '/board/' + encodeURIComponent(data.board_slug);
      if (data.card_id) url += '?highlight=' + data.card_id;
      window.location.href = url;
    };
  }

  function escapeHtml(s) {
    var d = document.createElement('div');
    d.textContent = s;
    return d.innerHTML;
  }

  // Deep-link highlight: scroll to and flash a card when ?highlight=cardId is present
  function checkHighlight() {
    var params = new URLSearchParams(window.location.search);
    var cardId = params.get('highlight');
    if (!cardId) return;

    var card = document.querySelector('[data-card-id="' + cardId + '"]');
    if (card) {
      card.scrollIntoView({ behavior: 'smooth', block: 'center' });
      card.classList.add('card-highlight-flash');
      setTimeout(function () { card.classList.remove('card-highlight-flash'); }, 3000);
    }

    // Clean up URL
    params.delete('highlight');
    var newUrl = window.location.pathname + (params.toString() ? '?' + params.toString() : '');
    window.history.replaceState({}, '', newUrl);
  }

  // Initialize
  document.addEventListener('DOMContentLoaded', function () {
    connect();
    checkHighlight();
  });

  // Re-check highlight after HTMX content swaps
  document.addEventListener('htmx:afterSettle', function () {
    checkHighlight();
  });
})();
