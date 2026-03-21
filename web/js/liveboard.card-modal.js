// LiveBoard: full card detail modal + card click handler.
(function () {
  var LB = window.LB;
  var cardModal = null;
  var cardModalBackdrop = null;

  function hideCardModal() {
    if (cardModalBackdrop) {
      cardModalBackdrop.remove();
      cardModalBackdrop = null;
      cardModal = null;
    }
  }

  function showCardModal(card) {
    hideCardModal();
    LB.hideQuickEdit();

    var colIdx = card.dataset.colIdx;
    var cardIdx = card.dataset.cardIdx;
    var slug = decodeURIComponent(window.location.pathname.replace(/^\/board\//, ""));
    var title = card.dataset.cardTitle || "";
    var body = card.dataset.cardBody || "";
    var tags = card.dataset.cardTags || "";
    var assignee = card.dataset.cardAssignee || "";
    var priority = card.dataset.cardPriority || "";
    var due = card.dataset.cardDue || "";
    var dueValue = { current: due };
    var assigneeValue = { current: assignee };
    var completed = card.dataset.cardCompleted === "true";
    var columnName = card.dataset.cardColumn || "";

    // Backdrop
    var backdrop = document.createElement("div");
    backdrop.className = "card-modal-backdrop";
    backdrop.addEventListener("click", function (e) {
      if (e.target === backdrop) hideCardModal();
    });

    // Modal
    var modal = document.createElement("div");
    modal.className = "card-modal";

    // Close button
    var closeBtn = document.createElement("button");
    closeBtn.className = "card-modal-close";
    closeBtn.innerHTML = "&times;";
    closeBtn.addEventListener("click", hideCardModal);
    modal.appendChild(closeBtn);

    // Main content (left)
    var main = document.createElement("div");
    main.className = "card-modal-main";

    // Column pill
    var colPill = document.createElement("span");
    colPill.className = "card-modal-column-pill";
    colPill.textContent = columnName;
    main.appendChild(colPill);

    // Completed badge
    if (completed) {
      var doneBadge = document.createElement("span");
      doneBadge.className = "status-badge completed";
      doneBadge.innerHTML = "&#10003; Done";
      doneBadge.style.marginLeft = "8px";
      main.appendChild(doneBadge);
    }

    // Title
    var titleInput = document.createElement("textarea");
    titleInput.className = "card-modal-title";
    titleInput.value = title;
    titleInput.rows = 1;
    titleInput.addEventListener("input", function () {
      this.style.height = "auto";
      this.style.height = this.scrollHeight + "px";
    });
    main.appendChild(titleInput);

    // Action bar
    var actionBar = document.createElement("div");
    actionBar.className = "card-modal-action-bar";
    ["Labels", "Dates", "Checklist", "Members"].forEach(function (label) {
      var btn = document.createElement("button");
      btn.className = "card-modal-action-btn";
      var icons = { Labels: "🏷", Dates: "📅", Checklist: "☑", Members: "👤" };
      btn.innerHTML = '<span class="card-modal-action-icon">' + icons[label] + "</span> " + label;

      if (label === "Dates") {
        btn.addEventListener("click", function (e) {
          e.stopPropagation();
          var existing = document.querySelector(".card-modal-datepicker");
          if (existing) { existing.remove(); return; }

          var picker = LB.createDatePicker(dueValue.current, function (newDue) {
            dueValue.current = newDue;
            updateDueDisplay();
            picker.remove();
          });

          // Position below the action bar
          actionBar.style.position = "relative";
          actionBar.appendChild(picker);

          setTimeout(function () {
            document.addEventListener("click", function closePicker(ev) {
              if (!picker.parentNode) { document.removeEventListener("click", closePicker); return; }
              if (!picker.contains(ev.target) && ev.target !== btn) {
                picker.remove();
                document.removeEventListener("click", closePicker);
              }
            });
          }, 0);
        });
      }

      if (label === "Members") {
        btn.addEventListener("click", function (e) {
          e.stopPropagation();
          var existing = document.querySelector(".card-modal-memberspicker");
          if (existing) { existing.remove(); return; }

          var picker = LB.createMembersPicker(assigneeValue.current, function (member) {
            assigneeValue.current = member;
            updateAssigneeDisplay();
            picker.remove();
          });

          actionBar.style.position = "relative";
          actionBar.appendChild(picker);

          setTimeout(function () {
            document.addEventListener("click", function closePicker(ev) {
              if (!picker.parentNode) { document.removeEventListener("click", closePicker); return; }
              if (!picker.contains(ev.target) && ev.target !== btn) {
                picker.remove();
                document.removeEventListener("click", closePicker);
              }
            });
          }, 0);
        });
      }

      actionBar.appendChild(btn);
    });
    main.appendChild(actionBar);

    // Description section
    var descHeader = document.createElement("div");
    descHeader.className = "card-modal-section-header";
    descHeader.innerHTML = '<span class="card-modal-section-icon">≡</span> Description';
    main.appendChild(descHeader);

    var bodyInput = document.createElement("textarea");
    bodyInput.className = "card-modal-body";
    bodyInput.placeholder = "Add a more detailed description...";
    bodyInput.value = body;
    bodyInput.rows = 4;
    bodyInput.addEventListener("input", function () {
      this.style.height = "auto";
      this.style.height = this.scrollHeight + "px";
    });
    main.appendChild(bodyInput);

    // Tags section
    var tagsHeader = document.createElement("div");
    tagsHeader.className = "card-modal-section-header";
    tagsHeader.innerHTML = '<span class="card-modal-section-icon">🏷</span> Tags';
    main.appendChild(tagsHeader);

    // Collect all unique tags from all cards on the board
    var allBoardTags = [];
    var allCards = document.querySelectorAll(".card[data-card-tags]");
    allCards.forEach(function (c) {
      var t = c.dataset.cardTags || "";
      t.split(",").forEach(function (s) {
        s = s.trim();
        if (s && allBoardTags.indexOf(s) === -1) allBoardTags.push(s);
      });
    });
    allBoardTags.sort(function (a, b) { return a.toLowerCase().localeCompare(b.toLowerCase()); });

    // Parse current tags
    var currentTags = [];
    if (tags) {
      tags.split(",").forEach(function (s) {
        s = s.trim();
        if (s && currentTags.indexOf(s) === -1) currentTags.push(s);
      });
    }

    // Tag container (chips + input + dropdown)
    var tagsContainer = document.createElement("div");
    tagsContainer.className = "card-modal-tags-container";

    var tagsInput = document.createElement("input");
    tagsInput.className = "card-modal-tags-input";
    tagsInput.type = "text";
    tagsInput.placeholder = currentTags.length ? "" : "Add tags...";

    var tagsDropdown = document.createElement("div");
    tagsDropdown.className = "card-modal-tags-dropdown";
    var dropdownActiveIdx = -1;

    function getTagsValue() {
      return currentTags.join(", ");
    }

    function renderChips() {
      // Remove existing chips
      Array.from(tagsContainer.querySelectorAll(".card-modal-tag-chip")).forEach(function (el) { el.remove(); });
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
        tagsContainer.insertBefore(chip, tagsInput);
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
      var suggestions = allBoardTags.filter(function (t) {
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

    tagsInput.addEventListener("input", function () {
      showDropdown(tagsInput.value);
    });

    tagsInput.addEventListener("focus", function () {
      showDropdown(tagsInput.value);
    });

    tagsInput.addEventListener("click", function () {
      showDropdown(tagsInput.value);
    });

    tagsInput.addEventListener("blur", function () {
      // Small delay to allow mousedown on dropdown items
      setTimeout(hideDropdown, 150);
    });

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
    });

    tagsContainer.addEventListener("click", function () {
      tagsInput.focus();
    });

    tagsContainer.appendChild(tagsInput);
    tagsContainer.appendChild(tagsDropdown);
    renderChips();
    main.appendChild(tagsContainer);

    // Save button
    var saveRow = document.createElement("div");
    saveRow.className = "card-modal-save-row";
    var saveBtn = document.createElement("button");
    saveBtn.className = "btn-primary btn-small";
    saveBtn.textContent = "Save";
    saveBtn.addEventListener("click", function () {
      htmx.ajax('POST', '/board/' + encodeURIComponent(slug) + '/cards/edit', {
        values: {
          col_idx: colIdx,
          card_idx: cardIdx,
          title: titleInput.value.trim(),
          body: bodyInput.value.trim(),
          tags: getTagsValue(),
          priority: priorityValue.current,
          due: dueValue.current,
          assignee: assigneeValue.current,
          name: slug,
          version: LB.getBoardVersion(),
        },
        target: '#board-content',
        swap: 'innerHTML'
      });
      hideCardModal();
    });
    saveRow.appendChild(saveBtn);
    main.appendChild(saveRow);

    modal.appendChild(main);

    // Sidebar (right)
    var sidebar = document.createElement("div");
    sidebar.className = "card-modal-sidebar";

    var actionsLabel = document.createElement("div");
    actionsLabel.className = "card-modal-sidebar-label";
    actionsLabel.textContent = "Actions";
    sidebar.appendChild(actionsLabel);

    // Complete/Incomplete
    var completeBtn = card.querySelector('[hx-post$="/cards/complete"]');
    if (completeBtn || completed) {
      var toggleBtn = document.createElement("button");
      toggleBtn.className = "card-modal-sidebar-btn";
      toggleBtn.innerHTML = completed
        ? '<span class="card-modal-action-icon">↩</span> Mark Incomplete'
        : '<span class="card-modal-action-icon">✓</span> Complete';
      toggleBtn.addEventListener("click", function () {
        hideCardModal();
        htmx.ajax('POST', '/board/' + encodeURIComponent(slug) + '/cards/complete', {
          values: { col_idx: colIdx, card_idx: cardIdx, name: slug, version: LB.getBoardVersion() },
          target: '#board-content',
          swap: 'innerHTML'
        });
      });
      sidebar.appendChild(toggleBtn);
    }

    // Move to
    var moveTriggers = Array.from(card.querySelectorAll(".move-trigger[data-target]"));
    if (moveTriggers.length > 0) {
      var moveLabel = document.createElement("div");
      moveLabel.className = "card-modal-sidebar-sublabel";
      moveLabel.textContent = "Move to";
      sidebar.appendChild(moveLabel);

      moveTriggers.forEach(function (trigger) {
        var target = trigger.dataset.target;
        var moveBtn = document.createElement("button");
        moveBtn.className = "card-modal-sidebar-btn card-modal-sidebar-btn-move";
        moveBtn.innerHTML = '<span class="card-modal-action-icon">→</span> ' + target;
        moveBtn.addEventListener("click", function () {
          hideCardModal();
          trigger.click();
        });
        sidebar.appendChild(moveBtn);
      });
    }

    // Delete
    var deleteHidden = card.querySelector('[hx-post$="/cards/delete"]');
    if (deleteHidden) {
      var sep = document.createElement("div");
      sep.className = "card-modal-sidebar-sep";
      sidebar.appendChild(sep);

      var delBtn = document.createElement("button");
      delBtn.className = "card-modal-sidebar-btn card-modal-sidebar-btn-danger";
      delBtn.innerHTML = '<span class="card-modal-action-icon">🗑</span> Delete';
      delBtn.addEventListener("click", function () {
        hideCardModal();
        deleteHidden.click();
      });
      sidebar.appendChild(delBtn);
    }

    // Priority section
    var prioritySep = document.createElement("div");
    prioritySep.className = "card-modal-sidebar-sep";
    sidebar.appendChild(prioritySep);

    var priorityLabel = document.createElement("div");
    priorityLabel.className = "card-modal-sidebar-label";
    priorityLabel.textContent = "Priority";
    sidebar.appendChild(priorityLabel);

    var priorityGroup = document.createElement("div");
    priorityGroup.className = "card-modal-priority-group";
    var priorityValue = { current: priority || "" };
    var priorityBtns = [];
    [
      { val: "", label: "—", title: "None" },
      { val: "low", label: "L", title: "Low" },
      { val: "medium", label: "M", title: "Medium" },
      { val: "high", label: "H", title: "High" },
      { val: "critical", label: "!", title: "Critical" },
    ].forEach(function (item) {
      var btn = document.createElement("button");
      btn.className =
        "card-modal-priority-btn" +
        (item.val === priorityValue.current
          ? " card-modal-priority-btn--active"
          : "") +
        (item.val ? " card-modal-priority-btn--" + item.val : "");
      btn.textContent = item.label;
      btn.title = item.title;
      btn.addEventListener("click", function () {
        priorityValue.current = item.val;
        priorityBtns.forEach(function (b) {
          b.className = b.className
            .replace(" card-modal-priority-btn--active", "");
        });
        btn.className += " card-modal-priority-btn--active";
      });
      priorityGroup.appendChild(btn);
      priorityBtns.push(btn);
    });
    sidebar.appendChild(priorityGroup);

    // Other metadata
    var metaWrap = document.createElement("div");
    metaWrap.className = "card-modal-meta-list";
    var assigneeDisplayEl = document.createElement("div");
    assigneeDisplayEl.className = "card-modal-meta-item";
    function updateAssigneeDisplay() {
      assigneeDisplayEl.innerHTML = assigneeValue.current
        ? "👤 " + assigneeValue.current
        : "👤 No assignee";
      assigneeDisplayEl.style.opacity = assigneeValue.current ? "1" : "0.5";
    }
    updateAssigneeDisplay();
    metaWrap.appendChild(assigneeDisplayEl);
    var dueDisplayEl = document.createElement("div");
    dueDisplayEl.className = "card-modal-meta-item";
    function updateDueDisplay() {
      dueDisplayEl.innerHTML = dueValue.current
        ? "📅 " + dueValue.current
        : "📅 No due date";
      dueDisplayEl.style.opacity = dueValue.current ? "1" : "0.5";
    }
    updateDueDisplay();
    metaWrap.appendChild(dueDisplayEl);
    sidebar.appendChild(metaWrap);

    modal.appendChild(sidebar);
    backdrop.appendChild(modal);
    document.body.appendChild(backdrop);
    cardModalBackdrop = backdrop;
    cardModal = modal;

    // Auto-size title
    titleInput.style.height = "auto";
    titleInput.style.height = titleInput.scrollHeight + "px";
    titleInput.focus();
  }

  // Attach card click for modal (distinguish from drag)
  function attachCardClick() {
    document.querySelectorAll(".card[data-card-idx]").forEach(function (card) {
      if (card.dataset.modalWired) return;
      card.dataset.modalWired = "1";

      card.addEventListener("mousedown", function () {
        LB.isDragging = false;
      });

      card.addEventListener("click", function (e) {
        if (LB.isDragging) return;
        // Don't open modal if clicking a button or link
        if (e.target.closest("button, a, input, textarea, select")) return;
        showCardModal(card);
      });
    });
  }

  // Escape key for modal
  document.addEventListener("keydown", function (e) {
    if (e.key === "Escape" && cardModalBackdrop) {
      hideCardModal();
    }
  });

  var collapsedHeaderClickWired = false;
  function attachCollapsedColumnClick() {
    if (collapsedHeaderClickWired) return;
    collapsedHeaderClickWired = true;
    document.addEventListener("click", function (e) {
      var header = e.target.closest(".column-header");
      if (!header) return;
      var col = header.closest(".column");
      if (!col || !col.classList.contains("collapsed")) return;
      var btn = header.querySelector(".column-collapse-btn");
      if (btn && e.target !== btn) btn.click();
    });
  }

  LB.hideCardModal = hideCardModal;
  LB.attachCardClick = attachCardClick;
  LB.attachCollapsedColumnClick = attachCollapsedColumnClick;
})();
