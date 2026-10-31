package api

import (
	"fmt"
	"t11-unstoppables/internal/auth"
	"t11-unstoppables/internal/config"
	"t11-unstoppables/internal/services"
	"t11-unstoppables/internal/storage"

	"github.com/gin-gonic/gin"
)

func NewRouter(cfg *config.Config, client *storage.Client) (*gin.Engine, error) {
	if cfg == nil || client == nil || client.Postgres == nil {
		return nil, fmt.Errorf("config and postgres storage are required")
	}
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	authService, err := auth.NewService(client.Postgres, cfg.JWTSecret)
	if err != nil {
		return nil, fmt.Errorf("configure authentication: %w", err)
	}
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), CORSMiddleware(cfg.CORSOrigins))
	svc := services.NewCoordinator(cfg, client)
	h := NewHandler(svc, authService)

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"service": "TrainSync API", "status": "running"})
	})

	v1 := r.Group("/api/v1")
	v1.GET("/health", h.Health)
	r.GET("/metrics", h.PrometheusMetrics)
	v1.POST("/auth/register", h.RegisterRider)
	v1.POST("/auth/login", h.Login)
	protected := v1.Group("")
	protected.Use(RequireAuth(authService))
	protected.GET("/riders", RequireRole("admin"), h.Riders)
	protected.GET("/drivers", RequireRole("admin"), h.Drivers)
	protected.POST("/trips", h.CreateTrip)
	protected.GET("/trips", h.Trips)
	protected.GET("/trips/:id", h.TripByID)
	protected.PATCH("/trips/:id/status", h.UpdateTripStatus)
	protected.POST("/pools/match", h.MatchPool)
	protected.POST("/fare/quote", h.FareQuote)
	protected.POST("/simulation/run", RequireRole("admin"), h.RunSimulation)
	protected.GET("/metrics", RequireRole("admin"), h.Metrics)
	protected.GET("/wallet", h.Wallet)
	protected.POST("/trips/:id/pay", h.PayTrip)
	protected.POST("/admin/wallets/:userId/credits", RequireRole("admin"), h.CreditWallet)
	return r, nil
}
