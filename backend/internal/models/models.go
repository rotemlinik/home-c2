package models

type Person struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

type Category struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Recurrence struct {
	Unit  string `json:"unit"`  // day | week | month
	Every int    `json:"every"` // e.g. every 2 weeks
}

type Task struct {
	ID          int64       `json:"id"`
	Title       string      `json:"title"`
	Type        string      `json:"type"` // one-off | recurring
	Recurrence  *Recurrence `json:"recurrence,omitempty"`
	DueDate     string      `json:"due_date,omitempty"`
	AssigneeID  *int64      `json:"assignee_id,omitempty"`
	CategoryID  *int64      `json:"category_id,omitempty"`
	Status      string      `json:"status"` // pending | done | snoozed
	CompletedBy *int64      `json:"completed_by,omitempty"`
	CompletedAt *string     `json:"completed_at,omitempty"`
	Notes       string      `json:"notes"`
	Source      string      `json:"source"`
	ParentID    *int64      `json:"parent_id,omitempty"`
	CreatedAt   string      `json:"created_at"`

	// Joined fields
	AssigneeName *string `json:"assignee_name,omitempty"`
	CategoryName *string `json:"category_name,omitempty"`
}

type CreateTaskRequest struct {
	Title      string      `json:"title"`
	Type       string      `json:"type"`
	Recurrence *Recurrence `json:"recurrence,omitempty"`
	DueDate    string      `json:"due_date"`
	AssigneeID *int64      `json:"assignee_id,omitempty"`
	CategoryID *int64      `json:"category_id,omitempty"`
	Notes      string      `json:"notes"`
}

type UpdateTaskRequest struct {
	Title      *string     `json:"title,omitempty"`
	Recurrence *Recurrence `json:"recurrence,omitempty"`
	DueDate    *string     `json:"due_date,omitempty"`
	AssigneeID *int64      `json:"assignee_id,omitempty"`
	CategoryID *int64      `json:"category_id,omitempty"`
	Notes      *string     `json:"notes,omitempty"`
}

type SnoozeRequest struct {
	Preset string `json:"preset"` // tomorrow | 3days | week | month
}

type DoneRequest struct {
	CompletedBy *int64 `json:"completed_by,omitempty"`
	Notes       string `json:"notes"`
}
