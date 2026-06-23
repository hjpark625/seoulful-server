package main

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"seoulful-server-go/internal/config"
	"seoulful-server-go/internal/handler"
	"seoulful-server-go/internal/middleware"
	"seoulful-server-go/internal/repository"
	"seoulful-server-go/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to initialize PostgreSQL connection pool: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("failed to connect to PostgreSQL: %v", err)
	}

	eventRepo := repository.NewEventRepository(pool)
	eventService := service.NewEventService(eventRepo)
	eventHandler := handler.NewEventHandler(eventService)

	gin.SetMode(cfg.GinMode)
	router := gin.Default()
	router.Use(middleware.CORS(cfg.CORSOrigin))

	router.GET("/health", handler.HealthCheck(pool))
	router.GET("/events", eventHandler.GetEvents)
	router.GET("/events/:id", eventHandler.GetEventByID)

	log.Printf("server starting on port %s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
