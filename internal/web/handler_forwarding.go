package web

import "net/http"

// Forwarding methods on Handler delegate to sub-handlers.
// These maintain backward compatibility with server.go and tests.
// They can be removed once all callers reference sub-handlers directly.

// BoardListPage forwards to BoardList.
func (h *Handler) BoardListPage(w http.ResponseWriter, r *http.Request) {
	h.BoardList.BoardListPage(w, r)
}

// HandleCreateBoard forwards to BoardList.
func (h *Handler) HandleCreateBoard(w http.ResponseWriter, r *http.Request) {
	h.BoardList.HandleCreateBoard(w, r)
}

// HandleDeleteBoard forwards to BoardList.
func (h *Handler) HandleDeleteBoard(w http.ResponseWriter, r *http.Request) {
	h.BoardList.HandleDeleteBoard(w, r)
}

// HandleSetBoardIconList forwards to BoardList.
func (h *Handler) HandleSetBoardIconList(w http.ResponseWriter, r *http.Request) {
	h.BoardList.HandleSetBoardIconList(w, r)
}

// HandleTogglePin forwards to BoardList.
func (h *Handler) HandleTogglePin(w http.ResponseWriter, r *http.Request) {
	h.BoardList.HandleTogglePin(w, r)
}

// HandleSidebarBoards forwards to BoardList.
func (h *Handler) HandleSidebarBoards(w http.ResponseWriter, r *http.Request) {
	h.BoardList.HandleSidebarBoards(w, r)
}

// BoardViewPage forwards to BoardView.
func (h *Handler) BoardViewPage(w http.ResponseWriter, r *http.Request) {
	h.BoardView.BoardViewPage(w, r)
}

// BoardContent forwards to BoardView.
func (h *Handler) BoardContent(w http.ResponseWriter, r *http.Request) {
	h.BoardView.BoardContent(w, r)
}

// HandleCreateCard forwards to BoardView.
func (h *Handler) HandleCreateCard(w http.ResponseWriter, r *http.Request) {
	h.BoardView.HandleCreateCard(w, r)
}

// HandleMoveCard forwards to BoardView.
func (h *Handler) HandleMoveCard(w http.ResponseWriter, r *http.Request) {
	h.BoardView.HandleMoveCard(w, r)
}

// HandleMoveCardToBoard forwards to BoardView.
func (h *Handler) HandleMoveCardToBoard(w http.ResponseWriter, r *http.Request) {
	h.BoardView.HandleMoveCardToBoard(w, r)
}

// HandleReorderCard forwards to BoardView.
func (h *Handler) HandleReorderCard(w http.ResponseWriter, r *http.Request) {
	h.BoardView.HandleReorderCard(w, r)
}

// HandleDeleteCard forwards to BoardView.
func (h *Handler) HandleDeleteCard(w http.ResponseWriter, r *http.Request) {
	h.BoardView.HandleDeleteCard(w, r)
}

// HandleToggleComplete forwards to BoardView.
func (h *Handler) HandleToggleComplete(w http.ResponseWriter, r *http.Request) {
	h.BoardView.HandleToggleComplete(w, r)
}

// HandleEditCard forwards to BoardView.
func (h *Handler) HandleEditCard(w http.ResponseWriter, r *http.Request) {
	h.BoardView.HandleEditCard(w, r)
}

// HandleCreateColumn forwards to BoardView.
func (h *Handler) HandleCreateColumn(w http.ResponseWriter, r *http.Request) {
	h.BoardView.HandleCreateColumn(w, r)
}

// HandleRenameColumn forwards to BoardView.
func (h *Handler) HandleRenameColumn(w http.ResponseWriter, r *http.Request) {
	h.BoardView.HandleRenameColumn(w, r)
}

// HandleDeleteColumn forwards to BoardView.
func (h *Handler) HandleDeleteColumn(w http.ResponseWriter, r *http.Request) {
	h.BoardView.HandleDeleteColumn(w, r)
}

// HandleToggleColumnCollapse forwards to BoardView.
func (h *Handler) HandleToggleColumnCollapse(w http.ResponseWriter, r *http.Request) {
	h.BoardView.HandleToggleColumnCollapse(w, r)
}

// HandleSortColumn forwards to BoardView.
func (h *Handler) HandleSortColumn(w http.ResponseWriter, r *http.Request) {
	h.BoardView.HandleSortColumn(w, r)
}

// HandleMoveColumn forwards to BoardView.
func (h *Handler) HandleMoveColumn(w http.ResponseWriter, r *http.Request) {
	h.BoardView.HandleMoveColumn(w, r)
}

// HandleUpdateBoardMeta forwards to BoardView.
func (h *Handler) HandleUpdateBoardMeta(w http.ResponseWriter, r *http.Request) {
	h.BoardView.HandleUpdateBoardMeta(w, r)
}

// HandleUpdateBoardSettings forwards to BoardView.
func (h *Handler) HandleUpdateBoardSettings(w http.ResponseWriter, r *http.Request) {
	h.BoardView.HandleUpdateBoardSettings(w, r)
}

// HandleSetBoardIcon forwards to BoardView.
func (h *Handler) HandleSetBoardIcon(w http.ResponseWriter, r *http.Request) {
	h.BoardView.HandleSetBoardIcon(w, r)
}

// RemindersPage forwards to Reminders.
func (h *Handler) RemindersPage(w http.ResponseWriter, r *http.Request) {
	h.Reminders.RemindersPage(w, r)
}

// HandleSetReminder forwards to Reminders.
func (h *Handler) HandleSetReminder(w http.ResponseWriter, r *http.Request) {
	h.Reminders.HandleSetReminder(w, r)
}

// HandleDismissReminder forwards to Reminders.
func (h *Handler) HandleDismissReminder(w http.ResponseWriter, r *http.Request) {
	h.Reminders.HandleDismissReminder(w, r)
}

// HandleSnoozeReminder forwards to Reminders.
func (h *Handler) HandleSnoozeReminder(w http.ResponseWriter, r *http.Request) {
	h.Reminders.HandleSnoozeReminder(w, r)
}

// HandleDeleteReminder forwards to Reminders.
func (h *Handler) HandleDeleteReminder(w http.ResponseWriter, r *http.Request) {
	h.Reminders.HandleDeleteReminder(w, r)
}

// HandleClearFired forwards to Reminders.
func (h *Handler) HandleClearFired(w http.ResponseWriter, r *http.Request) {
	h.Reminders.HandleClearFired(w, r)
}

// HandleClearHistory forwards to Reminders.
func (h *Handler) HandleClearHistory(w http.ResponseWriter, r *http.Request) {
	h.Reminders.HandleClearHistory(w, r)
}

// HandleUpdateReminderSettings forwards to Reminders.
func (h *Handler) HandleUpdateReminderSettings(w http.ResponseWriter, r *http.Request) {
	h.Reminders.HandleUpdateReminderSettings(w, r)
}

// SettingsHandler forwards to Settings.
func (h *Handler) SettingsHandler() http.Handler {
	return h.Settings.SettingsHandler()
}

// SettingsAPIHandler forwards to Settings.
func (h *Handler) SettingsAPIHandler() http.Handler {
	return h.Settings.SettingsAPIHandler()
}
