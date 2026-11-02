package api

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"regexp"
	"strings"
	"t11-unstoppables/internal/auth"
	"t11-unstoppables/internal/models"
	"t11-unstoppables/internal/otp"
	"t11-unstoppables/internal/routing"
	"t11-unstoppables/internal/services"
	"t11-unstoppables/internal/simulation"
	"t11-unstoppables/internal/storage"
	"t11-unstoppables/internal/transit"
	tripdomain "t11-unstoppables/internal/trips"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Svc  *services.Coordinator
	Auth *auth.Service
	SMS  *otp.TwilioVerify
}

func NewHandler(svc *services.Coordinator, authService *auth.Service, sms ...*otp.TwilioVerify) *Handler {
	handler := &Handler{Svc: svc, Auth: authService}
	if len(sms) > 0 {
		handler.SMS = sms[0]
	}
	return handler
}

var phoneE164Pattern = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)
var verificationCodePattern = regexp.MustCompile(`^[0-9]{4,10}$`)

func (h *Handler) Health(c *gin.Context) {
	status := h.Svc.Health(c.Request.Context())
	code := http.StatusOK
	if status.Status != "ok" {
		code = http.StatusServiceUnavailable
	}
	c.JSON(code, status)
}

func (h *Handler) RegisterAccount(c *gin.Context) {
	var payload struct {
		Name           string `json:"name" binding:"required"`
		Email          string `json:"email" binding:"required,email"`
		Phone          string `json:"phone"`
		Password       string `json:"password" binding:"required"`
		Role           string `json:"role"`
		InvitationCode string `json:"invitationCode"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "name, valid email, and password are required"})
		return
	}
	if payload.Role == "" {
		payload.Role = "rider"
	}
	session, err := h.Auth.Register(c.Request.Context(), payload.Name, payload.Email, payload.Phone, payload.Password, payload.Role, payload.InvitationCode)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": session})
}

func (h *Handler) CreateInvitation(c *gin.Context) {
	var payload struct {
		Role string `json:"role" binding:"required,oneof=driver admin"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "role must be driver or admin"})
		return
	}
	code, expiresAt, err := h.Auth.CreateInvitation(c.Request.Context(), payload.Role, c.GetString("userID"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": gin.H{
		"role": payload.Role, "invitationCode": code, "expiresAt": expiresAt,
	}})
}

func (h *Handler) Login(c *gin.Context) {
	var payload struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "valid email and password are required"})
		return
	}
	session, err := h.Auth.LoginAs(c.Request.Context(), payload.Email, payload.Password, payload.Role)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": session})
}

func (h *Handler) StartPhoneVerification(c *gin.Context) {
	var payload struct {
		Phone string `json:"phone" binding:"required"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil || !phoneE164Pattern.MatchString(payload.Phone) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "phone must be a valid E.164 number, for example +919876543210"})
		return
	}
	if err := h.SMS.Start(c.Request.Context(), payload.Phone); err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"success": true, "data": gin.H{"status": "pending", "phone": payload.Phone}})
}

func (h *Handler) CheckPhoneVerification(c *gin.Context) {
	var payload struct {
		Phone string `json:"phone" binding:"required"`
		Code  string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil || !phoneE164Pattern.MatchString(payload.Phone) || !verificationCodePattern.MatchString(payload.Code) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "a valid E.164 phone number and verification code are required"})
		return
	}
	if err := h.SMS.Check(c.Request.Context(), payload.Phone, payload.Code); err != nil {
		writeServiceError(c, err)
		return
	}
	if err := h.Svc.VerifyUserPhone(c.Request.Context(), c.GetString("userID"), payload.Phone); err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"status": "verified", "phone": payload.Phone}})
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

func (h *Handler) Vehicles(c *gin.Context) {
	data, err := h.Svc.ListVehicles(c.Request.Context())
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func (h *Handler) DriverMe(c *gin.Context) {
	data, err := h.Svc.DriverByUserID(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func (h *Handler) UpdateDriverAvailability(c *gin.Context) {
	var payload struct {
		Status string `json:"status" binding:"required,oneof=available offline"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "status must be available or offline"})
		return
	}
	data, err := h.Svc.UpdateDriverStatus(c.Request.Context(), c.GetString("userID"), payload.Status)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func (h *Handler) UpdateDriverVerification(c *gin.Context) {
	var payload struct {
		Verified *bool `json:"verified" binding:"required"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "verified must be true or false"})
		return
	}
	data, err := h.Svc.UpdateDriverVerification(c.Request.Context(), c.GetString("userID"), c.Param("userId"), *payload.Verified)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func (h *Handler) AssignDriverVehicle(c *gin.Context) {
	var payload struct {
		VehicleID string `json:"vehicleId" binding:"required,uuid"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "a valid vehicleId is required"})
		return
	}
	data, err := h.Svc.AssignDriverVehicle(c.Request.Context(), c.GetString("userID"), c.Param("userId"), payload.VehicleID)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func (h *Handler) Stations(c *gin.Context) {
	data, err := h.Svc.ListStations(c.Request.Context())
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func (h *Handler) PublicMetrics(c *gin.Context) {
	data, err := h.Svc.Metrics(c.Request.Context())
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func (h *Handler) TransitRoute(c *gin.Context) {
	originID := strings.TrimSpace(c.Query("originId"))
	destinationID := strings.TrimSpace(c.Query("destinationId"))
	departure := time.Now().UTC()
	if rawDeparture := strings.TrimSpace(c.Query("departureTime")); rawDeparture != "" {
		parsed, err := time.Parse(time.RFC3339, rawDeparture)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "departureTime must be an RFC3339 timestamp"})
			return
		}
		departure = parsed
	}
	route, err := h.Svc.TransitRoute(c.Request.Context(), originID, destinationID, departure)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.JSON(http.StatusOK, gin.H{"success": true, "data": route})
}

