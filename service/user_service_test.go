package service

import (
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"users/users"
)

var savedGetDB = getDB

func withMemoryDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&users.Users{}); err != nil {
		t.Fatal(err)
	}
	getDB = func() *gorm.DB { return db }
	t.Cleanup(func() { getDB = savedGetDB })
}

func TestGetAllUsers_NilDB(t *testing.T) {
	getDB = func() *gorm.DB { return nil }
	t.Cleanup(func() { getDB = savedGetDB })
	got, err := GetAllUsers()
	if err != nil || got != nil {
		t.Fatalf("GetAllUsers() = %v, %v; want nil, nil", got, err)
	}
}

func TestGetAllUsers_Empty(t *testing.T) {
	withMemoryDB(t)
	got, err := GetAllUsers()
	if err != nil || len(got) != 0 {
		t.Fatalf("GetAllUsers() = %v, %v", got, err)
	}
}

func TestGetAllUsers_WithRows(t *testing.T) {
	withMemoryDB(t)
	email := "a@b.c"
	u := &users.Users{NickName: "n1", Email: &email, Phone: "+1", Rate: 5}
	if err := CreateUser(u); err != nil {
		t.Fatal(err)
	}
	got, err := GetAllUsers()
	if err != nil || len(got) != 1 || got[0].NickName != "n1" {
		t.Fatalf("GetAllUsers() = %v, %v", got, err)
	}
}

func TestGetUserById(t *testing.T) {
	withMemoryDB(t)
	email := "x@y.z"
	u := &users.Users{NickName: "findme", Email: &email, Phone: "+100", Rate: 1}
	if err := CreateUser(u); err != nil {
		t.Fatal(err)
	}
	id := u.Id
	got, err := GetUserById(id)
	if err != nil || got.NickName != "findme" {
		t.Fatalf("GetUserById(%d) = %+v, %v", id, got, err)
	}
	_, err = GetUserById(99999)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("GetUserById missing) err = %v want ErrRecordNotFound", err)
	}
}

func TestCreateUser_NilDB(t *testing.T) {
	getDB = func() *gorm.DB { return nil }
	t.Cleanup(func() { getDB = savedGetDB })
	u := &users.Users{NickName: "x", Phone: "+2"}
	if err := CreateUser(u); err != nil {
		t.Fatalf("CreateUser nil db: %v", err)
	}
}

func TestCreateUser_Insert(t *testing.T) {
	withMemoryDB(t)
	email := "new@mail.com"
	u := &users.Users{NickName: "neo", Email: &email, Phone: "+3", Rate: 10}
	if err := CreateUser(u); err != nil {
		t.Fatal(err)
	}
	if u.Id == 0 {
		t.Fatal("expected Id after Create")
	}
}

func TestGetTopUsers_NilDB(t *testing.T) {
	getDB = func() *gorm.DB { return nil }
	t.Cleanup(func() { getDB = savedGetDB })
	got, err := GetTopUsers(5)
	if err != nil || got != nil {
		t.Fatalf("GetTopUsers() = %v, %v", got, err)
	}
}

func TestGetTopUsers_Order(t *testing.T) {
	withMemoryDB(t)
	e1, e2 := "l1@test", "l2@test"
	_ = CreateUser(&users.Users{NickName: "low", Email: &e1, Phone: "+10", Rate: 1})
	_ = CreateUser(&users.Users{NickName: "high", Email: &e2, Phone: "+11", Rate: 99})
	got, err := GetTopUsers(1)
	if err != nil || len(got) != 1 || got[0].NickName != "high" {
		t.Fatalf("GetTopUsers(1) = %+v, %v", got, err)
	}
}

func TestGetUserByEmail(t *testing.T) {
	withMemoryDB(t)
	email := "find@email"
	_ = CreateUser(&users.Users{NickName: "euser", Email: &email, Phone: "+20", Rate: 0})
	got, err := GetUserByEmail("find@email")
	if err != nil || got.NickName != "euser" {
		t.Fatalf("GetUserByEmail() = %+v, %v", got, err)
	}
	var empty users.Users
	got2, err := GetUserByEmail("missing@nowhere")
	if err != nil || got2 != empty {
		t.Fatalf("GetUserByEmail missing = %+v, %v", got2, err)
	}
}
