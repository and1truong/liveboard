// Package models defines the core data types for liveboard.
package models

import "time"

// BoardSettings holds per-board setting overrides.
// Pointer types allow nil = "inherit global default".
type BoardSettings struct {
	ShowCheckbox    *bool   `yaml:"show-checkbox,omitempty" json:"show_checkbox,omitempty"`
	CardPosition    *string `yaml:"card-position,omitempty" json:"card_position,omitempty"`
	ExpandColumns   *bool   `yaml:"expand-columns,omitempty" json:"expand_columns,omitempty"`
	ViewMode        *string `yaml:"view-mode,omitempty" json:"view_mode,omitempty"`
	CardDisplayMode *string `yaml:"card-display-mode,omitempty" json:"card_display_mode,omitempty"`
}

// Board represents a Kanban board backed by a single Markdown file.
type Board struct {
	Version      int           `yaml:"version" json:"version"`
	Name         string        `yaml:"name" json:"name"`
	Description  string        `yaml:"description,omitempty" json:"description,omitempty"`
	Icon         string        `yaml:"icon,omitempty" json:"icon,omitempty"`
	Tags         []string      `yaml:"tags,omitempty" json:"tags,omitempty"`
	ListCollapse []bool        `yaml:"list-collapse,omitempty" json:"list_collapse,omitempty"`
	Members      []string      `yaml:"members,omitempty" json:"members,omitempty"`
	Settings     BoardSettings `yaml:"settings,omitempty" json:"settings,omitempty"`
	Columns      []Column      `yaml:"-" json:"columns"`
	FilePath     string        `yaml:"-" json:"file_path"`
}

// Column represents a Kanban column (H2 heading in Markdown).
type Column struct {
	Name      string `json:"name"`
	Collapsed bool   `json:"collapsed"`
	Cards     []Card `json:"cards"`
}

// Card represents a task item (list item in Markdown).
type Card struct {
	Title     string            `json:"title"`
	Completed bool              `json:"completed"`
	Tags      []string          `json:"tags,omitempty"`
	Assignee  string            `json:"assignee,omitempty"`
	Priority  string            `json:"priority,omitempty"`
	Due       string            `json:"due,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Body      string            `json:"body,omitempty"`
}

// Config represents global or project-level configuration.
type Config struct {
	LLM       LLMConfig       `yaml:"llm,omitempty"`
	Workspace WorkspaceConfig `yaml:"workspace,omitempty"`
	Git       GitConfig       `yaml:"git,omitempty"`
	Board     BoardConfig     `yaml:"board,omitempty"`
}

// LLMConfig holds configuration for the LLM integration.
type LLMConfig struct {
	Provider string `yaml:"provider,omitempty"`
	Model    string `yaml:"model,omitempty"`
}

// WorkspaceConfig holds workspace-level settings.
type WorkspaceConfig struct {
	Default string `yaml:"default,omitempty"`
}

// GitConfig holds git auto-commit settings.
type GitConfig struct {
	AutoCommit   bool   `yaml:"auto_commit"`
	CommitFormat string `yaml:"commit_format,omitempty"`
}

// BoardConfig holds board-level defaults.
type BoardConfig struct {
	DefaultColumns []string `yaml:"default_columns,omitempty"`
}

// Event represents an internal state change event.
type Event struct {
	Type      string         `json:"type"`
	Board     string         `json:"board"`
	EntityID  string         `json:"entity_id"`
	Payload   map[string]any `json:"payload,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
}
