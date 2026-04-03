package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"housekeeping/internal/models"
)

type TasksHandler struct {
	db *sql.DB
}

func NewTasksHandler(db *sql.DB) *TasksHandler {
	return &TasksHandler{db: db}
}

const taskSelectCols = `
	t.id, t.title, t.type,
	t.recurrence_unit, t.recurrence_every,
	t.due_date, t.assignee_id, t.category_id,
	t.status, t.completed_by, t.completed_at,
	t.notes, t.source, t.parent_id, t.created_at,
	p.name, c.name
`

const taskSelectFrom = `
	FROM tasks t
	LEFT JOIN people p ON p.id = t.assignee_id
	LEFT JOIN categories c ON c.id = t.category_id
`

func scanTask(row interface {
	Scan(dest ...any) error
}) (models.Task, error) {
	var t models.Task
	var recUnit sql.NullString
	var recEvery sql.NullInt64
	var dueDate sql.NullString
	var completedAt sql.NullString
	var assigneeName sql.NullString
	var categoryName sql.NullString

	err := row.Scan(
		&t.ID, &t.Title, &t.Type,
		&recUnit, &recEvery,
		&dueDate, &t.AssigneeID, &t.CategoryID,
		&t.Status, &t.CompletedBy, &completedAt,
		&t.Notes, &t.Source, &t.ParentID, &t.CreatedAt,
		&assigneeName, &categoryName,
	)
	if err != nil {
		return t, err
	}

	if recUnit.Valid && recEvery.Valid {
		t.Recurrence = &models.Recurrence{
			Unit:  recUnit.String,
			Every: int(recEvery.Int64),
		}
	}
	if dueDate.Valid {
		t.DueDate = dueDate.String
	}
	if completedAt.Valid {
		t.CompletedAt = &completedAt.String
	}
	if assigneeName.Valid {
		t.AssigneeName = &assigneeName.String
	}
	if categoryName.Valid {
		t.CategoryName = &categoryName.String
	}

	return t, nil
}

func (h *TasksHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	view := q.Get("view")     // today | upcoming | all
	status := q.Get("status") // pending | done | snoozed
	assignee := q.Get("assignee")
	category := q.Get("category")

	where := "WHERE 1=1"
	args := []any{}

	switch view {
	case "today":
		where += " AND t.due_date <= date('now') AND t.status = 'pending'"
	case "upcoming":
		where += " AND t.due_date BETWEEN date('now') AND date('now', '+7 days') AND t.status = 'pending'"
	}

	if status != "" {
		where += " AND t.status = ?"
		args = append(args, status)
	}
	if assignee != "" {
		where += " AND t.assignee_id = ?"
		args = append(args, assignee)
	}
	if category != "" {
		where += " AND t.category_id = ?"
		args = append(args, category)
	}

	query := fmt.Sprintf("SELECT %s %s %s ORDER BY t.due_date IS NULL ASC, t.due_date ASC, t.id ASC", taskSelectCols, taskSelectFrom, where)
	rows, err := h.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		respondErr(w, err, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	tasks := []models.Task{}
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			respondErr(w, err, http.StatusInternalServerError)
			return
		}
		tasks = append(tasks, t)
	}
	respondJSON(w, tasks)
}

func (h *TasksHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Title == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.Type != "one-off" && req.Type != "recurring" {
		http.Error(w, "type must be one-off or recurring", http.StatusBadRequest)
		return
	}
	if req.Type == "recurring" && req.Recurrence == nil {
		http.Error(w, "recurrence required for recurring tasks", http.StatusBadRequest)
		return
	}

	var recUnit, recEvery any
	if req.Recurrence != nil {
		recUnit = req.Recurrence.Unit
		recEvery = req.Recurrence.Every
	}

	res, err := h.db.ExecContext(r.Context(), `
		INSERT INTO tasks (title, type, recurrence_unit, recurrence_every, due_date, assignee_id, category_id, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, req.Title, req.Type, recUnit, recEvery, nullStr(req.DueDate), req.AssigneeID, req.CategoryID, req.Notes)
	if err != nil {
		respondErr(w, err, http.StatusInternalServerError)
		return
	}

	id, _ := res.LastInsertId()
	h.getAndRespond(w, r, id)
}

func (h *TasksHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req models.UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.Title != nil {
		if _, err := h.db.ExecContext(r.Context(), `UPDATE tasks SET title = ? WHERE id = ?`, *req.Title, id); err != nil {
			respondErr(w, err, http.StatusInternalServerError)
			return
		}
	}
	if req.DueDate != nil {
		if _, err := h.db.ExecContext(r.Context(), `UPDATE tasks SET due_date = ? WHERE id = ?`, nullStr(*req.DueDate), id); err != nil {
			respondErr(w, err, http.StatusInternalServerError)
			return
		}
	}
	if req.AssigneeID != nil {
		if _, err := h.db.ExecContext(r.Context(), `UPDATE tasks SET assignee_id = ? WHERE id = ?`, *req.AssigneeID, id); err != nil {
			respondErr(w, err, http.StatusInternalServerError)
			return
		}
	}
	if req.CategoryID != nil {
		if _, err := h.db.ExecContext(r.Context(), `UPDATE tasks SET category_id = ? WHERE id = ?`, *req.CategoryID, id); err != nil {
			respondErr(w, err, http.StatusInternalServerError)
			return
		}
	}
	if req.Notes != nil {
		if _, err := h.db.ExecContext(r.Context(), `UPDATE tasks SET notes = ? WHERE id = ?`, *req.Notes, id); err != nil {
			respondErr(w, err, http.StatusInternalServerError)
			return
		}
	}
	if req.Recurrence != nil {
		if _, err := h.db.ExecContext(r.Context(), `UPDATE tasks SET recurrence_unit = ?, recurrence_every = ? WHERE id = ?`,
			req.Recurrence.Unit, req.Recurrence.Every, id); err != nil {
			respondErr(w, err, http.StatusInternalServerError)
			return
		}
	}

	h.getAndRespond(w, r, id)
}

func (h *TasksHandler) Done(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req models.DoneRequest
	json.NewDecoder(r.Body).Decode(&req)

	now := time.Now().UTC().Format(time.RFC3339)

	// Get the task to check if it's recurring
	row := h.db.QueryRowContext(r.Context(), fmt.Sprintf("SELECT %s %s WHERE t.id = ?", taskSelectCols, taskSelectFrom), id)
	task, err := scanTask(row)
	if err != nil {
		respondErr(w, err, http.StatusNotFound)
		return
	}

	// Mark as done
	if _, err := h.db.ExecContext(r.Context(),
		`UPDATE tasks SET status = 'done', completed_by = ?, completed_at = ? WHERE id = ?`,
		req.CompletedBy, now, id,
	); err != nil {
		respondErr(w, err, http.StatusInternalServerError)
		return
	}

	// For recurring tasks, create the next occurrence
	if task.Type == "recurring" && task.Recurrence != nil && task.DueDate != "" {
		nextDue, err := nextDate(task.DueDate, task.Recurrence)
		if err == nil {
			parentID := id
			if task.ParentID != nil {
				parentID = *task.ParentID
			}
			h.db.ExecContext(r.Context(), `
				INSERT INTO tasks (title, type, recurrence_unit, recurrence_every, due_date, assignee_id, category_id, notes, source, parent_id)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, task.Title, task.Type, task.Recurrence.Unit, task.Recurrence.Every,
				nextDue, task.AssigneeID, task.CategoryID, task.Notes, task.Source, parentID)
		}
	}

	h.getAndRespond(w, r, id)
}

