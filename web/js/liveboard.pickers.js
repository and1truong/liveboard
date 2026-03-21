// LiveBoard: date picker and members picker.
(function () {
  var LB = window.LB;

  function createDatePicker(currentDue, onSelect) {
    var container = document.createElement("div");
    container.className = "card-modal-datepicker";
    container.addEventListener("click", function (e) { e.stopPropagation(); });

    var now = new Date();
    var viewYear = now.getFullYear();
    var viewMonth = now.getMonth();

    // Parse current due date
    if (currentDue) {
      var parts = currentDue.split("-");
      if (parts.length === 3) {
        viewYear = parseInt(parts[0], 10);
        viewMonth = parseInt(parts[1], 10) - 1;
      }
    }

    var monthNames = ["January","February","March","April","May","June",
      "July","August","September","October","November","December"];

    var header = document.createElement("div");
    header.className = "datepicker-header";
    var prevBtn = document.createElement("button");
    prevBtn.className = "datepicker-nav";
    prevBtn.innerHTML = "&#8249;";
    prevBtn.addEventListener("click", function () {
      viewMonth--;
      if (viewMonth < 0) { viewMonth = 11; viewYear--; }
      render();
    });
    var nextBtn = document.createElement("button");
    nextBtn.className = "datepicker-nav";
    nextBtn.innerHTML = "&#8250;";
    nextBtn.addEventListener("click", function () {
      viewMonth++;
      if (viewMonth > 11) { viewMonth = 0; viewYear++; }
      render();
    });
    var monthLabel = document.createElement("span");
    monthLabel.className = "datepicker-month";
    header.appendChild(prevBtn);
    header.appendChild(monthLabel);
    header.appendChild(nextBtn);
    container.appendChild(header);

    var weekdays = document.createElement("div");
    weekdays.className = "datepicker-weekdays";
    ["Su","Mo","Tu","We","Th","Fr","Sa"].forEach(function (d) {
      var s = document.createElement("span");
      s.textContent = d;
      weekdays.appendChild(s);
    });
    container.appendChild(weekdays);

    var grid = document.createElement("div");
    grid.className = "datepicker-grid";
    container.appendChild(grid);

    var removeBtn = document.createElement("button");
    removeBtn.className = "datepicker-remove";
    removeBtn.textContent = "Remove date";
    removeBtn.addEventListener("click", function () { onSelect(""); });
    container.appendChild(removeBtn);

    function pad(n) { return n < 10 ? "0" + n : "" + n; }

    function render() {
      monthLabel.textContent = monthNames[viewMonth] + " " + viewYear;
      grid.innerHTML = "";

      var firstDay = new Date(viewYear, viewMonth, 1).getDay();
      var daysInMonth = new Date(viewYear, viewMonth + 1, 0).getDate();
      var today = new Date();
      var todayStr = today.getFullYear() + "-" + pad(today.getMonth() + 1) + "-" + pad(today.getDate());

      for (var i = 0; i < firstDay; i++) {
        var empty = document.createElement("span");
        empty.className = "datepicker-day datepicker-day--empty";
        grid.appendChild(empty);
      }

      for (var d = 1; d <= daysInMonth; d++) {
        var dayBtn = document.createElement("button");
        dayBtn.className = "datepicker-day";
        dayBtn.textContent = d;
        var dateStr = viewYear + "-" + pad(viewMonth + 1) + "-" + pad(d);
        if (dateStr === todayStr) dayBtn.classList.add("datepicker-day--today");
        if (dateStr === currentDue) dayBtn.classList.add("datepicker-day--selected");
        dayBtn.dataset.date = dateStr;
        dayBtn.addEventListener("click", function () { onSelect(this.dataset.date); });
        grid.appendChild(dayBtn);
      }
    }

    render();
    return container;
  }

  function createMembersPicker(currentAssignee, onSelect) {
    var container = document.createElement("div");
    container.className = "card-modal-memberspicker";
    container.addEventListener("click", function (e) { e.stopPropagation(); });

    // Collect members from board data attribute and from all card assignees.
    var boardView = document.querySelector(".board-view");
    var boardMembersRaw = boardView ? (boardView.dataset.boardMembers || "") : "";
    var boardMembers = boardMembersRaw ? boardMembersRaw.split(",").map(function (s) { return s.trim(); }).filter(Boolean) : [];

    // Also collect assignees from all cards on the board.
    var allCards = document.querySelectorAll("[data-card-assignee]");
    allCards.forEach(function (c) {
      var a = c.dataset.cardAssignee;
      if (a && boardMembers.indexOf(a) === -1) {
        boardMembers.push(a);
      }
    });

    // Sort alphabetically.
    boardMembers.sort(function (a, b) { return a.toLowerCase().localeCompare(b.toLowerCase()); });

    function render() {
      container.innerHTML = "";

      // Title
      var title = document.createElement("div");
      title.className = "memberspicker-title";
      title.textContent = "Assign Member";
      container.appendChild(title);

      // Input for adding new member
      var inputRow = document.createElement("div");
      inputRow.className = "memberspicker-input-row";
      var input = document.createElement("input");
      input.type = "text";
      input.className = "memberspicker-input";
      input.placeholder = "Add new member...";
      var addBtn = document.createElement("button");
      addBtn.className = "memberspicker-add-btn";
      addBtn.textContent = "+";
      addBtn.addEventListener("click", function () {
        var name = input.value.trim();
        if (!name) return;
        if (boardMembers.indexOf(name) === -1) {
          boardMembers.push(name);
          boardMembers.sort(function (a, b) { return a.toLowerCase().localeCompare(b.toLowerCase()); });
        }
        input.value = "";
        render();
      });
      input.addEventListener("keydown", function (e) {
        if (e.key === "Enter") { e.preventDefault(); addBtn.click(); }
      });
      inputRow.appendChild(input);
      inputRow.appendChild(addBtn);
      container.appendChild(inputRow);

      // Member list
      if (boardMembers.length > 0) {
        var list = document.createElement("div");
        list.className = "memberspicker-list";

        boardMembers.forEach(function (member) {
          var item = document.createElement("button");
          item.className = "memberspicker-item" + (member === currentAssignee ? " memberspicker-item--active" : "");
          item.innerHTML = '<span class="memberspicker-avatar">' + member.charAt(0).toUpperCase() + "</span> " + member;
          item.addEventListener("click", function () {
            onSelect(member);
          });
          list.appendChild(item);
        });

        container.appendChild(list);
      } else {
        var empty = document.createElement("div");
        empty.className = "memberspicker-empty";
        empty.textContent = "No members yet. Add one above.";
        container.appendChild(empty);
      }

      // Clear assignee button
      var clearBtn = document.createElement("button");
      clearBtn.className = "memberspicker-clear";
      clearBtn.textContent = "Clear assignee";
      clearBtn.addEventListener("click", function () { onSelect(""); });
      container.appendChild(clearBtn);
    }

    render();
    return container;
  }

  LB.createDatePicker = createDatePicker;
  LB.createMembersPicker = createMembersPicker;
})();