func (h *Handler) CreateTrip(c *gin.Context) {
	var payload struct {
		OriginID           string   `json:"originId" binding:"required"`
		Destination        string   `json:"destination" binding:"required"`
		PoolMode           string   `json:"poolMode"`
		PickupLatitude     *float64 `json:"pickupLatitude" binding:"required"`
		PickupLongitude    *float64 `json:"pickupLongitude" binding:"required"`
		DropoffLatitude    *float64 `json:"dropoffLatitude" binding:"required"`
		DropoffLongitude   *float64 `json:"dropoffLongitude" binding:"required"`
		PartySize          int      `json:"partySize"`
		RequiresAccessible bool     `json:"requiresAccessible"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "originId, destination, pickup and drop-off coordinates are required"})
		return
	}
	if payload.PoolMode == "" {
		payload.PoolMode = "corridor"
	}
	if payload.PartySize == 0 {
		payload.PartySize = 1
	}
	if payload.PartySize < 1 || payload.PartySize > 8 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "partySize must be between 1 and 8"})
		return
	}
	trip, err := h.Svc.CreateTrip(c.Request.Context(), c.GetString("userID"), payload.OriginID, payload.Destination, payload.PoolMode,
		*payload.PickupLatitude, *payload.PickupLongitude, *payload.DropoffLatitude, *payload.DropoffLongitude, payload.PartySize, payload.RequiresAccessible)
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
		TripID           string   `json:"tripId"`
		PoolMode         string   `json:"poolMode" binding:"required,oneof=corridor solo"`
		PickupLatitude   *float64 `json:"pickupLatitude" binding:"required"`
		PickupLongitude  *float64 `json:"pickupLongitude" binding:"required"`
		DropoffLatitude  *float64 `json:"dropoffLatitude" binding:"required"`
		DropoffLongitude *float64 `json:"dropoffLongitude" binding:"required"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "poolMode and pickup and drop-off coordinates are required"})
		return
	}
	quote, err := h.Svc.FareQuote(c.Request.Context(), c.GetString("userID"), payload.TripID,
		*payload.PickupLatitude, *payload.PickupLongitude, *payload.DropoffLatitude, *payload.DropoffLongitude, payload.PoolMode)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": quote})
}

func (h *Handler) PoolRoutePreview(c *gin.Context) {
	preview, err := h.Svc.PoolRoutePreview(c.Request.Context(), c.Param("id"), c.GetString("userID"), c.GetString("role"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": preview})
}

func (h *Handler) RunSimulation(c *gin.Context) {
	result, err := h.Svc.RunSimulation(simulation.DefaultScenario())
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

func (h *Handler) DispatchPending(c *gin.Context) {
	result, err := h.Svc.DispatchPending(c.Request.Context())
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
		AmountCents int64  `json:"amountPaise" binding:"required,gt=0"`
		Description string `json:"description" binding:"required,max=240"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "positive amountPaise and description are required"})
		return
	}
	data, err := h.Svc.CreditWallet(c.Request.Context(), c.GetString("userID"), c.Param("userId"), c.GetHeader("Idempotency-Key"), payload.Description, payload.AmountCents)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": data})
}

func (h *Handler) CreateSupportTicket(c *gin.Context) {
	var payload struct {
		Category string `json:"category" binding:"required"`
		Subject  string `json:"subject" binding:"required,max=120"`
		Message  string `json:"message" binding:"required,max=2000"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "category, subject, and a message up to 2000 characters are required"})
		return
	}
	ticket, err := h.Svc.CreateSupportTicket(c.Request.Context(), c.GetString("userID"), payload.Category, payload.Subject, payload.Message)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": ticket})
}

