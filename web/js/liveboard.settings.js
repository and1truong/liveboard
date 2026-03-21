// LiveBoard: board settings panel controller.
(function () {
  var LB = window.LB;

  function initBoardSettingsPanel() {
    var boardView = document.querySelector(".board-view");
    if (!boardView) return;

    // Abort previous listeners to prevent duplicates after HTMX re-renders
    if (boardView._settingsAbort) boardView._settingsAbort.abort();
    var ac = new AbortController();
    boardView._settingsAbort = ac;
    var opts = { signal: ac.signal };

    var gearBtn = boardView.querySelector(".board-settings-btn");
    var backdrop = boardView.querySelector(".board-settings-backdrop");
    var panel = boardView.querySelector(".board-settings-panel");
    if (!gearBtn || !backdrop || !panel) return;

    var closeBtn = panel.querySelector(".board-settings-close");
    var slug = boardView.dataset.boardSlug;

    // Meta fields
    var bsBoardName = document.getElementById("bsBoardName");
    var bsBoardDescription = document.getElementById("bsBoardDescription");
    var bsSaveMeta = document.getElementById("bsSaveMeta");
    var bsMetaSavedMsg = document.getElementById("bsMetaSavedMsg");
    var bsTagsContainer = document.getElementById("bsBoardTagsContainer");

    // Tags state
    var currentTags = [];
    var tagsSavedTimer = null;

    function initTagsUI() {
      if (!bsTagsContainer) return;
      bsTagsContainer.innerHTML = "";

      var tagsInput = document.createElement("input");
      tagsInput.className = "card-modal-tags-input";
      tagsInput.type = "text";
      tagsInput.placeholder = currentTags.length ? "" : "Add tags...";

      var tagsDropdown = document.createElement("div");
      tagsDropdown.className = "card-modal-tags-dropdown";
      var dropdownActiveIdx = -1;

      // Collect suggestions from card tags
      var allSuggestions = [];
      document.querySelectorAll(".card[data-card-tags]").forEach(function (c) {
        (c.dataset.cardTags || "").split(",").forEach(function (s) {
          s = s.trim();
          if (s && allSuggestions.indexOf(s) === -1) allSuggestions.push(s);
        });
      });
      currentTags.forEach(function (t) {
        if (allSuggestions.indexOf(t) === -1) allSuggestions.push(t);
      });
      allSuggestions.sort(function (a, b) { return a.toLowerCase().localeCompare(b.toLowerCase()); });

      function renderChips() {
        Array.from(bsTagsContainer.querySelectorAll(".card-modal-tag-chip")).forEach(function (el) { el.remove(); });
        currentTags.forEach(function (tag, idx) {
          var chip = document.createElement("span");
          chip.className = "card-modal-tag-chip";
          chip.textContent = tag;
          var removeBtn = document.createElement("button");
          removeBtn.className = "card-modal-tag-chip-remove";
          removeBtn.type = "button";
          removeBtn.innerHTML = "&times;";
          removeBtn.addEventListener("click", function (e) {
            e.stopPropagation();
            currentTags.splice(idx, 1);
            renderChips();
            tagsInput.placeholder = currentTags.length ? "" : "Add tags...";
          });
          chip.appendChild(removeBtn);
          bsTagsContainer.insertBefore(chip, tagsInput);
        });
      }

      function addTag(tag) {
        tag = tag.trim();
        if (!tag || currentTags.indexOf(tag) !== -1) return;
        currentTags.push(tag);
        renderChips();
        tagsInput.value = "";
        tagsInput.placeholder = "";
        hideDropdown();
      }

      function showDropdown(filter) {
        tagsDropdown.innerHTML = "";
        dropdownActiveIdx = -1;
        var f = (filter || "").toLowerCase();
        var suggestions = allSuggestions.filter(function (t) {
          return currentTags.indexOf(t) === -1 && (!f || t.toLowerCase().indexOf(f) !== -1);
        });
        if (suggestions.length === 0) {
          if (f) {
            var hint = document.createElement("div");
            hint.className = "card-modal-tags-dropdown-empty";
            hint.textContent = 'Press Enter to add "' + filter + '"';
            tagsDropdown.appendChild(hint);
          }
          tagsDropdown.classList.toggle("open", !!f);
          return;
        }
        suggestions.forEach(function (t) {
          var item = document.createElement("div");
          item.className = "card-modal-tags-dropdown-item";
          item.textContent = t;
          item.addEventListener("mousedown", function (e) {
            e.preventDefault();
            addTag(t);
            tagsInput.focus();
          });
          tagsDropdown.appendChild(item);
        });
        tagsDropdown.classList.add("open");
      }

      function hideDropdown() {
        tagsDropdown.classList.remove("open");
        dropdownActiveIdx = -1;
      }

      tagsInput.addEventListener("input", function () { showDropdown(tagsInput.value); }, opts);
      tagsInput.addEventListener("focus", function () { showDropdown(tagsInput.value); }, opts);
      tagsInput.addEventListener("click", function () { showDropdown(tagsInput.value); }, opts);
      tagsInput.addEventListener("blur", function () { setTimeout(hideDropdown, 150); }, opts);

      tagsInput.addEventListener("keydown", function (e) {
        var items = tagsDropdown.querySelectorAll(".card-modal-tags-dropdown-item");
        if (e.key === "ArrowDown") {
          e.preventDefault();
          dropdownActiveIdx = Math.min(dropdownActiveIdx + 1, items.length - 1);
          items.forEach(function (it, i) { it.classList.toggle("active", i === dropdownActiveIdx); });
        } else if (e.key === "ArrowUp") {
          e.preventDefault();
          dropdownActiveIdx = Math.max(dropdownActiveIdx - 1, 0);
          items.forEach(function (it, i) { it.classList.toggle("active", i === dropdownActiveIdx); });
        } else if (e.key === "Enter") {
          e.preventDefault();
          if (dropdownActiveIdx >= 0 && items[dropdownActiveIdx]) {
            addTag(items[dropdownActiveIdx].textContent);
          } else if (tagsInput.value.trim()) {
            addTag(tagsInput.value);
          }
          tagsInput.focus();
        } else if (e.key === "Backspace" && !tagsInput.value && currentTags.length) {
          currentTags.pop();
          renderChips();
          tagsInput.placeholder = currentTags.length ? "" : "Add tags...";
        } else if (e.key === "Escape") {
          hideDropdown();
        }
      }, opts);

      bsTagsContainer.addEventListener("click", function () { tagsInput.focus(); }, opts);
      bsTagsContainer.appendChild(tagsInput);
      bsTagsContainer.appendChild(tagsDropdown);
      renderChips();
    }

    // Display settings
    var bsShowCheckbox = document.getElementById("bsShowCheckbox");
    var bsCardPosition = document.getElementById("bsCardPosition");
    var bsExpandColumns = document.getElementById("bsExpandColumns");
    var bsViewMode = document.getElementById("bsViewMode");
    var bsCardDisplayMode = document.getElementById("bsCardDisplayMode");

    // Populate all fields from data attributes
    function populateFromData() {
      // Meta fields
      if (bsBoardName) bsBoardName.value = boardView.dataset.boardName || "";
      if (bsBoardDescription) bsBoardDescription.value = boardView.dataset.boardDescription || "";

      // Tags
      currentTags = [];
      var tagsRaw = boardView.dataset.boardTags || "";
      if (tagsRaw) {
        tagsRaw.split(",").forEach(function (s) {
          s = s.trim();
          if (s && currentTags.indexOf(s) === -1) currentTags.push(s);
        });
      }
      initTagsUI();

      // Display settings
      if (bsShowCheckbox) bsShowCheckbox.value = boardView.dataset.bsShowCheckbox || "";
      if (bsCardPosition) bsCardPosition.value = boardView.dataset.bsCardPosition || "";
      if (bsExpandColumns) bsExpandColumns.value = boardView.dataset.bsExpandColumns || "false";
      if (bsViewMode) bsViewMode.value = boardView.dataset.bsViewMode || boardView.dataset.viewMode || "board";
      if (bsCardDisplayMode) bsCardDisplayMode.value = boardView.dataset.bsCardDisplayMode || "";

      // Show/hide reset buttons
      updateResetButtons();
    }

    // Save meta handler
    if (bsSaveMeta) {
      bsSaveMeta.addEventListener("click", function () {
        var newName = bsBoardName ? bsBoardName.value.trim() : "";
        if (!newName) return;
        htmx.ajax('POST', '/board/' + encodeURIComponent(slug) + '/meta', {
          values: {
            board_name: newName,
            description: bsBoardDescription ? bsBoardDescription.value.trim() : "",
            tags: currentTags.join(", "),
            name: slug,
            version: LB.getBoardVersion(),
          },
          target: '#board-content',
          swap: 'innerHTML'
        });
        if (bsMetaSavedMsg) {
          bsMetaSavedMsg.style.display = "";
          clearTimeout(tagsSavedTimer);
          tagsSavedTimer = setTimeout(function () { bsMetaSavedMsg.style.display = "none"; }, 2000);
        }
      }, opts);
    }

    function updateResetButtons() {
      panel.querySelectorAll(".btn-reset-setting").forEach(function (btn) {
        var setting = btn.dataset.setting;
        var hasOverride = false;
        if (setting === "show_checkbox") hasOverride = bsShowCheckbox && bsShowCheckbox.value !== "";
        if (setting === "card_position") hasOverride = bsCardPosition && bsCardPosition.value !== "";
        if (setting === "card_display_mode") hasOverride = bsCardDisplayMode && bsCardDisplayMode.value !== "";
        btn.style.display = hasOverride ? "" : "none";
      });
    }

    function sendBoardSettings() {
      var params = { name: slug, version: LB.getBoardVersion() };
      if (bsShowCheckbox && bsShowCheckbox.value !== "") {
        params.show_checkbox = bsShowCheckbox.value;
      }
      if (bsCardPosition && bsCardPosition.value !== "") {
        params.card_position = bsCardPosition.value;
      }
      if (bsExpandColumns) {
        params.expand_columns = bsExpandColumns.value;
      }
      if (bsViewMode) {
        params.view_mode = bsViewMode.value;
      }
      if (bsCardDisplayMode && bsCardDisplayMode.value !== "") {
        params.card_display_mode = bsCardDisplayMode.value;
      }
      htmx.ajax('POST', '/board/' + encodeURIComponent(slug) + '/settings', {
        values: params,
        target: '#board-content',
        swap: 'innerHTML'
      });
    }

    function openSettings() {
      backdrop.style.display = "";
      populateFromData();
    }
    function closeSettings() {
      backdrop.style.display = "none";
    }

    gearBtn.addEventListener("click", function () {
      var isOpen = backdrop.style.display !== "none";
      if (isOpen) closeSettings(); else openSettings();
    }, opts);

    if (closeBtn) {
      closeBtn.addEventListener("click", closeSettings, opts);
    }

    backdrop.addEventListener("click", function (e) {
      if (e.target === backdrop) closeSettings();
    }, opts);

    document.addEventListener("keydown", function (e) {
      if (e.key === "Escape" && backdrop.style.display !== "none") {
        closeSettings();
      }
    }, opts);

    // On change: send update
    [bsShowCheckbox, bsCardPosition, bsExpandColumns, bsViewMode, bsCardDisplayMode].forEach(function (el) {
      if (el) {
        el.addEventListener("change", function () {
          updateResetButtons();
          sendBoardSettings();
        }, opts);
      }
    });

    // Reset buttons
    panel.querySelectorAll(".btn-reset-setting").forEach(function (btn) {
      btn.addEventListener("click", function () {
        var setting = btn.dataset.setting;
        if (setting === "show_checkbox" && bsShowCheckbox) bsShowCheckbox.value = "";
        if (setting === "card_position" && bsCardPosition) bsCardPosition.value = "";
        if (setting === "card_display_mode" && bsCardDisplayMode) bsCardDisplayMode.value = "";
        updateResetButtons();
        sendBoardSettings();
      }, opts);
    });

    populateFromData();
  }

  LB.initBoardSettingsPanel = initBoardSettingsPanel;
})();
