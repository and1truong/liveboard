// LiveBoard: Date Picker Alpine component.
document.addEventListener('alpine:init', function () {
  Alpine.data('datePicker', function (currentDue, onSelect) {
    var now = new Date();
    var vy = now.getFullYear();
    var vm = now.getMonth();
    if (currentDue) {
      var parts = currentDue.split('-');
      if (parts.length === 3) {
        vy = parseInt(parts[0], 10);
        vm = parseInt(parts[1], 10) - 1;
      }
    }
    return {
      viewYear: vy,
      viewMonth: vm,
      selected: currentDue || '',
      monthNames: ['January','February','March','April','May','June',
        'July','August','September','October','November','December'],
      weekdays: ['Su','Mo','Tu','We','Th','Fr','Sa'],
      _onSelect: onSelect,

      get monthLabel() { return this.monthNames[this.viewMonth] + ' ' + this.viewYear; },

      get days() {
        var firstDay = new Date(this.viewYear, this.viewMonth, 1).getDay();
        var daysInMonth = new Date(this.viewYear, this.viewMonth + 1, 0).getDate();
        var result = [];
        for (var i = 0; i < firstDay; i++) result.push(null);
        for (var d = 1; d <= daysInMonth; d++) result.push(d);
        return result;
      },

      pad: function (n) { return n < 10 ? '0' + n : '' + n; },

      dateStr: function (d) {
        return this.viewYear + '-' + this.pad(this.viewMonth + 1) + '-' + this.pad(d);
      },

      get todayStr() {
        var t = new Date();
        return t.getFullYear() + '-' + this.pad(t.getMonth() + 1) + '-' + this.pad(t.getDate());
      },

      prevMonth: function () {
        this.viewMonth--;
        if (this.viewMonth < 0) { this.viewMonth = 11; this.viewYear--; }
      },

      nextMonth: function () {
        this.viewMonth++;
        if (this.viewMonth > 11) { this.viewMonth = 0; this.viewYear++; }
      },

      selectDate: function (d) {
        var val = this.dateStr(d);
        this.selected = val;
        if (this._onSelect) this._onSelect(val);
      },

      removeDate: function () {
        this.selected = '';
        if (this._onSelect) this._onSelect('');
      }
    };
  });
});
