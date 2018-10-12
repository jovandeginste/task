package main

import (
	"time"

	humanize "github.com/dustin/go-humanize"
)

type TaskConfig struct {
	Tasks map[string]Task
}

type Task struct {
	Title      string            `yaml:"title"`
	Comments   []TaskComment     `yaml:"comments,omitempty"`
	Assignee   string            `yaml:"assignee,omitempty"`
	State      string            `yaml:"state,omitempty"`
	AfterTasks []string          `yaml:"after,omitempty"`
	CreatedAt  string            `yaml:"created_at,omitempty"`
	UpdatedAt  string            `yaml:"updated_at,omitempty"`
	Fields     map[string]string `yaml:"fields,omitempty"`
}

type TaskComment struct {
	Comment string
	By      string
	At      string
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
