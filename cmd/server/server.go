package server

import (
	"ecotracker/internal/config"
	"ecotracker/internal/handler"
	"ecotracker/internal/middleware"
	"ecotracker/internal/repository"
	"ecotracker/internal/service"
	"ecotracker/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	router *gin.Engine
	cfg    *config.Config
}

func New(cfg *config.Config, db *pgxpool.Pool) *Server {
	// ─── Utilities ────────────────────────────────────────────────────────────
	jwtUtil := utils.NewJWTUtil(cfg.JWTSecret)
	storageClient := utils.NewStorageClient(cfg.SupabaseURL, cfg.SupabaseServiceRoleKey)

	// ─── Repositories ─────────────────────────────────────────────────────────
	authRepo := repository.NewAuthRepository(db)
	pickupRepo := repository.NewPickupRepository(db)
	categoryRepo := repository.NewWasteCategoryRepository(db)
	pointLogRepo := repository.NewPointLogRepository(db)
	voucherRepo := repository.NewVoucherRepository(db)

	// ─── Services ─────────────────────────────────────────────────────────────
	authService := service.NewAuthService(authRepo, jwtUtil)
	pickupService := service.NewPickupService(pickupRepo, categoryRepo, authRepo, storageClient, cfg.StorageBucket)
	voucherService := service.NewVoucherService(voucherRepo, authRepo)
	pointLogService := service.NewPointLogService(pointLogRepo)
	categoryService := service.NewWasteCategoryService(categoryRepo)

	// ─── Handlers ─────────────────────────────────────────────────────────────
	authHandler := handler.NewAuthHandler(authService)
	pickupHandler := handler.NewPickupHandler(pickupService)
	voucherHandler := handler.NewVoucherHandler(voucherService)
	pointLogHandler := handler.NewPointLogHandler(pointLogService)
	categoryHandler := handler.NewWasteCategoryHandler(categoryService)

	// ─── Router ───────────────────────────────────────────────────────────────
	router := gin.Default()

	// Increase max multipart memory to 10 MB
	router.MaxMultipartMemory = 10 << 20

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "ecotracker"})
	})

	v1 := router.Group("/api/v1")

	// ─── Public routes ────────────────────────────────────────────────────────
	auth := v1.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
	}

	// ─── Protected routes (all roles) ─────────────────────────────────────────
	protected := v1.Group("/")
	protected.Use(middleware.AuthMiddleware(jwtUtil))
	{
		// Profile
		protected.GET("auth/profile", authHandler.GetProfile)

		// Waste categories (read-only, all roles)
		protected.GET("categories", categoryHandler.GetCategories)

		// Point logs
		protected.GET("points/logs", pointLogHandler.GetMyPointLogs)

		// Vouchers
		protected.GET("vouchers", voucherHandler.ListVouchers)
		protected.GET("vouchers/my", voucherHandler.GetMyVouchers)

		// Get pickup detail (both user and collector can access)
		protected.GET("pickups/:id", pickupHandler.GetPickupDetail)
	}

	// ─── User-only routes ─────────────────────────────────────────────────────
	userRoutes := v1.Group("/")
	userRoutes.Use(middleware.AuthMiddleware(jwtUtil))
	userRoutes.Use(middleware.RequireRole("user"))
	{
		userRoutes.POST("pickups", pickupHandler.CreatePickup)
		userRoutes.GET("pickups/my", pickupHandler.GetMyPickups)
		userRoutes.POST("vouchers/:id/claim", voucherHandler.ClaimVoucher)
	}

	// ─── Collector-only routes ────────────────────────────────────────────────
	collectorRoutes := v1.Group("/collector")
	collectorRoutes.Use(middleware.AuthMiddleware(jwtUtil))
	collectorRoutes.Use(middleware.RequireRole("collector"))
	{
		collectorRoutes.GET("pickups/pending", pickupHandler.GetPendingPickups)
		collectorRoutes.GET("pickups/my-tasks", pickupHandler.GetMyTasks)
		collectorRoutes.POST("pickups/:id/take", pickupHandler.TakeTask)
		collectorRoutes.POST("pickups/:id/complete", pickupHandler.CompleteTask)
	}

	return &Server{router: router, cfg: cfg}
}

func (s *Server) Run() error {
	return s.router.Run(":" + s.cfg.Port)
}
