package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"housekeeping/internal/models"
)

type CategoriesHandler struct {
	db *sql.DB
}

func NewCategoriesHandler(db *sql.DB) *CategoriesHandler {
	return &CategoriesHandler{db: db}
}

func (h *CategoriesHandler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(r.Context(), `SELECT id, name FROM categories ORDER BY name`)
	if err != nil {
		respondErr(w, err, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	cats := []models.Category{}
	for rows.Next() {
		var c models.Category
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			respondErr(w, err, http.StatusInternalServerError)
			return
		}
		cats = append(cats, c)
	}
	respondJSON(w, cats)
}

func (h *CategoriesHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	res, err := h.db.ExecContext(r.Context(), `INSERT INTO categories (name) VALUES (?)`, req.Name)
	if err != nil {
		respondErr(w, err, http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()
	respondJSON(w, models.Category{ID: id, Name: req.Name})
}

func (h *CategoriesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if _, err := h.db.ExecContext(r.Context(), `DELETE FROM categories WHERE id = ?`, id); err != nil {
		respondErr(w, err, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
