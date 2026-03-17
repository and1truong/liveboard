package models

import "time"

// Board represents a Kanban board backed by a single Markdown file.
type Board struct {
	Name        string   `yaml:"name" json:"name"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	Tags        []string `yaml:"tags,omitempty" json:"tags,omitempty"`
	Columns     []Column `yaml:"-" json:"columns"`
	FilePath    string   `yaml:"-" json:"file_path"`
}

// Column represents a Kanban column (H2 heading in Markdown).
type Column struct {
	Name  string `json:"name"`
	Cards []Card `json:"cards"`
}

// Card represents a task item (list item in Markdown).
type Card struct {
	ID        string            `json:"id"`
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

type LLMConfig struct {
	Provider string `yaml:"provider,omitempty"`
	Model    string `yaml:"model,omitempty"`
}

type WorkspaceConfig struct {
	Default string `yaml:"default,omitempty"`
}

type GitConfig struct {
	AutoCommit    bool   `yaml:"auto_commit"`
	CommitFormat  string `yaml:"commit_format,omitempty"`
}

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
