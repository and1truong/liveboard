// LiveBoard: Alpine.js global stores.
// Loaded before all other Alpine components so stores are available at init time.
document.addEventListener('alpine:init', function () {

  // ── Board Store ──────────────────────────────────────────────────────
  // Shared board-level data: slug, version, tags, members.
  // Eliminates duplicate DOM scanning across cardModal, quickEdit, boardSettings.
  Alpine.store('board', {
    slug: '',
    version: '0',
    tags: [],
    members: [],

    init: function () {
      this.refresh();
    },

    refresh: function () {
      var bv = document.querySelector('.board-view');
      if (!bv) return;

      this.slug = bv.dataset.boardSlug || '';

      // Version from hidden input
      var vEl = document.getElementById('board-version');
      this.version = vEl ? vEl.value : '0';

      // Members: from board-view data + all card assignees
      var membersRaw = bv.dataset.boardMembers || '';
      var members = membersRaw ? membersRaw.split(',').map(function (s) { return s.trim(); }).filter(Boolean) : [];
      document.querySelectorAll('[data-card-assignee]').forEach(function (c) {
        var a = c.dataset.cardAssignee;
        if (a && members.indexOf(a) === -1) members.push(a);
      });
      this.members = members;

      // Tags: from all cards
      var tags = [];
      document.querySelectorAll('.card[data-card-tags]').forEach(function (c) {
        (c.dataset.cardTags || '').split(',').forEach(function (s) {
          s = s.trim();
          if (s && tags.indexOf(s) === -1) tags.push(s);
        });
      });
      tags.sort(function (a, b) { return a.toLowerCase().localeCompare(b.toLowerCase()); });
      this.tags = tags;
    }
  });

  // ── UI Store ─────────────────────────────────────────────────────────
  // Cross-component UI coordination: active modal, drag state, sidebar.
  Alpine.store('ui', {
    activeModal: null,
    isDragging: false,
    sidebarCollapsed: localStorage.getItem('sidebarCollapsed') === 'true',
    activeTag: null,
    focusedColumn: '',
    hideCompleted: localStorage.getItem('lb_hideCompleted') === 'true',
    searchQuery: '',

    toggleHideCompleted: function () {
      this.hideCompleted = !this.hideCompleted;
      localStorage.setItem('lb_hideCompleted', this.hideCompleted);
      var bv = document.querySelector('.board-view');
      if (bv) bv.setAttribute('data-hide-completed', this.hideCompleted);
      if (window.LB && LB.applyFilters) LB.applyFilters();
    },

    openModal: function (name) {
      this.activeModal = name;
    },

    closeModal: function (name) {
      if (this.activeModal === name) this.activeModal = null;
    },

    isModalOpen: function () {
      return this.activeModal !== null;
    },

    toggleSidebar: function () {
      this.sidebarCollapsed = !this.sidebarCollapsed;
      localStorage.setItem('sidebarCollapsed', this.sidebarCollapsed);
    }
  });
});
