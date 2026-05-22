package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"archive-system/internal/config"
	"archive-system/internal/database"
	"archive-system/internal/handlers"
	"archive-system/internal/middleware"
	"archive-system/internal/repository"
	"archive-system/internal/services"
	ws "archive-system/internal/websocket"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Config error: %v", err)
	}

	db, err := database.Connect(&cfg.Database)
	if err != nil {
		log.Fatalf("❌ Database error: %v", err)
	}
	defer db.Close()

	if err := os.MkdirAll(cfg.Upload.Dir+"/transactions", 0755); err != nil {
		log.Fatalf("❌ Cannot create uploads dir: %v", err)
	}
	log.Printf("✅ Uploads directory ready: %s", cfg.Upload.Dir)

	// ── WebSocket Hub ─────────────────────────────────────────
	hub := ws.NewHub()

	// ── Dependencies ──────────────────────────────────────────
	userRepo := repository.NewUserRepository(db)
	orgRepo := repository.NewOrgRepository(db)
	txRepo := repository.NewTransactionRepository(db)

	authSvc := services.NewAuthService(userRepo, cfg)
	notifSvc := services.NewNotificationService(db, hub)
	txSvc := services.NewTransactionService(txRepo, userRepo, orgRepo, notifSvc)
	fileSvc := services.NewFileService(txRepo, cfg.Upload.Dir, db)

	authHandler := handlers.NewAuthHandler(authSvc, userRepo)
	orgHandler := handlers.NewOrgHandler(orgRepo)
	txHandler := handlers.NewTransactionHandler(txSvc, fileSvc)
	fileHandler := handlers.NewFileHandler(fileSvc)
	notifHandler := handlers.NewNotificationHandler(notifSvc)
	wsHandler := ws.NewWSHandler(hub, cfg.JWT.Secret)

	// ── Gin ───────────────────────────────────────────────────
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.Default()
	router.MaxMultipartMemory = cfg.Upload.MaxFileSizeMB << 20

	router.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := db.Pool.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "db": "down"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "db": "up"})
	})

	// WebSocket endpoint
	router.GET("/ws", wsHandler.Handle)

	// ── Routes ────────────────────────────────────────────────
	v1 := router.Group("/api/v1")
	{
		// Public
		v1.POST("/auth/login", authHandler.Login)
		v1.POST("/auth/register", authHandler.Register)
		v1.GET("/organizations", orgHandler.GetAll)
		v1.POST("/organizations", orgHandler.Create)
		v1.GET("/organizations/:code", orgHandler.GetByCode)

		// Protected
		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(authSvc))
		{
			protected.GET("/auth/me", authHandler.Me)

			// Transactions
			protected.POST("/transactions", txHandler.Create)
			protected.GET("/transactions", txHandler.GetAll)
			protected.GET("/transactions/:id", txHandler.GetByID)
			protected.PUT("/transactions/:id/accept", txHandler.Accept)
			protected.PUT("/transactions/:id/reject", txHandler.Reject)

			// Files
			protected.GET("/files/:attachment_id", fileHandler.Download)

			// Notifications
			protected.GET("/notifications", notifHandler.GetAll)
			protected.PUT("/notifications/read-all", notifHandler.MarkAllRead)
		}
	}

	// ── Server ────────────────────────────────────────────────
	srv := &http.Server{Addr: ":" + cfg.AppPort, Handler: router}

	go func() {
		log.Printf("🚀 Server running on port %s", cfg.AppPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
