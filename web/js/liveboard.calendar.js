// LiveBoard: Calendar View Alpine component.
document.addEventListener('alpine:init', function () {
  Alpine.data('calendarView', function () {
    var now = new Date();
    var boardEl = document.querySelector('.board-view');
    var wsAttr = boardEl ? boardEl.getAttribute('data-week-start') : 'sunday';
    var ws = (wsAttr === 'monday') ? 1 : 0;

    return {
      subView: 'month',
      viewYear: now.getFullYear(),
      viewMonth: now.getMonth(),
      viewDay: now.getDate(),
      weekStart: ws,
      _cardsByDate: {},
      _unscheduledCards: [],

      monthNames: ['January','February','March','April','May','June',
        'July','August','September','October','November','December'],
      monthNamesShort: ['Jan','Feb','Mar','Apr','May','Jun',
        'Jul','Aug','Sep','Oct','Nov','Dec'],
      dayNames: ['Sunday','Monday','Tuesday','Wednesday','Thursday','Friday','Saturday'],
      weekdayShort: ['Su','Mo','Tu','We','Th','Fr','Sa'],

      get weekdayLabels() {
        var labels = [];
        for (var i = 0; i < 7; i++) {
          labels.push(this.weekdayShort[(this.weekStart + i) % 7]);
        }
        return labels;
      },

      pad: function (n) { return n < 10 ? '0' + n : '' + n; },

      dateStr: function (y, m, d) {
        return y + '-' + this.pad(m + 1) + '-' + this.pad(d);
      },

      get todayStr() {
        var t = new Date();
        return t.getFullYear() + '-' + this.pad(t.getMonth() + 1) + '-' + this.pad(t.getDate());
      },

      get title() {
        if (this.subView === 'month') {
          return this.monthNames[this.viewMonth] + ' ' + this.viewYear;
        }
        if (this.subView === 'day') {
          var d = new Date(this.viewYear, this.viewMonth, this.viewDay);
          return this.dayNames[d.getDay()] + ', ' + this.monthNames[this.viewMonth] + ' ' + this.viewDay + ', ' + this.viewYear;
        }
        // week
        var start = this._weekStartDate();
        var end = new Date(start.getTime());
        end.setDate(end.getDate() + 6);
        var label = this.monthNamesShort[start.getMonth()] + ' ' + start.getDate();
        if (start.getMonth() !== end.getMonth() || start.getFullYear() !== end.getFullYear()) {
          label = label + ' – ' + this.monthNamesShort[end.getMonth()] + ' ' + end.getDate() + ', ' + end.getFullYear();
        } else {
          label = label + ' – ' + end.getDate() + ', ' + end.getFullYear();
        }
        return label;
      },

      _weekStartDate: function () {
        var d = new Date(this.viewYear, this.viewMonth, this.viewDay);
        var day = d.getDay();
        var diff = (day - this.weekStart + 7) % 7;
        d.setDate(d.getDate() - diff);
        return d;
      },

      prev: function () {
        if (this.subView === 'month') {
          this.viewMonth--;
          if (this.viewMonth < 0) { this.viewMonth = 11; this.viewYear--; }
        } else if (this.subView === 'week') {
          var d = new Date(this.viewYear, this.viewMonth, this.viewDay - 7);
          this.viewYear = d.getFullYear();
          this.viewMonth = d.getMonth();
          this.viewDay = d.getDate();
        } else {
          var d = new Date(this.viewYear, this.viewMonth, this.viewDay - 1);
          this.viewYear = d.getFullYear();
          this.viewMonth = d.getMonth();
          this.viewDay = d.getDate();
        }
      },

      next: function () {
        if (this.subView === 'month') {
          this.viewMonth++;
          if (this.viewMonth > 11) { this.viewMonth = 0; this.viewYear++; }
        } else if (this.subView === 'week') {
          var d = new Date(this.viewYear, this.viewMonth, this.viewDay + 7);
          this.viewYear = d.getFullYear();
          this.viewMonth = d.getMonth();
          this.viewDay = d.getDate();
        } else {
          var d = new Date(this.viewYear, this.viewMonth, this.viewDay + 1);
          this.viewYear = d.getFullYear();
          this.viewMonth = d.getMonth();
          this.viewDay = d.getDate();
        }
      },

      goToday: function () {
        var t = new Date();
        this.viewYear = t.getFullYear();
        this.viewMonth = t.getMonth();
        this.viewDay = t.getDate();
      },

      get monthDays() {
        var firstOfMonth = new Date(this.viewYear, this.viewMonth, 1);
        var firstDow = firstOfMonth.getDay();
        var leadingBlanks = (firstDow - this.weekStart + 7) % 7;
        var daysInMonth = new Date(this.viewYear, this.viewMonth + 1, 0).getDate();
        var prevMonthDays = new Date(this.viewYear, this.viewMonth, 0).getDate();
        var result = [];
        var today = this.todayStr;

        // leading days from previous month
        for (var i = leadingBlanks - 1; i >= 0; i--) {
          var pd = prevMonthDays - i;
          var pm = this.viewMonth - 1;
          var py = this.viewYear;
          if (pm < 0) { pm = 11; py--; }
          result.push({
            date: pd,
            dateStr: this.dateStr(py, pm, pd),
            isToday: false,
            isOtherMonth: true
          });
        }

        // current month
        for (var d = 1; d <= daysInMonth; d++) {
          var ds = this.dateStr(this.viewYear, this.viewMonth, d);
          result.push({
            date: d,
            dateStr: ds,
            isToday: ds === today,
            isOtherMonth: false
          });
        }

        // trailing days from next month
        var totalCells = result.length;
        var rows = Math.ceil(totalCells / 7);
        var needed = rows * 7 - totalCells;
        var nm = this.viewMonth + 1;
        var ny = this.viewYear;
        if (nm > 11) { nm = 0; ny++; }
        for (var t = 1; t <= needed; t++) {
          result.push({
            date: t,
            dateStr: this.dateStr(ny, nm, t),
            isToday: false,
            isOtherMonth: true
          });
        }

        return result;
      },

      get weekDays() {
        var start = this._weekStartDate();
        var result = [];
        var today = this.todayStr;
        for (var i = 0; i < 7; i++) {
          var d = new Date(start.getTime());
          d.setDate(d.getDate() + i);
          var ds = this.dateStr(d.getFullYear(), d.getMonth(), d.getDate());
          result.push({
            date: d.getDate(),
            dateStr: ds,
            isToday: ds === today,
            isOtherMonth: d.getMonth() !== this.viewMonth,
            monthLabel: this.monthNamesShort[d.getMonth()]
          });
        }
        return result;
      },

      get dayDate() {
        var ds = this.dateStr(this.viewYear, this.viewMonth, this.viewDay);
        return {
          dateStr: ds,
          isToday: ds === this.todayStr
        };
      },

      initCards: function () {
        var byDate = {};
        var unscheduled = [];
        var els = document.querySelectorAll('.calendar-card-data');
        for (var i = 0; i < els.length; i++) {
          var el = els[i];
          var card = {
            title: el.getAttribute('data-card-title') || '',
            due: el.getAttribute('data-card-due') || '',
            priority: el.getAttribute('data-card-priority') || '',
            assignee: el.getAttribute('data-card-assignee') || '',
            tags: el.getAttribute('data-card-tags') || '',
            colIdx: parseInt(el.getAttribute('data-col-idx') || '0', 10),
            cardIdx: parseInt(el.getAttribute('data-card-idx') || '0', 10),
            completed: el.getAttribute('data-card-completed') === 'true',
            columnName: el.getAttribute('data-card-column') || '',
            body: el.getAttribute('data-card-body') || ''
          };
          if (!card.due) {
            unscheduled.push(card);
          } else {
            if (!byDate[card.due]) byDate[card.due] = [];
            byDate[card.due].push(card);
          }
        }
        this._cardsByDate = byDate;
        this._unscheduledCards = unscheduled;
      },

      _matchesFilter: function (card) {
        var ui = Alpine.store('ui');
        if (ui && ui.hideCompleted && card.completed) return false;
        var q = ui ? (ui.searchQuery || '').toLowerCase().trim() : '';
        if (q) {
          var hay = [card.title, card.body, card.tags, card.assignee, card.columnName].join(' ').toLowerCase();
          if (hay.indexOf(q) === -1) return false;
        }
        return true;
      },

      cardsForDate: function (dateStr) {
        var self = this;
        return (this._cardsByDate[dateStr] || []).filter(function (c) { return self._matchesFilter(c); });
      },

      get unscheduledCards() {
        var self = this;
        return this._unscheduledCards.filter(function (c) { return self._matchesFilter(c); });
      },

      selectDay: function (dateStr) {
        var parts = dateStr.split('-');
        this.viewYear = parseInt(parts[0], 10);
        this.viewMonth = parseInt(parts[1], 10) - 1;
        this.viewDay = parseInt(parts[2], 10);
        this.subView = 'day';
      },

      handleDragStart: function (event, card) {
        event.dataTransfer.setData('text/plain', JSON.stringify(card));
        event.target.classList.add('dragging');
      },

      handleDrop: function (event, dateStr) {
        event.preventDefault();
        var raw = event.dataTransfer.getData('text/plain');
        if (!raw) return;
        try {
          var data = JSON.parse(raw);
        } catch (e) { return; }
        var boardEl = document.querySelector('.board-view');
        var slug = boardEl ? boardEl.dataset.boardSlug : '';
        if (!slug) return;
        var version = window.LB ? window.LB.getBoardVersion() : -1;
        htmx.ajax('POST', '/board/' + encodeURIComponent(slug) + '/cards/edit', {
          values: {
            col_idx: data.colIdx,
            card_idx: data.cardIdx,
            title: data.title,
            due: dateStr,
            tags: data.tags || '',
            priority: data.priority || '',
            assignee: data.assignee || '',
            body: data.body || '',
            version: version
          },
          target: '#board-content',
          swap: 'innerHTML'
        });
      },

      openCardModal: function (card) {
        // Find the clicked chip element and pass it to the card modal
        var el = document.querySelector('.calendar-card-chip[data-col-idx="' + card.colIdx + '"][data-card-idx="' + card.cardIdx + '"]');
        if (!el) return;
        var modalComp = document.querySelector('[x-data^="cardModal"]');
        if (modalComp && modalComp._x_dataStack) {
          Alpine.$data(modalComp).show(el);
        }
      }
    };
  });
});
