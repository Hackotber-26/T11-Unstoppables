package api

import (
	"fmt"
	"t11-unstoppables/internal/auth"
	"t11-unstoppables/internal/config"
	"t11-unstoppables/internal/otp"
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
	sms, err := otp.NewTwilioVerify(cfg.TwilioAccountSID, cfg.TwilioAuthToken, cfg.TwilioVerifyServiceSID)
	if err != nil {
		return nil, fmt.Errorf("configure SMS verification: %w", err)
	}
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), CORSMiddleware(cfg.CORSOrigins))
	svc := services.NewCoordinator(cfg, client)
	h := NewHandler(svc, authService, sms)

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"service": "TrainSync API", "status": "running"})
	})

	v1 := r.Group("/api/v1")
	v1.GET("/health", h.Health)
	v1.GET("/stations", h.Stations)
	v1.GET("/subscriptions/plans", h.SubscriptionPlans)
	v1.GET("/public/metrics", h.PublicMetrics)
	r.GET("/metrics", h.PrometheusMetrics)
	v1.POST("/auth/register", h.RegisterAccount)
	v1.POST("/auth/login", h.Login)
	protected := v1.Group("")
	protected.Use(RequireAuth(authService))
	protected.GET("/transit/route", RequireRole("rider"), h.TransitRoute)
	protected.POST("/admin/invitations", RequireRole("admin"), h.CreateInvitation)
	protected.POST("/auth/otp/start", h.StartPhoneVerification)
	protected.POST("/auth/otp/check", h.CheckPhoneVerification)
	protected.GET("/admin/support/tickets", RequireRole("admin"), h.AdminSupportTickets)
	protected.PATCH("/admin/support/tickets/:id/status", RequireRole("admin"), h.UpdateSupportTicketStatus)
	protected.GET("/riders", RequireRole("admin"), h.Riders)
	protected.GET("/drivers", RequireRole("admin"), h.Drivers)
	protected.GET("/vehicles", RequireRole("admin"), h.Vehicles)
	protected.GET("/drivers/me", RequireRole("driver"), h.DriverMe)
	protected.PATCH("/drivers/me/availability", RequireRole("driver"), h.UpdateDriverAvailability)
	protected.POST("/drivers/me/location", RequireRole("driver"), h.DriverLocation)
	protected.POST("/navigation/route", RequireRole("driver", "rider"), h.NavigationRoute)
	protected.GET("/drivers/me/documents", RequireRole("driver"), h.DriverDocuments)
	protected.POST("/drivers/me/documents", RequireRole("driver"), h.UploadDriverDocument)
	protected.GET("/drivers/me/earnings", RequireRole("driver"), h.DriverEarnings)
	protected.GET("/drivers/me/ratings", RequireRole("driver"), h.DriverRatingSummary)
	protected.POST("/drivers/me/trips/:tripId/bay", RequireRole("driver"), h.AssignBay)
	protected.PATCH("/drivers/me/trips/:tripId/bay-arrival", RequireRole("driver"), h.MarkBayArrival)
	protected.PATCH("/admin/drivers/:userId/verification", RequireRole("admin"), h.UpdateDriverVerification)
	protected.PATCH("/admin/drivers/:userId/vehicle", RequireRole("admin"), h.AssignDriverVehicle)
	protected.GET("/admin/driver-documents", RequireRole("admin"), h.AdminDriverDocuments)
	protected.GET("/admin/driver-documents/:id/download", RequireRole("admin"), h.DownloadDriverDocument)
	protected.PATCH("/admin/driver-documents/:id/review", RequireRole("admin"), h.ReviewDriverDocument)
	protected.GET("/admin/bays", RequireRole("admin"), h.AdminBays)
	protected.POST("/admin/bays", RequireRole("admin"), h.SaveBay)
	protected.POST("/admin/subscription-plans", RequireRole("admin"), h.SaveSubscriptionPlan)
	protected.PATCH("/admin/subscription-plans/:id/active", RequireRole("admin"), h.SetSubscriptionPlanActive)
	protected.GET("/admin/subscription-plans", RequireRole("admin"), h.SubscriptionPlans)
	protected.POST("/trips", RequireRole("rider"), h.CreateTrip)
	protected.GET("/trips", h.Trips)
	protected.GET("/trips/:id", h.TripByID)
	protected.GET("/trips/:id/location", h.TripLocation)
	protected.GET("/trips/:id/pool-route", h.PoolRoutePreview)
	protected.PATCH("/trips/:id/status", h.UpdateTripStatus)
	protected.POST("/pools/match", RequireRole("rider"), h.MatchPool)
	protected.POST("/fare/quote", h.FareQuote)
	protected.POST("/simulation/run", RequireRole("admin"), h.RunSimulation)
	protected.POST("/admin/dispatch/run", RequireRole("admin"), h.DispatchPending)
	protected.GET("/metrics", RequireRole("admin"), h.Metrics)
	protected.GET("/wallet", RequireRole("rider"), h.Wallet)
	protected.POST("/trips/:id/pay", RequireRole("rider"), h.PayTrip)
	protected.POST("/trips/:id/rating", RequireRole("rider"), h.CreateTripRating)
	protected.GET("/trips/:id/rating", RequireRole("rider"), h.TripRating)
	protected.POST("/support/tickets", RequireRole("rider"), h.CreateSupportTicket)
	protected.GET("/support/tickets", RequireRole("rider"), h.SupportTickets)
	protected.GET("/subscriptions/me", RequireRole("rider"), h.CurrentSubscription)
	protected.POST("/subscriptions", RequireRole("rider"), h.Subscribe)
	protected.DELETE("/subscriptions/:id", RequireRole("rider"), h.CancelSubscription)
	protected.POST("/admin/wallets/:userId/credits", RequireRole("admin"), h.CreditWallet)
	return r, nil
}