func (h *Handler) SupportTickets(c *gin.Context) {
	tickets, err := h.Svc.SupportTickets(c.Request.Context(), c.GetString("userID"), false)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": tickets})
}

func (h *Handler) AdminSupportTickets(c *gin.Context) {
	tickets, err := h.Svc.SupportTickets(c.Request.Context(), "", true)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": tickets})
}

func (h *Handler) UpdateSupportTicketStatus(c *gin.Context) {
	var payload struct {
		Status string `json:"status" binding:"required,oneof=open in_progress resolved"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "status must be open, in_progress, or resolved"})
		return
	}
	ticket, err := h.Svc.UpdateSupportTicketStatus(c.Request.Context(), c.GetString("userID"), c.Param("id"), payload.Status)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": ticket})
}

func (h *Handler) CreateTripRating(c *gin.Context) {
	var payload struct {
		Stars   int    `json:"stars" binding:"required,min=1,max=5"`
		Comment string `json:"comment" binding:"max=1000"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "stars must be 1 to 5 and comment must be at most 1000 characters"})
		return
	}
	rating, err := h.Svc.CreateTripRating(c.Request.Context(), c.GetString("userID"), c.Param("id"), payload.Stars, payload.Comment)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": rating})
}

func (h *Handler) TripRating(c *gin.Context) {
	rating, err := h.Svc.TripRating(c.Request.Context(), c.GetString("userID"), c.Param("id"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": rating})
}

func (h *Handler) DriverLocation(c *gin.Context) {
	var payload struct {
		TripID         string   `json:"tripId" binding:"required,uuid"`
		Latitude       *float64 `json:"latitude" binding:"required"`
		Longitude      *float64 `json:"longitude" binding:"required"`
		AccuracyMeters *float64 `json:"accuracyMeters" binding:"required"`
		SpeedMPS       *float64 `json:"speedMps"`
		HeadingDegrees *float64 `json:"headingDegrees"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "tripId, latitude, longitude, and accuracy are required"})
		return
	}
	location := models.DriverLocation{
		TripID: payload.TripID, Latitude: *payload.Latitude, Longitude: *payload.Longitude,
		AccuracyMeters: *payload.AccuracyMeters, SpeedMPS: payload.SpeedMPS, HeadingDegrees: payload.HeadingDegrees,
	}
	if err := h.Svc.SaveDriverLocation(c.Request.Context(), c.GetString("userID"), location); err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"success": true, "data": gin.H{"recordedAt": time.Now().UTC()}})
}

func (h *Handler) TripLocation(c *gin.Context) {
	location, err := h.Svc.TripLocation(c.Request.Context(), c.Param("id"), c.GetString("userID"), c.GetString("role"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": location})
}

func (h *Handler) NavigationRoute(c *gin.Context) {
	var payload struct {
		StartLatitude  *float64 `json:"startLatitude" binding:"required"`
		StartLongitude *float64 `json:"startLongitude" binding:"required"`
		EndLatitude    *float64 `json:"endLatitude" binding:"required"`
		EndLongitude   *float64 `json:"endLongitude" binding:"required"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "all start and destination coordinates are required"})
		return
	}
	route, err := h.Svc.NavigationRoute(c.Request.Context(), *payload.StartLatitude, *payload.StartLongitude, *payload.EndLatitude, *payload.EndLongitude)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": route})
}

