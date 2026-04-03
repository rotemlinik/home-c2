package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"housekeeping/internal/models"
)

type PeopleHandler struct {
	db *sql.DB
}

func NewPeopleHandler(db *sql.DB) *PeopleHandler {
	return &PeopleHandler{db: db}
}

func (h *PeopleHandler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(r.Context(), `SELECT id, name, created_at FROM people ORDER BY name`)
	if err != nil {
		respondErr(w, err, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	people := []models.Person{}
	for rows.Next() {
		var p models.Person
		if err := rows.Scan(&p.ID, &p.Name, &p.CreatedAt); err != nil {
			respondErr(w, err, http.StatusInternalServerError)
			return
		}
		people = append(people, p)
	}
	respondJSON(w, people)
}

func (h *PeopleHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	res, err := h.db.ExecContext(r.Context(), `INSERT INTO people (name) VALUES (?)`, req.Name)
	if err != nil {
		respondErr(w, err, http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()

	var p models.Person
	h.db.QueryRowContext(r.Context(), `SELECT id, name, created_at FROM people WHERE id = ?`, id).
		Scan(&p.ID, &p.Name, &p.CreatedAt)
	respondJSON(w, p)
}

func (h *PeopleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if _, err := h.db.ExecContext(r.Context(), `DELETE FROM people WHERE id = ?`, id); err != nil {
		respondErr(w, err, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
