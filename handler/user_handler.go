package handler

import (
	"encoding/json"
	"net/http"
	"users/service"
	"users/users"

	"strconv"

	"github.com/go-chi/chi/v5"
)

func GetAllUsers(w http.ResponseWriter, r *http.Request) {

	result, err := service.GetAllUsers()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func GetUserById(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	result, err := service.GetUserById(id)

	if err != nil {
		http.Error(w, "Client not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)

}
func GetTopUsersByLimit(w http.ResponseWriter, r *http.Request) {
	limitStr := chi.URLParam(r, "limit")
	limit, err := strconv.ParseUint(limitStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid Limit", http.StatusBadRequest)
		return
	}
	result, err := service.GetTopUsers(int(limit))

	if err != nil {
		http.Error(w, "Client not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)

}
func CreateUser(w http.ResponseWriter, r *http.Request) {
	var user users.Users
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result := service.CreateUser(&user)
	if result != nil {
		http.Error(w, result.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}
