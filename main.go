package main

import (
	"app/models"
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type App struct {
	DB *sql.DB
}

func main() {
	app := App{}
	_ = godotenv.Load(".env")

	err := models.ConnectToDB()
	defer func() {
		if models.DB != nil {
			models.DB.Close()
		}
	}()

	if err != nil {
		log.Fatal(err)
	}
	app.DB = models.DB

	migrationCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := models.RunMigrations(migrationCtx); err != nil {
		log.Panicf("migration failed: %v", err)
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", os.Getenv("PORT")),
		Handler: app.router(),
	}

	fmt.Printf("Server runnning on http://localhost:%s", os.Getenv("PORT"))

	err = srv.ListenAndServe()
	if err != nil {
		log.Panic(err)
	}
}
