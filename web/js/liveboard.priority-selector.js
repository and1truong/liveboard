// LiveBoard: Priority Selector Alpine component.
document.addEventListener('alpine:init', function () {
  Alpine.data('prioritySelector', function (initial) {
    return {
      value: initial || '',
      options: [
        { val: '', label: '\u2014', title: 'None' },
        { val: 'low', label: 'L', title: 'Low' },
        { val: 'medium', label: 'M', title: 'Medium' },
        { val: 'high', label: 'H', title: 'High' },
        { val: 'critical', label: '!', title: 'Critical' }
      ],
      select: function (val) { this.value = val; }
    };
  });
});
