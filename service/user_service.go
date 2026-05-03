package service

import (
	"users/database"
	"users/users"

	"gorm.io/gorm"
)

// getDB подменяется в тестах; в проде — database.GetDB.
var getDB = func() *gorm.DB {
	return database.GetDB()
}

func GetAllUsers() ([]users.Users, error) {
	db := getDB()
	if db == nil {
		return nil, nil
	}

	var users []users.Users
	result := db.Find(&users)

	return users, result.Error
}
func GetUserById(id uint64) (users.Users, error) {
	db := getDB()

	var user users.Users
	result := db.First(&user, id)
	return user, result.Error
}

func CreateUser(user *users.Users) error {
	db := getDB()
	if db == nil {
		return nil
	}

	result := db.Create(user)
	return result.Error
}

func GetTopUsers(limit int) ([]users.Users, error) {
	db := getDB()
	if db == nil {
		return nil, nil
	}

	var users []users.Users
	result := db.Order("rate DESC").Limit(limit).Find(&users)
	return users, result.Error
}

func GetUserByEmail(email string) (users.Users, error) {
	db := getDB()

	var user users.Users
	res := db.Where("email = ?", email).Find(&user)
	return user, res.Error
}
