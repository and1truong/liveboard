// LiveBoard: Tag Chips Alpine component.
// Usage: x-data="tagChips()"
// Parent scope MUST provide: `tags` (array) and `tagSuggestions` (array)
// tagChips does NOT own `tags` — it inherits from the parent via Alpine scope chain.
document.addEventListener('alpine:init', function () {
  Alpine.data('tagChips', function () {
    return {
      // `tags` and `tagSuggestions` inherited from parent x-data scope
      tcInput: '',
      tcDropdownOpen: false,
      tcActiveIdx: -1,

      get filtered() {
        var self = this;
        var f = this.tcInput.toLowerCase();
        var suggestions = this.tagSuggestions || [];
        return suggestions.filter(function (t) {
          return self.tags.indexOf(t) === -1 && (!f || t.toLowerCase().indexOf(f) !== -1);
        });
      },

      addTag: function (tag) {
        tag = tag.trim();
        if (!tag || this.tags.indexOf(tag) !== -1) return;
        this.tags.push(tag);
        this.tcInput = '';
        this.tcDropdownOpen = true;
        this.tcActiveIdx = -1;
        this.$dispatch('tags-changed');
      },

      removeTag: function (idx) {
        this.tags.splice(idx, 1);
        this.$dispatch('tags-changed');
      },

      showDropdown: function () {
        this.tcDropdownOpen = true;
        this.tcActiveIdx = -1;
      },

      hideDropdown: function () {
        var self = this;
        setTimeout(function () { self.tcDropdownOpen = false; self.tcActiveIdx = -1; }, 150);
      },

      handleKeydown: function (e) {
        if (e.key === 'ArrowDown') {
          e.preventDefault();
          this.tcActiveIdx = Math.min(this.tcActiveIdx + 1, this.filtered.length - 1);
        } else if (e.key === 'ArrowUp') {
          e.preventDefault();
          this.tcActiveIdx = Math.max(this.tcActiveIdx - 1, 0);
        } else if (e.key === 'Enter') {
          e.preventDefault();
          if (this.tcActiveIdx >= 0 && this.filtered[this.tcActiveIdx]) {
            this.addTag(this.filtered[this.tcActiveIdx]);
          } else if (this.tcInput.trim()) {
            this.addTag(this.tcInput);
          }
        } else if (e.key === 'Backspace' && !this.tcInput && this.tags.length) {
          this.tags.pop();
          this.$dispatch('tags-changed');
        } else if (e.key === 'Escape') {
          this.tcDropdownOpen = false;
          this.tcActiveIdx = -1;
        }
      },

      getTagsValue: function () {
        return this.tags.join(', ');
      }
    };
  });
});