func (h *Handler) UploadDriverDocument(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 10<<20)
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil {
		if c.Request.MultipartForm != nil {
			_ = c.Request.MultipartForm.RemoveAll()
		}
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "document must be a file no larger than 10 MB"})
		return
	}
	if c.Request.MultipartForm != nil {
		defer c.Request.MultipartForm.RemoveAll()
	}
	file, header, err := c.Request.FormFile("document")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "document file is required"})
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, (10<<20)+1))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "document could not be read"})
		return
	}
	if len(data) == 0 || len(data) > 10<<20 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "document must be between 1 byte and 10 MB"})
		return
	}
	contentType := http.DetectContentType(data)
	if contentType == "application/octet-stream" && strings.HasPrefix(string(data), "%PDF-") {
		contentType = "application/pdf"
	}
	document, err := h.Svc.SaveDriverDocument(c.Request.Context(), c.GetString("userID"), c.PostForm("documentType"), header.Filename, contentType, data)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": document})
}

func (h *Handler) DriverDocuments(c *gin.Context) {
	documents, err := h.Svc.DriverDocuments(c.Request.Context(), c.GetString("userID"), false)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": documents})
}

func (h *Handler) AdminDriverDocuments(c *gin.Context) {
	documents, err := h.Svc.DriverDocuments(c.Request.Context(), "", true)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": documents})
}

func (h *Handler) ReviewDriverDocument(c *gin.Context) {
	var payload struct {
		Status string `json:"status" binding:"required,oneof=approved rejected"`
		Note   string `json:"note" binding:"max=500"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "status must be approved or rejected and note may be up to 500 characters"})
		return
	}
	document, err := h.Svc.ReviewDriverDocument(c.Request.Context(), c.GetString("userID"), c.Param("id"), payload.Status, payload.Note)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": document})
}

func (h *Handler) DownloadDriverDocument(c *gin.Context) {
	document, data, err := h.Svc.ReadDriverDocument(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.Header("Content-Type", document.ContentType)
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": document.FileName}))
	c.Header("Cache-Control", "private, no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, document.ContentType, data)
}

func (h *Handler) DriverEarnings(c *gin.Context) {
	earnings, err := h.Svc.DriverEarnings(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": earnings})
}

func (h *Handler) DriverRatingSummary(c *gin.Context) {
	summary, err := h.Svc.DriverRatingSummary(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": summary})
}

func (h *Handler) AdminBays(c *gin.Context) {
	bays, err := h.Svc.ListBays(c.Request.Context())
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": bays})
}

func (h *Handler) SaveBay(c *gin.Context) {
	var payload models.StationBay
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "valid station, label, capacity, walk distance, and bay settings are required"})
		return
	}
	bay, err := h.Svc.SaveBay(c.Request.Context(), payload)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": bay})
}

func (h *Handler) AssignBay(c *gin.Context) {
	assignment, err := h.Svc.AssignBay(c.Request.Context(), c.GetString("userID"), c.Param("tripId"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": assignment})
}

func (h *Handler) MarkBayArrival(c *gin.Context) {
	assignment, err := h.Svc.MarkBayArrival(c.Request.Context(), c.GetString("userID"), c.Param("tripId"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": assignment})
}

func (h *Handler) SubscriptionPlans(c *gin.Context) {
	plans, err := h.Svc.SubscriptionPlans(c.Request.Context(), c.GetString("role") == "admin")
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": plans})
}

func (h *Handler) SaveSubscriptionPlan(c *gin.Context) {
	var payload models.SubscriptionPlan
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "valid plan fields are required"})
		return
	}
	plan, err := h.Svc.SaveSubscriptionPlan(c.Request.Context(), payload)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": plan})
}

func (h *Handler) SetSubscriptionPlanActive(c *gin.Context) {
	var payload struct {
		Active *bool `json:"active" binding:"required"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "active must be true or false"})
		return
	}
	if err := h.Svc.SetSubscriptionPlanActive(c.Request.Context(), c.Param("id"), *payload.Active); err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"id": c.Param("id"), "active": *payload.Active}})
}

