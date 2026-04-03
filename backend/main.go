package main

import (
	"bufio"
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"housekeeping/internal/db"
	"housekeeping/internal/handlers"
	"housekeeping/internal/whatsapp"
)

func loadEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return // .env is optional
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		// Strip inline comments (e.g. "value # comment")
		if idx := strings.Index(val, " #"); idx != -1 {
			val = val[:idx]
		}
		if os.Getenv(strings.TrimSpace(key)) == "" {
			os.Setenv(strings.TrimSpace(key), strings.TrimSpace(val))
		}
	}
}

func main() {
	loadEnv(".env")
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "housekeeping.db"
	}

	database, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer database.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	waClient := whatsapp.NewClient(database)
	go waClient.AutoReconnect(ctx)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"http://localhost:5173", "http://localhost:3000"},
		AllowedMethods: []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type"},
	}))

	tasks := handlers.NewTasksHandler(database)
	people := handlers.NewPeopleHandler(database)
	categories := handlers.NewCategoriesHandler(database)
	integrations := handlers.NewIntegrationsHandler(database, waClient)
	sync := handlers.NewSyncHandler(database, waClient)

	r.Route("/api", func(r chi.Router) {
		r.Route("/tasks", func(r chi.Router) {
			r.Get("/", tasks.List)
			r.Post("/", tasks.Create)
			r.Patch("/{id}", tasks.Update)
			r.Post("/{id}/done", tasks.Done)
			r.Post("/{id}/reopen", tasks.Reopen)
			r.Post("/{id}/snooze", tasks.Snooze)
			r.Delete("/{id}", tasks.Delete)
			r.Get("/{id}/history", tasks.History)
		})

		r.Route("/people", func(r chi.Router) {
			r.Get("/", people.List)
			r.Post("/", people.Create)
			r.Delete("/{id}", people.Delete)
		})

		r.Route("/categories", func(r chi.Router) {
			r.Get("/", categories.List)
			r.Post("/", categories.Create)
			r.Delete("/{id}", categories.Delete)
		})

		r.Route("/integrations/gmail", func(r chi.Router) {
			r.Get("/auth", integrations.GmailAuth)
			r.Get("/callback", integrations.GmailCallback)
			r.Get("/status", integrations.GmailStatus)
			r.Delete("/", integrations.GmailDisconnect)
		})

		r.Route("/integrations/whatsapp", func(r chi.Router) {
			r.Post("/connect", integrations.WhatsAppConnect)
			r.Get("/status", integrations.WhatsAppStatus)
			r.Get("/qr", integrations.WhatsAppQR)
			r.Post("/disconnect", integrations.WhatsAppDisconnect)
			r.Get("/contacts", integrations.WhatsAppListContacts)
			r.Post("/contacts", integrations.WhatsAppAddContact)
			r.Delete("/contacts/{id}", integrations.WhatsAppRemoveContact)
		})

		r.Route("/sync", func(r chi.Router) {
			r.Post("/gmail", sync.SyncGmail)
			r.Post("/whatsapp", sync.SyncWhatsApp)
			r.Get("/history", sync.SyncHistory)
		})
	})

	addr := ":8080"
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
