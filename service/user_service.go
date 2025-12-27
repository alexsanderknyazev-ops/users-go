package service

import (
	"users/database"
	"users/users"
)

func GetAllUsers() ([]users.Users, error) {
	db := database.GetDB()
	if db == nil {
		return nil, nil
	}

	var users []users.Users
	result := db.Find(&users)

	return users, result.Error
}
func GetUserById(id uint64) (users.Users, error) {
	db := database.GetDB()

	var user users.Users
	result := db.First(&user, id)
	return user, result.Error
}

func CreateUser(user *users.Users) error {
	db := database.GetDB()
	if db == nil {
		return nil
	}

	result := db.Create(user)
	return result.Error
}

func GetTopUsers(limit int) ([]users.Users, error) {
	db := database.GetDB()
	if db == nil {
		return nil, nil
	}

	var users []users.Users
	result := db.Order("rate DESC").Limit(limit).Find(&users)
	return users, result.Error
}
