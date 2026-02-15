package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/dotechhq/zenith/services/auth/internal/handlers"
	"github.com/dotechhq/zenith/services/auth/internal/storage"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "zenith-auth-dev-secret"
	}
	issuer := os.Getenv("ISSUER")
	if issuer == "" {
		issuer = "https://auth.zenith.dev"
	}

	store := storage.NewMemoryStore()

	app := fiber.New(fiber.Config{
		AppName:      "Zenith Auth",
		ServerHeader: "Zenith-Auth",
	})

	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New())

	oidcHandler := handlers.NewOIDCHandler(store, jwtSecret, issuer)
	realmHandler := handlers.NewRealmHandler(store)

	// Health
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "healthy"})
	})

	// Admin API - realm management
	admin := app.Group("/admin")
	admin.Post("/realms", realmHandler.Create)
	admin.Get("/realms", realmHandler.List)
	admin.Get("/realms/:realm", realmHandler.Get)
	admin.Delete("/realms/:realm", realmHandler.Delete)
	admin.Post("/realms/:realm/clients", realmHandler.CreateClient)
	admin.Get("/realms/:realm/clients", realmHandler.ListClients)
	admin.Get("/realms/:realm/users", realmHandler.ListUsers)
	admin.Get("/realms/:realm/users/:userId", realmHandler.GetUser)
	admin.Delete("/realms/:realm/users/:userId", realmHandler.DeleteUser)

	// OIDC endpoints (per realm)
	realms := app.Group("/realms/:realm")
	realms.Get("/.well-known/openid-configuration", oidcHandler.Discovery)
	realms.Post("/protocol/openid-connect/token", oidcHandler.Token)
	realms.Post("/register", oidcHandler.Register)

	go func() {
		addr := fmt.Sprintf(":%s", port)
		log.Printf("Zenith Auth starting on %s", addr)
		if err := app.Listen(addr); err != nil {
			log.Fatalf("Failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")
	app.Shutdown()
}
