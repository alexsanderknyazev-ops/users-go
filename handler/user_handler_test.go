package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
	"users/database"
	"users/users"
)

func testDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&users.Users{}); err != nil {
		t.Fatal(err)
	}
	prev := database.DB
	database.DB = db
	return func() { database.DB = prev }
}

func TestGetAllUsers_HTTP(t *testing.T) {
	defer testDB(t)()
	r := chi.NewRouter()
	r.Get("/users/", GetAllUsers)
	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/users/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestGetUserById_HTTP(t *testing.T) {
	defer testDB(t)()
	db := database.DB
	em := "h@t.t"
	_ = db.Create(&users.Users{NickName: "httpu", Email: &em, Phone: "+900", Rate: 1})
	var u users.Users
	db.First(&u)

	r := chi.NewRouter()
	r.Get("/{id}", GetUserById)
	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/" + strconv.FormatUint(u.Id, 10))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestGetUserById_NotFound(t *testing.T) {
	defer testDB(t)()
	r := chi.NewRouter()
	r.Get("/{id}", GetUserById)
	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)

	resp, _ := http.Get(ts.URL + "/999999")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404 got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestGetUserById_BadID(t *testing.T) {
	defer testDB(t)()
	r := chi.NewRouter()
	r.Get("/{id}", GetUserById)
	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)

	resp, _ := http.Get(ts.URL + "/notanumber")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestGetTopUsersByLimit_HTTP(t *testing.T) {
	defer testDB(t)()
	r := chi.NewRouter()
	r.Get("/limit/{limit}", GetTopUsersByLimit)
	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/limit/3")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestGetTopUsersByLimit_BadLimit(t *testing.T) {
	defer testDB(t)()
	r := chi.NewRouter()
	r.Get("/limit/{limit}", GetTopUsersByLimit)
	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)

	resp, _ := http.Get(ts.URL + "/limit/xx")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestGetUserByEmail_HTTP(t *testing.T) {
	defer testDB(t)()
	db := database.DB
	em := "route@mail"
	_ = db.Create(&users.Users{NickName: "em", Email: &em, Phone: "+800", Rate: 0})

	r := chi.NewRouter()
	r.Get("/email/{email}", GetUserByEmail)
	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/email/route%40mail")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestCreateUser_HTTP(t *testing.T) {
	defer testDB(t)()
	r := chi.NewRouter()
	r.Post("/", CreateUser)
	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)

	body := map[string]any{"nick_name": "api", "phone": "+700", "rate": float64(1)}
	b, _ := json.Marshal(body)
	resp, err := http.Post(ts.URL+"/", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestCreateUser_InvalidJSON(t *testing.T) {
	defer testDB(t)()
	r := chi.NewRouter()
	r.Post("/", CreateUser)
	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)

	resp, err := http.Post(ts.URL+"/", "application/json", bytes.NewReader([]byte("not-json")))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 got %d", resp.StatusCode)
	}
}
