package router

import (
	"net/http"
	"time"
	"users/handler"

	"github.com/go-chi/chi/v5"
)

func Route() *chi.Mux {

	r := chi.NewRouter()

	r.Route("/users", func(r chi.Router) {
		r.Get("/", handler.GetAllUsers)
		r.Post("/", handler.CreateUser)
		r.Get("/{id}", handler.GetUserById)
		r.Get("/limit/{limit}", handler.GetTopUsersByLimit)
		r.Get("/health", healthCheck)
		r.Get("/email/{email}", handler.GetUserByEmail)
		// r.Put("/{id}", handler.UpdateClient)
		// r.Delete("/{id}", handler.DeleteClient)
	})
	return r
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "ok", "timestamp": "` + time.Now().Format(time.RFC3339) + `"}`))
}