func (h *Handler) CurrentSubscription(c *gin.Context) {
	subscription, err := h.Svc.CurrentSubscription(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": subscription})
}

func (h *Handler) Subscribe(c *gin.Context) {
	var payload struct {
		PlanID string `json:"planId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "planId is required"})
		return
	}
	subscription, err := h.Svc.Subscribe(c.Request.Context(), c.GetString("userID"), payload.PlanID, c.GetHeader("Idempotency-Key"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": subscription})
}

func (h *Handler) CancelSubscription(c *gin.Context) {
	if err := h.Svc.CancelSubscription(c.Request.Context(), c.GetString("userID"), c.Param("id")); err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"cancelled": true}})
}

func writeServiceError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	message := "request could not be completed"
	switch {
	case errors.Is(err, transit.ErrNotConfigured):
		status = http.StatusServiceUnavailable
		message = "transit routing is unavailable until Google Maps Routes API is configured"
	case errors.Is(err, transit.ErrInvalidRequest):
		status = http.StatusBadRequest
		message = "departure time must be between 7 days ago and 100 days from now"
	case errors.Is(err, transit.ErrNoTransitRoute):
		status = http.StatusNotFound
		message = "no transit itinerary with a train or metro leg is available for this request"
	case errors.Is(err, transit.ErrProviderFailure):
		status = http.StatusBadGateway
		message = "transit provider is temporarily unavailable"
	case errors.Is(err, routing.ErrInvalidCoordinates):
		status = http.StatusBadRequest
		message = "route coordinates are invalid"
	case errors.Is(err, routing.ErrRouterUnavailable):
		status = http.StatusServiceUnavailable
		message = "road routing is unavailable until an OSRM service is configured"
	case errors.Is(err, otp.ErrNotConfigured):
		status = http.StatusServiceUnavailable
		message = "phone verification is unavailable until the SMS provider is configured"
	case errors.Is(err, otp.ErrInvalidCode), errors.Is(err, storage.ErrInvalidCode):
		status = http.StatusBadRequest
		message = "verification code is invalid or expired"
	case errors.Is(err, otp.ErrProviderRejected):
		status = http.StatusBadGateway
		message = "SMS provider rejected the verification request"
	case errors.Is(err, storage.ErrInvalidLocation):
		status = http.StatusBadRequest
		message = "location values are invalid"
	case errors.Is(err, storage.ErrInvalidDocument):
		status = http.StatusBadRequest
		message = "document type, file name, or file content is not allowed"
	case errors.Is(err, storage.ErrInvalidBay):
		status = http.StatusConflict
		message = "no suitable open bay is available, or the bay configuration is invalid"
	case errors.Is(err, storage.ErrInvalidSubscriptionPlan), errors.Is(err, storage.ErrInvalidSubscriptionRequest):
		status = http.StatusBadRequest
		message = "subscription plan fields or idempotency key are invalid"
	case errors.Is(err, storage.ErrActiveSubscription):
		status = http.StatusConflict
		message = "cancel the current subscription before purchasing another plan"
	case errors.Is(err, storage.ErrInvalidStation):
		status = http.StatusBadRequest
		message = "origin station is not active in the metro station catalog"
	case errors.Is(err, routing.ErrInvalidCoordinates):
		status = http.StatusBadRequest
		message = "pickup and drop-off coordinates are invalid"
	case errors.Is(err, routing.ErrRouterUnavailable):
		status = http.StatusServiceUnavailable
		message = "road routing is temporarily unavailable; no estimated fare or trip was created"
	case errors.Is(err, storage.ErrDriverBusy):
		status = http.StatusConflict
		message = "availability cannot change while a trip is in progress"
	case errors.Is(err, storage.ErrDriverNotVerified):
		status = http.StatusForbidden
		message = "driver verification is required before going available"
	case errors.Is(err, storage.ErrVehicleUnavailable):
		status = http.StatusConflict
		message = "vehicle must be active, verified, and not assigned to another driver"
	case errors.Is(err, auth.ErrInvalidCredentials):
		status = http.StatusUnauthorized
		message = "invalid email or password"
	case errors.Is(err, auth.ErrInvalidRegistration):
		status = http.StatusBadRequest
		message = "registration details do not meet requirements"
	case errors.Is(err, auth.ErrInvalidInvitation), errors.Is(err, storage.ErrInvalidInvitation):
		status = http.StatusBadRequest
		message = "a valid, unused invitation code for this account type is required"
	case errors.Is(err, storage.ErrConflict):
		status = http.StatusConflict
		message = "an account with this email already exists"
	case errors.Is(err, storage.ErrTripNotCompleted):
		status = http.StatusConflict
		message = "trip ratings are available after a trip is completed"
	case errors.Is(err, storage.ErrAlreadyRated):
		status = http.StatusConflict
		message = "this trip has already been rated"
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
	case strings.Contains(err.Error(), "support status") || strings.Contains(err.Error(), "category, subject") || strings.Contains(err.Error(), "rating must"):
		status = http.StatusBadRequest
		message = err.Error()
	}
	c.JSON(status, gin.H{"success": false, "message": message, "timestamp": time.Now().UTC()})
}
