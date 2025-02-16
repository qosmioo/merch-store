package main

import (
	"context"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/qosmioo/merch-store/cfg"
	httpHandler "github.com/qosmioo/merch-store/internal/delivery/http"
	"github.com/qosmioo/merch-store/internal/repository"
	"github.com/qosmioo/merch-store/internal/usecase"
	"github.com/qosmioo/merch-store/pkg/logger"
)

func main() {
	logger := logger.InitLogger()

	config, err := cfg.LoadConfig()
	if err != nil {
		log.Fatalf("Unable to load config: %v\n", err)
	}

	databaseURL := config.GetDatabaseURL()

	dbpool, err := pgxpool.Connect(context.Background(), databaseURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer dbpool.Close()

	employeeRepo := repository.NewEmployeeRepository(dbpool, logger)
	employeeUsecase := usecase.NewEmployeeUsecase(employeeRepo, logger)

	router := mux.NewRouter()
	handler := httpHandler.NewHandler(employeeUsecase, logger)
	handler.RegisterRoutes(router)

	log.Println("Server is running on port 8080")
	http.ListenAndServe(":8080", router)
}
