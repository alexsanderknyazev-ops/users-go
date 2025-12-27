package service

import (
	"users/database"
	"users/users"
)

func GetAllUsers() ([]users.User, error) {
	db := database.GetDB()
	if db == nil {
		return nil, nil
	}

	var users []users.User
	result := db.Find(&users)

	return users, result.Error
}

func GetUserById(id uint64) (users.User, error) {
	db := database.GetDB()

	var user users.User
	result := db.First(&user, id)
	return user, result.Error
}

func CreateUser(user *users.User) error {
	db := database.GetDB()
	if db == nil {
		return nil
	}

	result := db.Create(user)
	return result.Error
}
