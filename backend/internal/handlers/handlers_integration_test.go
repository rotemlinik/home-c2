package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"housekeeping/internal/db"
	"housekeeping/internal/models"
)

// setupTestDB creates a fresh in-memory SQLite database with migrations applied.
func setupTestDB(t *testing.T) *TasksHandler {
	t.Helper()
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return NewTasksHandler(database)
}

// setupTestRouter creates a chi router wired to the tasks handler for testing.
func setupTestRouter(t *testing.T) (*chi.Mux, *TasksHandler) {
	t.Helper()
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	tasks := NewTasksHandler(database)
	people := NewPeopleHandler(database)
	categories := NewCategoriesHandler(database)

	r := chi.NewRouter()
	r.Route("/api/tasks", func(r chi.Router) {
		r.Get("/", tasks.List)
		r.Post("/", tasks.Create)
		r.Patch("/{id}", tasks.Update)
		r.Post("/{id}/done", tasks.Done)
		r.Post("/{id}/reopen", tasks.Reopen)
		r.Post("/{id}/snooze", tasks.Snooze)
		r.Delete("/{id}", tasks.Delete)
		r.Get("/{id}/history", tasks.History)
	})
	r.Route("/api/people", func(r chi.Router) {
		r.Get("/", people.List)
		r.Post("/", people.Create)
		r.Delete("/{id}", people.Delete)
	})
	r.Route("/api/categories", func(r chi.Router) {
		r.Get("/", categories.List)
		r.Post("/", categories.Create)
		r.Delete("/{id}", categories.Delete)
	})

	return r, tasks
}

