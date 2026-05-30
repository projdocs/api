package main

import (
	"log"

	"github.com/projdocs/api/config"
	"github.com/projdocs/api/internal/db"
	"github.com/projdocs/api/internal/router"
)

func main() {

	log.Printf("Version: %s\n", config.Version)

	// init config
	if _, err := config.Init(); err != nil {
		log.Fatalf("config error: %s", err)
	}

	// setup database
	database, err := db.Init()
	if err != nil {
		log.Fatalf("unable to connect to database: %v", err)
	}
	defer database.Close()

	// start the API router
	r := router.New()
	log.Println("listening on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
