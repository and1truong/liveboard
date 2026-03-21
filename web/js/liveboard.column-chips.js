// LiveBoard: Column Chips Alpine component (settings page, no dropdown).
document.addEventListener('alpine:init', function () {
  Alpine.data('columnChips', function (initial) {
    return {
      cols: initial || [],
      input: '',

      addCol: function (val) {
        val = (val || '').replace(/,/g, '').trim();
        if (!val) return;
        this.cols.push(val);
        this.input = '';
        this.$dispatch('cols-changed');
      },

      removeCol: function (idx) {
        this.cols.splice(idx, 1);
        this.$dispatch('cols-changed');
      },

      handleKeydown: function (e) {
        if (e.key === 'Enter' || e.key === ',') {
          e.preventDefault();
          this.addCol(this.input);
        } else if (e.key === 'Backspace' && !this.input && this.cols.length) {
          this.cols.pop();
          this.$dispatch('cols-changed');
        }
      }
    };
  });
});
