package main

import (
	"time"

	humanize "github.com/dustin/go-humanize"
)

type TaskConfig struct {
	Tasks map[string]Task `json:"tasks" yaml:"tasks"`
}

type Task struct {
	Title      string            `json:"title" yaml:"title"`
	Comments   []TaskComment     `json:"comments,omitempty" yaml:"comments,omitempty"`
	Assignee   string            `json:"assignee,omitempty" yaml:"assignee,omitempty"`
	State      string            `json:"state,omitempty" yaml:"state,omitempty"`
	AfterTasks []string          `json:"after,omitempty" yaml:"after,omitempty"`
	CreatedAt  string            `json:"created_at,omitempty" yaml:"created_at,omitempty"`
	UpdatedAt  string            `json:"updated_at,omitempty" yaml:"updated_at,omitempty"`
	Fields     map[string]string `json:"fields,omitempty" yaml:"fields,omitempty"`
}

type TaskComment struct {
	Comment string `json:"comment" yaml:"comment"`
	By      string `json:"by" yaml:"by"`
	At      string `json:"at" yaml:"at"`
}

func (tc *TaskComment) HumanAt() string {
	return humanAt(tc.At)
}

func (t *Task) HumanCreatedAt() string {
	return humanAt(t.CreatedAt)
}

func (t *Task) HumanUpdatedAt() string {
	return humanAt(t.UpdatedAt)
}

func (t *Task) Update() {
	now := time.Now().Format(time.RFC3339)
	t.UpdatedAt = now
	if t.CreatedAt == "" {
		t.CreatedAt = now
	}
}

func humanAt(theTime string) string {
	t, _ := time.Parse(time.RFC3339, theTime)
	return humanize.Time(t)
}
