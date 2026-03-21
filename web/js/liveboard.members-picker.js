// LiveBoard: Members Picker Alpine component.
document.addEventListener('alpine:init', function () {
  Alpine.data('membersPicker', function (currentAssignee, boardMembers, onSelect) {
    return {
      members: (boardMembers || []).slice().sort(function (a, b) { return a.toLowerCase().localeCompare(b.toLowerCase()); }),
      current: currentAssignee || '',
      newMember: '',
      _onSelect: onSelect,

      addMember: function () {
        var name = this.newMember.trim();
        if (!name) return;
        if (this.members.indexOf(name) === -1) {
          this.members.push(name);
          this.members.sort(function (a, b) { return a.toLowerCase().localeCompare(b.toLowerCase()); });
        }
        this.newMember = '';
      },

      selectMember: function (m) {
        this.current = m;
        if (this._onSelect) this._onSelect(m);
      },

      clear: function () {
        this.current = '';
        if (this._onSelect) this._onSelect('');
      }
    };
  });
});