func (h *TasksHandler) Reopen(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if _, err := h.db.ExecContext(r.Context(),
		`UPDATE tasks SET status = 'pending', completed_by = NULL, completed_at = NULL WHERE id = ?`, id,
	); err != nil {
		respondErr(w, err, http.StatusInternalServerError)
		return
	}

	h.getAndRespond(w, r, id)
}

func (h *TasksHandler) Snooze(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req models.SnoozeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	var offset string
	switch req.Preset {
	case "tomorrow":
		offset = "+1 day"
	case "3days":
		offset = "+3 days"
	case "week":
		offset = "+7 days"
	case "month":
		offset = "+1 month"
	default:
		http.Error(w, "invalid preset", http.StatusBadRequest)
		return
	}

	newDue := time.Now().UTC()
	switch req.Preset {
	case "tomorrow":
		newDue = newDue.AddDate(0, 0, 1)
	case "3days":
		newDue = newDue.AddDate(0, 0, 3)
	case "week":
		newDue = newDue.AddDate(0, 0, 7)
	case "month":
		newDue = newDue.AddDate(0, 1, 0)
	}
	_ = offset

	if _, err := h.db.ExecContext(r.Context(),
		`UPDATE tasks SET status = 'snoozed', due_date = ? WHERE id = ?`,
		newDue.Format("2006-01-02"), id,
	); err != nil {
		respondErr(w, err, http.StatusInternalServerError)
		return
	}

	h.getAndRespond(w, r, id)
}

func (h *TasksHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if _, err := h.db.ExecContext(r.Context(), `DELETE FROM tasks WHERE id = ?`, id); err != nil {
		respondErr(w, err, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *TasksHandler) History(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	// Find the root parent
	rootID := id
	var parentID sql.NullInt64
	h.db.QueryRowContext(r.Context(), `SELECT COALESCE(parent_id, id) FROM tasks WHERE id = ?`, id).Scan(&parentID)
	if parentID.Valid {
		rootID = parentID.Int64
	}

	query := fmt.Sprintf(`SELECT %s %s WHERE (t.id = ? OR t.parent_id = ?) AND t.status = 'done' ORDER BY t.completed_at DESC`, taskSelectCols, taskSelectFrom)
	rows, err := h.db.QueryContext(r.Context(), query, rootID, rootID)
	if err != nil {
		respondErr(w, err, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	tasks := []models.Task{}
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			respondErr(w, err, http.StatusInternalServerError)
			return
		}
		tasks = append(tasks, t)
	}
	respondJSON(w, tasks)
}

func (h *TasksHandler) getAndRespond(w http.ResponseWriter, r *http.Request, id int64) {
	row := h.db.QueryRowContext(r.Context(), fmt.Sprintf("SELECT %s %s WHERE t.id = ?", taskSelectCols, taskSelectFrom), id)
	t, err := scanTask(row)
	if err != nil {
		respondErr(w, err, http.StatusInternalServerError)
		return
	}
	respondJSON(w, t)
}

func nextDate(from string, rec *models.Recurrence) (string, error) {
	t, err := time.Parse("2006-01-02", from)
	if err != nil {
		return "", err
	}
	switch rec.Unit {
	case "day":
		t = t.AddDate(0, 0, rec.Every)
	case "week":
		t = t.AddDate(0, 0, rec.Every*7)
	case "month":
		t = t.AddDate(0, rec.Every, 0)
	}
	return t.Format("2006-01-02"), nil
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
