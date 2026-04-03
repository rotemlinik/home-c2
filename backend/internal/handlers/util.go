package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

func respondJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("encode response: %v", err)
	}
}

func respondErr(w http.ResponseWriter, err error, code int) {
	log.Printf("error %d: %v", code, err)
	http.Error(w, err.Error(), code)
}
