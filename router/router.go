package router

import (
	"users/handler"

	"github.com/go-chi/chi/v5"
)

func Route() *chi.Mux {

	r := chi.NewRouter()

	r.Route("/users", func(r chi.Router) {
		r.Get("/", handler.GetAllUsers)
		r.Post("/", handler.CreateUser)
		r.Get("/{id}", handler.GetUserById)
		// r.Put("/{id}", handler.UpdateClient)
		// r.Delete("/{id}", handler.DeleteClient)
	})
	return r
}
