package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"t11-unstoppables/internal/auth"
	"t11-unstoppables/internal/services"
	"t11-unstoppables/internal/simulation"
	"t11-unstoppables/internal/storage"
	tripdomain "t11-unstoppables/internal/trips"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Svc  *services.Coordinator
	Auth *auth.Service
}

func NewHandler(svc *services.Coordinator, authService *auth.Service) *Handler {
	return &Handler{Svc: svc, Auth: authService}
}

func (h *Handler) Health(c *gin.Context) {
	status := h.Svc.Health(c.Request.Context())
	code := http.StatusOK
	if status.Status != "ok" {
		code = http.StatusServiceUnavailable
	}
	c.JSON(code, status)
}

func (h *Handler) RegisterRider(c *gin.Context) {
	var payload struct {
		Name     string `json:"name" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Phone    string `json:"phone"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "name, valid email, and password are required"})
		return
	}
	session, err := h.Auth.RegisterRider(c.Request.Context(), payload.Name, payload.Email, payload.Phone, payload.Password)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": session})
}

func (h *Handler) Login(c *gin.Context) {
	var payload struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "valid email and password are required"})
		return
	}
	session, err := h.Auth.Login(c.Request.Context(), payload.Email, payload.Password)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": session})
}

func (h *Handler) Riders(c *gin.Context) {
	data, err := h.Svc.ListRiders(c.Request.Context())
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func (h *Handler) Drivers(c *gin.Context) {
	data, err := h.Svc.ListDrivers(c.Request.Context())
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func (h *Handler) CreateTrip(c *gin.Context) {
	var payload struct {
		OriginID       string   `json:"originId" binding:"required"`
		Destination    string   `json:"destination" binding:"required"`
		PoolMode       string   `json:"poolMode"`
		DistanceMeters *float64 `json:"distanceMeters" binding:"required"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "originId, destination, and distanceMeters are required"})
		return
	}
	if payload.PoolMode == "" {
		payload.PoolMode = "corridor"
	}
	trip, err := h.Svc.CreateTrip(c.Request.Context(), c.GetString("userID"), payload.OriginID, payload.Destination, payload.PoolMode, *payload.DistanceMeters)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": trip})
}

func (h *Handler) Trips(c *gin.Context) {
	data, err := h.Svc.ListTrips(c.Request.Context(), c.GetString("userID"), c.GetString("role"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func (h *Handler) TripByID(c *gin.Context) {
	trip, err := h.Svc.TripByID(c.Request.Context(), c.Param("id"), c.GetString("userID"), c.GetString("role"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": trip})
}

func (h *Handler) UpdateTripStatus(c *gin.Context) {
	var payload struct {
		Status tripdomain.Status `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "status is required"})
		return
	}
	trip, err := h.Svc.TransitionTrip(c.Request.Context(), c.Param("id"), c.GetString("userID"), c.GetString("role"), payload.Status)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": trip})
}

func (h *Handler) MatchPool(c *gin.Context) {
	data, err := h.Svc.MatchPool(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func (h *Handler) FareQuote(c *gin.Context) {
	var payload struct {
		TripID         string   `json:"tripId"`
		PoolMode       string   `json:"poolMode" binding:"required,oneof=corridor solo"`
		DistanceMeters *float64 `json:"distanceMeters" binding:"required"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "poolMode and distanceMeters are required"})
		return
	}
	quote, err := h.Svc.FareQuote(payload.TripID, *payload.DistanceMeters, payload.PoolMode)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": quote})
}

func (h *Handler) RunSimulation(c *gin.Context) {
	result, err := h.Svc.RunSimulation(simulation.DefaultScenario())
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

func (h *Handler) Metrics(c *gin.Context) {
	data, err := h.Svc.Metrics(c.Request.Context())
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func (h *Handler) PrometheusMetrics(c *gin.Context) {
	data, err := h.Svc.Metrics(c.Request.Context())
	if err != nil {
		c.String(http.StatusServiceUnavailable, "# metrics unavailable\n")
		return
	}
	c.Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	c.String(http.StatusOK, fmt.Sprintf(
		"# HELP trainsync_trips_total Total recorded trips\n# TYPE trainsync_trips_total counter\ntrainsync_trips_total %v\n"+
			"# HELP trainsync_trips_requested Current requested trips\n# TYPE trainsync_trips_requested gauge\ntrainsync_trips_requested %v\n"+
			"# HELP trainsync_trips_completed Completed trips\n# TYPE trainsync_trips_completed counter\ntrainsync_trips_completed %v\n"+
			"# HELP trainsync_pool_trip_rate Share of trips configured for corridor pooling\n# TYPE trainsync_pool_trip_rate gauge\ntrainsync_pool_trip_rate %v\n",
		data["tripsTotal"], data["tripsRequested"], data["tripsCompleted"], data["poolTripRate"],
	))
}

func (h *Handler) Wallet(c *gin.Context) {
	data, err := h.Svc.WalletSummary(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func (h *Handler) PayTrip(c *gin.Context) {
	data, err := h.Svc.PayTrip(c.Request.Context(), c.GetString("userID"), c.Param("id"), c.GetHeader("Idempotency-Key"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func (h *Handler) CreditWallet(c *gin.Context) {
	var payload struct {
		AmountCents int64  `json:"amountCents" binding:"required,gt=0"`
		Description string `json:"description" binding:"required,max=240"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "positive amountCents and description are required"})
		return
	}
	data, err := h.Svc.CreditWallet(c.Request.Context(), c.GetString("userID"), c.Param("userId"), c.GetHeader("Idempotency-Key"), payload.Description, payload.AmountCents)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": data})
}

func writeServiceError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	message := "request could not be completed"
	switch {
	case errors.Is(err, auth.ErrInvalidCredentials):
		status = http.StatusUnauthorized
		message = "invalid email or password"
	case errors.Is(err, auth.ErrInvalidRegistration):
		status = http.StatusBadRequest
		message = "registration details do not meet requirements"
	case errors.Is(err, storage.ErrConflict):
		status = http.StatusConflict
		message = "an account with this email already exists"
	case errors.Is(err, storage.ErrNotFound):
		status = http.StatusNotFound
		message = "resource not found"
	case errors.Is(err, storage.ErrInsufficientFunds):
		status = http.StatusPaymentRequired
		message = "wallet balance is insufficient"
	case errors.Is(err, storage.ErrIdempotencyConflict):
		status = http.StatusConflict
		message = "idempotency key was already used"
	case errors.Is(err, storage.ErrTripNotPayable):
		status = http.StatusConflict
		message = "cancelled trips cannot be charged"
	case strings.Contains(err.Error(), "poolMode") || strings.Contains(err.Error(), "distanceMeters"):
		status = http.StatusBadRequest
		message = err.Error()
	}
	c.JSON(status, gin.H{"success": false, "message": message, "timestamp": time.Now().UTC()})
}
