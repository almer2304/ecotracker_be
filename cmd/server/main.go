package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ecotracker/backend/internal/config"
	"github.com/ecotracker/backend/internal/domain"
	"github.com/ecotracker/backend/internal/handler"
	"github.com/ecotracker/backend/internal/middleware"
	"github.com/ecotracker/backend/internal/repository"
	"github.com/ecotracker/backend/internal/service"
	"github.com/ecotracker/backend/internal/utils"
	ws "github.com/ecotracker/backend/internal/websocket"
	"github.com/ecotracker/backend/internal/worker"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	// 1. LOGGING
	logrus.SetFormatter(&logrus.JSONFormatter{TimestampFormat: "2006-01-02 15:04:05"})
	logrus.SetLevel(logrus.InfoLevel)
	gin.SetMode(gin.ReleaseMode)

	// 2. CONFIG
	cfg, err := config.Load()
	if err != nil {
		logrus.WithError(err).Fatal("Gagal memuat konfigurasi")
	}
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	logrus.WithField("env", cfg.App.Env).Info("EcoTracker V2.0 Backend dimulai")

	// 3. DATABASE
	db, err := config.NewDatabase(&cfg.DB)
	if err != nil {
		logrus.WithError(err).Fatal("Gagal terhubung ke database")
	}
	defer db.Close()

	// 4. REDIS
	redisClient, _ := config.NewRedis(&cfg.Redis)
	if redisClient != nil {
		defer redisClient.Close()
	}

	// 5. UTILITIES
	jwtManager := utils.NewJWTManager(cfg.JWT.Secret, cfg.JWT.AccessExpiry, cfg.JWT.RefreshExpiry)

	var storageClient *utils.StorageClient
	if cfg.Supabase.URL != "" && cfg.Supabase.Key != "" {
		storageClient = utils.NewStorageClient(
			cfg.Supabase.URL, cfg.Supabase.Key,
			cfg.Supabase.BucketPickups, cfg.Supabase.BucketReports, cfg.Supabase.BucketAvatars,
		)
	}

	// 6. WEBSOCKET HUB
	wsHub := ws.NewHub()
	go wsHub.Run()
	wsNotifier := ws.NewNotifier(wsHub)
	wsHandler := ws.NewHandler(wsHub)
	logrus.Info("WebSocket Hub dimulai")

	// 7. REPOSITORIES
	authRepo := repository.NewAuthRepository(db)
	pickupRepo := repository.NewPickupRepository(db)
	collectorRepo := repository.NewCollectorRepository(db)
	badgeRepo := repository.NewBadgeRepository(db)
	reportRepo := repository.NewReportRepository(db)
	feedbackRepo := repository.NewFeedbackRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	pointLogRepo := repository.NewPointLogRepository(db)

	// 8. SERVICES
	badgeService := service.NewBadgeService(badgeRepo)

	// Assignment service dengan WebSocket notifier
	assignmentService := service.NewAssignmentService(
		pickupRepo, collectorRepo, authRepo,
		db, cfg.Worker.AssignmentTimeout, wsNotifier,
	)

	pickupService := service.NewPickupService(pickupRepo, assignmentService, storageClient)

	collectorService := service.NewCollectorService(
		authRepo, pickupRepo, categoryRepo, pointLogRepo, badgeService, db,
	)
	// Inject WebSocket notifier ke collector service
	//ini masih bermasalah barisnya
	collectorService.SetNotifier(wsNotifier)

	authService := service.NewAuthService(authRepo, jwtManager, cfg.Bcrypt.Cost)
	reportService := service.NewReportService(reportRepo, authRepo, badgeService, storageClient)
	feedbackService := service.NewFeedbackService(feedbackRepo, pickupRepo)
	adminService := service.NewAdminService(authRepo, pickupRepo, reportRepo, feedbackRepo, collectorRepo, cfg.Bcrypt.Cost)

	// 9. HANDLERS
	authHandler := handler.NewAuthHandler(authService, cfg.App.AdminSecret)
	pickupHandler := handler.NewPickupHandler(pickupService)
	collectorHandler := handler.NewCollectorHandler(collectorService)
	badgeHandler := handler.NewBadgeHandler(badgeService)
	reportHandler := handler.NewReportHandler(reportService)
	feedbackHandler := handler.NewFeedbackHandler(feedbackService)
	adminHandler := handler.NewAdminHandler(adminService)
	categoryHandler := handler.NewCategoryHandler(categoryRepo)

	// 10. ROUTER
	router := gin.New()
	router.Use(middleware.Recovery())
	router.Use(middleware.Logger())
	router.Use(middleware.CORS(cfg.CORS.AllowedOrigins))
	router.Use(middleware.RateLimiter(redisClient, 100, time.Minute))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		stats := wsNotifier.GetStats()
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "EcoTracker V2.0",
			"time":    time.Now().Format(time.RFC3339),
			"ws":      stats,
		})
	})

	// WebSocket endpoint (butuh auth)
	router.GET("/ws", middleware.AuthMiddleware(jwtManager), wsHandler.ServeWS)

	// API v1
	v1 := router.Group("/api/v1")
	setupRoutes(v1, authHandler, pickupHandler, collectorHandler, badgeHandler,
		reportHandler, feedbackHandler, adminHandler, categoryHandler, jwtManager)

	// 11. WORKERS
	assignmentWorker := worker.NewAssignmentWorker(assignmentService, pickupRepo, cfg.Worker.TimeoutCheckInterval)
	assignmentWorker.Start()
	defer assignmentWorker.Stop()

	// 12. START SERVER
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.App.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logrus.WithField("port", cfg.App.Port).Info("Server berjalan")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.WithError(err).Fatal("Server gagal berjalan")
		}
	}()

	// 13. GRACEFUL SHUTDOWN
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logrus.Info("Mematikan server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	logrus.Info("Server berhenti")
}

