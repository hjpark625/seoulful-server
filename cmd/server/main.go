package main

import (
	"log"

	"github.com/gin-gonic/gin"
	supabase "github.com/supabase-community/supabase-go"

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

	client, err := supabase.NewClient(cfg.SupabaseURL, cfg.SupabaseKey, nil)
	if err != nil {
		log.Fatalf("failed to initialize supabase client: %v", err)
	}

	eventRepo := repository.NewEventRepository(client)
	eventService := service.NewEventService(eventRepo)
	eventHandler := handler.NewEventHandler(eventService)

	gin.SetMode(cfg.GinMode)
	router := gin.Default()
	router.Use(middleware.CORS(cfg.CORSOrigin))

	router.GET("/health", handler.HealthCheck)
	router.GET("/events", eventHandler.GetEvents)
	router.GET("/events/:id", eventHandler.GetEventByID)

	log.Printf("server starting on port %s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