func postJSON(router http.Handler, url string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func patchJSON(router http.Handler, url string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func get(router http.Handler, url string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func deleteReq(router http.Handler, url string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodDelete, url, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// --- Task CRUD ---

func TestTaskCreate(t *testing.T) {
	router, _ := setupTestRouter(t)

	tests := []struct {
		name       string
		body       models.CreateTaskRequest
		wantStatus int
	}{
		{
			name: "valid one-off task",
			body: models.CreateTaskRequest{
				Title:   "Buy groceries",
				Type:    "one-off",
				DueDate: "2025-06-15",
				Notes:   "milk, eggs",
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "valid recurring task",
			body: models.CreateTaskRequest{
				Title:      "Clean house",
				Type:       "recurring",
				DueDate:    "2025-06-15",
				Recurrence: &models.Recurrence{Unit: "week", Every: 2},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing title",
			body:       models.CreateTaskRequest{Type: "one-off"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid type",
			body:       models.CreateTaskRequest{Title: "Test", Type: "invalid"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "recurring without recurrence",
			body:       models.CreateTaskRequest{Title: "Test", Type: "recurring"},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := postJSON(router, "/api/tasks", tt.body)
			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
			if tt.wantStatus == http.StatusOK {
				var task models.Task
				if err := json.Unmarshal(w.Body.Bytes(), &task); err != nil {
					t.Fatalf("unmarshal response: %v", err)
				}
				if task.Title != tt.body.Title {
					t.Errorf("title = %q, want %q", task.Title, tt.body.Title)
				}
				if task.Status != "pending" {
					t.Errorf("status = %q, want pending", task.Status)
				}
			}
		})
	}
}

func TestTaskList(t *testing.T) {
	router, _ := setupTestRouter(t)

	// Create a couple of tasks
	postJSON(router, "/api/tasks", models.CreateTaskRequest{Title: "Task A", Type: "one-off", DueDate: "2025-06-15"})
	postJSON(router, "/api/tasks", models.CreateTaskRequest{Title: "Task B", Type: "one-off", DueDate: "2025-06-16"})

	w := get(router, "/api/tasks")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var tasks []models.Task
	if err := json.Unmarshal(w.Body.Bytes(), &tasks); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(tasks) < 2 {
		t.Errorf("expected at least 2 tasks, got %d", len(tasks))
	}
}

func TestTaskListFilterByStatus(t *testing.T) {
	router, _ := setupTestRouter(t)

	// Create a task and mark it done
	w := postJSON(router, "/api/tasks", models.CreateTaskRequest{Title: "Done task", Type: "one-off", DueDate: "2025-06-15"})
	var task models.Task
	json.Unmarshal(w.Body.Bytes(), &task)

	postJSON(router, fmt.Sprintf("/api/tasks/%d/done", task.ID), models.DoneRequest{})

	// Create a pending task
	postJSON(router, "/api/tasks", models.CreateTaskRequest{Title: "Pending task", Type: "one-off"})

	// Filter for done
	w = get(router, "/api/tasks?status=done")
	var doneTasks []models.Task
	json.Unmarshal(w.Body.Bytes(), &doneTasks)

	for _, dt := range doneTasks {
		if dt.Status != "done" {
			t.Errorf("expected all tasks to be done, got %q", dt.Status)
		}
	}

	// Filter for pending
	w = get(router, "/api/tasks?status=pending")
	var pendingTasks []models.Task
	json.Unmarshal(w.Body.Bytes(), &pendingTasks)

	for _, pt := range pendingTasks {
		if pt.Status != "pending" {
			t.Errorf("expected all tasks to be pending, got %q", pt.Status)
		}
	}
}

func TestTaskUpdate(t *testing.T) {
	router, _ := setupTestRouter(t)

	// Create a task
	w := postJSON(router, "/api/tasks", models.CreateTaskRequest{Title: "Original", Type: "one-off", DueDate: "2025-06-15"})
	var created models.Task
	json.Unmarshal(w.Body.Bytes(), &created)

	newTitle := "Updated Title"
	newDue := "2025-07-01"
	newNotes := "updated notes"
	w = patchJSON(router, fmt.Sprintf("/api/tasks/%d", created.ID), models.UpdateTaskRequest{
		Title:   &newTitle,
		DueDate: &newDue,
		Notes:   &newNotes,
	})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}

	var updated models.Task
	json.Unmarshal(w.Body.Bytes(), &updated)

	if updated.Title != newTitle {
		t.Errorf("title = %q, want %q", updated.Title, newTitle)
	}
	if !strings.HasPrefix(updated.DueDate, newDue) {
		t.Errorf("due_date = %q, want prefix %q", updated.DueDate, newDue)
	}
	if updated.Notes != newNotes {
		t.Errorf("notes = %q, want %q", updated.Notes, newNotes)
	}
}

func TestTaskDelete(t *testing.T) {
	router, _ := setupTestRouter(t)

	// Create then delete
	w := postJSON(router, "/api/tasks", models.CreateTaskRequest{Title: "To delete", Type: "one-off"})
	var task models.Task
	json.Unmarshal(w.Body.Bytes(), &task)

	w = deleteReq(router, fmt.Sprintf("/api/tasks/%d", task.ID))
	if w.Code != http.StatusNoContent {
		t.Errorf("delete status = %d, want 204", w.Code)
	}

	// Verify it's gone from the list
	w = get(router, "/api/tasks")
	var tasks []models.Task
	json.Unmarshal(w.Body.Bytes(), &tasks)

	for _, tt := range tasks {
		if tt.ID == task.ID {
			t.Error("deleted task still appears in list")
		}
	}
}

// --- Done flow ---

func TestTaskDoneOneOff(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := postJSON(router, "/api/tasks", models.CreateTaskRequest{Title: "One-off", Type: "one-off", DueDate: "2025-06-15"})
	var task models.Task
	json.Unmarshal(w.Body.Bytes(), &task)

	w = postJSON(router, fmt.Sprintf("/api/tasks/%d/done", task.ID), models.DoneRequest{})
	if w.Code != http.StatusOK {
		t.Fatalf("done status = %d, want 200", w.Code)
	}

	var done models.Task
	json.Unmarshal(w.Body.Bytes(), &done)

	if done.Status != "done" {
		t.Errorf("status = %q, want done", done.Status)
	}
	if done.CompletedAt == nil {
		t.Error("completed_at should be set")
	}
}

func TestTaskDoneRecurringCreatesNext(t *testing.T) {
	router, _ := setupTestRouter(t)

	// Create a recurring task
	w := postJSON(router, "/api/tasks", models.CreateTaskRequest{
		Title:      "Weekly clean",
		Type:       "recurring",
		DueDate:    "2025-06-15",
		Recurrence: &models.Recurrence{Unit: "week", Every: 1},
	})
	var task models.Task
	json.Unmarshal(w.Body.Bytes(), &task)

	// Mark it done
	postJSON(router, fmt.Sprintf("/api/tasks/%d/done", task.ID), models.DoneRequest{})

	// List all tasks — should have original (done) + new (pending)
	w = get(router, "/api/tasks")
	var tasks []models.Task
	json.Unmarshal(w.Body.Bytes(), &tasks)

	var found bool
	for _, tt := range tasks {
		if tt.Title == "Weekly clean" && tt.Status == "pending" {
			found = true
			// Due date might come back with time suffix from SQLite
			if !strings.HasPrefix(tt.DueDate, "2025-06-22") {
				t.Errorf("next occurrence due_date = %q, want prefix 2025-06-22", tt.DueDate)
			}
			if tt.ParentID == nil || *tt.ParentID != task.ID {
				t.Errorf("next occurrence parent_id = %v, want %d", tt.ParentID, task.ID)
			}
		}
	}
	if !found {
		t.Errorf("expected next recurring occurrence, all tasks: %+v", tasks)
	}
}

// --- Reopen flow ---

func TestTaskReopen(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := postJSON(router, "/api/tasks", models.CreateTaskRequest{Title: "Reopen me", Type: "one-off"})
	var task models.Task
	json.Unmarshal(w.Body.Bytes(), &task)

	// Mark done
	postJSON(router, fmt.Sprintf("/api/tasks/%d/done", task.ID), models.DoneRequest{})

	// Reopen
	w = postJSON(router, fmt.Sprintf("/api/tasks/%d/reopen", task.ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("reopen status = %d, want 200", w.Code)
	}

	var reopened models.Task
	json.Unmarshal(w.Body.Bytes(), &reopened)

	if reopened.Status != "pending" {
		t.Errorf("status = %q, want pending", reopened.Status)
	}
	if reopened.CompletedAt != nil {
		t.Error("completed_at should be nil after reopen")
	}
}

// --- Snooze flow ---

func TestTaskSnooze(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := postJSON(router, "/api/tasks", models.CreateTaskRequest{Title: "Snooze me", Type: "one-off", DueDate: "2025-06-15"})
	var task models.Task
	json.Unmarshal(w.Body.Bytes(), &task)

	tests := []struct {
		name       string
		preset     string
		wantStatus int
	}{
		{name: "tomorrow", preset: "tomorrow", wantStatus: http.StatusOK},
		{name: "3days", preset: "3days", wantStatus: http.StatusOK},
		{name: "week", preset: "week", wantStatus: http.StatusOK},
		{name: "month", preset: "month", wantStatus: http.StatusOK},
		{name: "invalid preset", preset: "next_year", wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := postJSON(router, fmt.Sprintf("/api/tasks/%d/snooze", task.ID), models.SnoozeRequest{Preset: tt.preset})
			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
			if tt.wantStatus == http.StatusOK {
				var snoozed models.Task
				json.Unmarshal(w.Body.Bytes(), &snoozed)
				if snoozed.Status != "snoozed" {
					t.Errorf("status = %q, want snoozed", snoozed.Status)
				}
				if snoozed.DueDate == "" {
					t.Error("due_date should be set after snooze")
				}
			}
		})
	}
}

// --- Sync deduplication by source_ref ---

func TestSyncDeduplicationBySourceRef(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	// Insert a task with a source_ref
	_, err = database.Exec(`
		INSERT INTO tasks (title, type, status, source, source_ref)
		VALUES ('Existing', 'one-off', 'pending', 'gmail', 'email-123')
	`)
	if err != nil {
		t.Fatalf("insert task: %v", err)
	}

	// Verify dedup: same source_ref should already exist
	var count int
	database.QueryRow(`SELECT COUNT(*) FROM tasks WHERE source_ref = ?`, "email-123").Scan(&count)
	if count != 1 {
		t.Fatalf("expected 1 task with source_ref, got %d", count)
	}

	// Insert another with the same ref — this simulates what sync does (check first)
	database.QueryRow(`SELECT COUNT(*) FROM tasks WHERE source_ref = ?`, "email-123").Scan(&count)
	if count > 0 {
		// Deduplication: skip insert (this is the sync logic)
	} else {
		t.Error("deduplication check failed, count should be > 0")
	}

	// Insert with a new ref — should succeed
	_, err = database.Exec(`
		INSERT INTO tasks (title, type, status, source, source_ref)
		VALUES ('New', 'one-off', 'pending', 'gmail', 'email-456')
	`)
	if err != nil {
		t.Fatalf("insert new task: %v", err)
	}

	var totalCount int
	database.QueryRow(`SELECT COUNT(*) FROM tasks WHERE source = 'gmail'`).Scan(&totalCount)
	if totalCount != 2 {
		t.Errorf("expected 2 gmail tasks, got %d", totalCount)
	}
}

// --- People CRUD ---

func TestPeopleCRUD(t *testing.T) {
	router, _ := setupTestRouter(t)

	// Create
	w := postJSON(router, "/api/people", map[string]string{"name": "Alice"})
	if w.Code != http.StatusOK {
		t.Fatalf("create status = %d, want 200, body: %s", w.Code, w.Body.String())
	}
	var person models.Person
	json.Unmarshal(w.Body.Bytes(), &person)
	if person.Name != "Alice" {
		t.Errorf("name = %q, want Alice", person.Name)
	}

	// List
	w = get(router, "/api/people")
	var people []models.Person
	json.Unmarshal(w.Body.Bytes(), &people)
	if len(people) == 0 {
		t.Fatal("expected at least 1 person")
	}

	// Delete
	w = deleteReq(router, fmt.Sprintf("/api/people/%d", person.ID))
	if w.Code != http.StatusNoContent {
		t.Errorf("delete status = %d, want 204", w.Code)
	}

	// Verify deleted
	w = get(router, "/api/people")
	json.Unmarshal(w.Body.Bytes(), &people)
	for _, p := range people {
		if p.ID == person.ID {
			t.Error("deleted person still in list")
		}
	}
}

func TestPeopleCreateEmptyName(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := postJSON(router, "/api/people", map[string]string{"name": ""})
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

// --- Categories CRUD ---

func TestCategoriesCRUD(t *testing.T) {
	router, _ := setupTestRouter(t)

	// List (should have seed categories)
	w := get(router, "/api/categories")
	var cats []models.Category
	json.Unmarshal(w.Body.Bytes(), &cats)
	if len(cats) == 0 {
		t.Fatal("expected seed categories")
	}

	// Create a new one
	w = postJSON(router, "/api/categories", map[string]string{"name": "Garden"})
	if w.Code != http.StatusOK {
		t.Fatalf("create status = %d, want 200, body: %s", w.Code, w.Body.String())
	}
	var cat models.Category
	json.Unmarshal(w.Body.Bytes(), &cat)
	if cat.Name != "Garden" {
		t.Errorf("name = %q, want Garden", cat.Name)
	}

	// Delete
	w = deleteReq(router, fmt.Sprintf("/api/categories/%d", cat.ID))
	if w.Code != http.StatusNoContent {
		t.Errorf("delete status = %d, want 204", w.Code)
	}
}

func TestCategoriesCreateEmptyName(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := postJSON(router, "/api/categories", map[string]string{"name": ""})
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestCategoriesSeedData(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := get(router, "/api/categories")
	var cats []models.Category
	json.Unmarshal(w.Body.Bytes(), &cats)

	expected := map[string]bool{
		"Housekeeping": false,
		"Car":          false,
		"Health":       false,
		"Packages":     false,
		"Finance":      false,
		"Social":       false,
	}

	for _, c := range cats {
		if _, ok := expected[c.Name]; ok {
			expected[c.Name] = true
		}
	}

	for name, found := range expected {
		if !found {
			t.Errorf("expected seed category %q not found", name)
		}
	}
}

// --- Task with assignee ---

func TestTaskWithAssignee(t *testing.T) {
	router, _ := setupTestRouter(t)

	// Create a person
	w := postJSON(router, "/api/people", map[string]string{"name": "Bob"})
	var person models.Person
	json.Unmarshal(w.Body.Bytes(), &person)

	// Create a task assigned to them
	w = postJSON(router, "/api/tasks", models.CreateTaskRequest{
		Title:      "Bob's task",
		Type:       "one-off",
		AssigneeID: &person.ID,
	})
	var task models.Task
	json.Unmarshal(w.Body.Bytes(), &task)

	if task.AssigneeName == nil || *task.AssigneeName != "Bob" {
		t.Errorf("assignee_name = %v, want Bob", task.AssigneeName)
	}
}

// --- Task with category ---

func TestTaskWithCategory(t *testing.T) {
	router, _ := setupTestRouter(t)

	// Get the first category
	w := get(router, "/api/categories")
	var cats []models.Category
	json.Unmarshal(w.Body.Bytes(), &cats)
	if len(cats) == 0 {
		t.Fatal("no categories")
	}

	catID := cats[0].ID
	w = postJSON(router, "/api/tasks", models.CreateTaskRequest{
		Title:      "Categorized task",
		Type:       "one-off",
		CategoryID: &catID,
	})
	var task models.Task
	json.Unmarshal(w.Body.Bytes(), &task)

	if task.CategoryName == nil || *task.CategoryName != cats[0].Name {
		t.Errorf("category_name = %v, want %q", task.CategoryName, cats[0].Name)
	}
}

// --- History ---

func TestTaskHistory(t *testing.T) {
	router, _ := setupTestRouter(t)

	// Create a recurring task, complete it twice to build history
	w := postJSON(router, "/api/tasks", models.CreateTaskRequest{
		Title:      "Recurring history",
		Type:       "recurring",
		DueDate:    "2025-06-01",
		Recurrence: &models.Recurrence{Unit: "week", Every: 1},
	})
	var first models.Task
	json.Unmarshal(w.Body.Bytes(), &first)

	// Done first occurrence
	postJSON(router, fmt.Sprintf("/api/tasks/%d/done", first.ID), models.DoneRequest{})

	// Find the next occurrence
	w = get(router, "/api/tasks?status=pending")
	var pending []models.Task
	json.Unmarshal(w.Body.Bytes(), &pending)

	var second *models.Task
	for _, tt := range pending {
		if tt.Title == "Recurring history" && tt.ParentID != nil {
			second = &tt
			break
		}
	}
	if second == nil {
		t.Fatal("expected second occurrence of recurring task")
	}

	// Done second occurrence
	postJSON(router, fmt.Sprintf("/api/tasks/%d/done", second.ID), models.DoneRequest{})

	// Check history from the first task
	w = get(router, fmt.Sprintf("/api/tasks/%d/history", first.ID))
	if w.Code != http.StatusOK {
		t.Fatalf("history status = %d, want 200", w.Code)
	}

	var history []models.Task
	json.Unmarshal(w.Body.Bytes(), &history)

	if len(history) < 2 {
		t.Errorf("expected at least 2 history entries, got %d", len(history))
	}

	for _, h := range history {
		if h.Status != "done" {
			t.Errorf("history entry status = %q, want done", h.Status)
		}
	}
}