func setupRoutes(
	v1 *gin.RouterGroup,
	authHandler *handler.AuthHandler,
	pickupHandler *handler.PickupHandler,
	collectorHandler *handler.CollectorHandler,
	badgeHandler *handler.BadgeHandler,
	reportHandler *handler.ReportHandler,
	feedbackHandler *handler.FeedbackHandler,
	adminHandler *handler.AdminHandler,
	categoryHandler *handler.CategoryHandler,
	jwtManager *utils.JWTManager,
) {
	auth := middleware.AuthMiddleware(jwtManager)
	requireUser := middleware.RequireRole(domain.RoleUser)
	requireCollector := middleware.RequireRole(domain.RoleCollector)
	requireAdmin := middleware.RequireRole(domain.RoleAdmin)

	// Auth
	authGroup := v1.Group("/auth")
	{
		authGroup.POST("/register", authHandler.Register)
		authGroup.POST("/login", authHandler.Login)
		authGroup.POST("/refresh", authHandler.RefreshToken)
		authGroup.GET("/profile", auth, authHandler.GetProfile)
		authGroup.POST("/register-admin", authHandler.RegisterAdmin)
		authGroup.POST("/register-collector", authHandler.RegisterCollector)
	}

	// Categories (public)
	v1.GET("/categories", categoryHandler.GetAllCategories)

	// Pickups
	pickupGroup := v1.Group("/pickups", auth)
	{
		pickupGroup.POST("", requireUser, pickupHandler.CreatePickup)
		pickupGroup.GET("/my", pickupHandler.GetMyPickups)
		pickupGroup.GET("/:id", pickupHandler.GetPickupDetail)
	}

	// Collector
	collectorGroup := v1.Group("/collector", auth, requireCollector)
	{
		collectorGroup.PUT("/status", collectorHandler.UpdateStatus)
		collectorGroup.PUT("/location", collectorHandler.UpdateLocation)
		collectorGroup.GET("/assigned", collectorHandler.GetAssignedPickup)
		collectorGroup.POST("/pickups/:id/accept", collectorHandler.AcceptPickup)
		collectorGroup.POST("/pickups/:id/start", collectorHandler.StartPickup)
		collectorGroup.POST("/pickups/:id/arrive", collectorHandler.ArriveAtPickup)
		collectorGroup.POST("/pickups/:id/complete", collectorHandler.CompletePickup)
		collectorGroup.GET("/history", collectorHandler.GetHistory)
	}

	// Badges
	badgeGroup := v1.Group("/badges", auth)
	{
		badgeGroup.GET("", badgeHandler.GetAllBadges)
		badgeGroup.GET("/my", badgeHandler.GetMyBadges)
	}

	// Reports
	reportGroup := v1.Group("/reports", auth)
	{
		reportGroup.POST("", requireUser, reportHandler.CreateReport)
		reportGroup.GET("/my", reportHandler.GetMyReports)
		reportGroup.GET("/:id", reportHandler.GetReportDetail)
	}

	// Feedback
	feedbackGroup := v1.Group("/feedback", auth)
	{
		feedbackGroup.POST("", requireUser, feedbackHandler.CreateFeedback)
		feedbackGroup.GET("/my", feedbackHandler.GetMyFeedback)
	}

	// Admin
	adminGroup := v1.Group("/admin", auth, requireAdmin)
	{
		adminGroup.GET("/dashboard", adminHandler.GetDashboard)
		adminGroup.GET("/collectors", adminHandler.ListCollectors)
		adminGroup.POST("/collectors", adminHandler.CreateCollector)
		adminGroup.DELETE("/collectors/:id", adminHandler.DeleteCollector)
		adminGroup.GET("/pickups", adminHandler.ListPickups)
		adminGroup.GET("/reports", adminHandler.ListReports)
		adminGroup.PUT("/reports/:id", adminHandler.UpdateReport)
		adminGroup.GET("/feedback", adminHandler.ListFeedback)
		adminGroup.PUT("/feedback/:id/respond", adminHandler.RespondToFeedback)
	}
}
