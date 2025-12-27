package main

import (
	"log"
	"net/http"
	"users/database"
	"users/router"
)

func main() {

	database.InitDB()
	log.Default().Println("main - Init DB")

	r := router.Route()
	log.Default().Println("main - Init Route")

	port := ":8080"
	log.Printf("Server starting on %s", port)

	if err := http.ListenAndServe(port, r); err != nil {
		log.Fatal(err)
	}
}
