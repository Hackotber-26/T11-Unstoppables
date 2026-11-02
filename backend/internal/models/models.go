package models

import (
	"encoding/json"
	"time"
)

type User struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Email         string    `json:"email"`
	Role          string    `json:"role"`
	Phone         string    `json:"phone,omitempty"`
	PhoneVerified bool      `json:"phoneVerified"`
	PasswordHash  string    `json:"-"`
	CreatedAt     time.Time `json:"createdAt"`
}

type Rider struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	StationID string    `json:"stationId,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type Driver struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	VehicleID string `json:"vehicleId,omitempty"`
	Status    string `json:"status"`
	Capacity  int    `json:"capacity,omitempty"`
	Verified  bool   `json:"verified"`
}

type Vehicle struct {
	ID                     string `json:"id"`
	Label                  string `json:"label"`
	Type                   string `json:"vehicleType"`
	Capacity               int    `json:"capacity"`
	Active                 bool   `json:"active"`
	Verified               bool   `json:"verified"`
	AccessibilitySupported bool   `json:"accessibilitySupported"`
	DriverStatus           string `json:"driverStatus"`
}

type Station struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Lines []string `json:"lines"`
}

type Trip struct {
	ID                 string    `json:"id"`
	RiderID            string    `json:"riderId"`
	DriverID           string    `json:"driverId,omitempty"`
	Status             string    `json:"status"`
	OriginID           string    `json:"originId"`
	Destination        string    `json:"destination"`
	PoolMode           string    `json:"poolMode"`
	PoolID             string    `json:"poolId,omitempty"`
	DistanceMeters     float64   `json:"distanceMeters"`
	PickupLatitude     *float64  `json:"pickupLatitude,omitempty"`
	PickupLongitude    *float64  `json:"pickupLongitude,omitempty"`
	DropoffLatitude    *float64  `json:"dropoffLatitude,omitempty"`
	DropoffLongitude   *float64  `json:"dropoffLongitude,omitempty"`
	PartySize          int       `json:"partySize"`
	RequiresAccessible bool      `json:"requiresAccessible"`
	FareAmount         float64   `json:"fareAmount,omitempty"`
	FareCents          int64     `json:"farePaise"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type PoolCandidate struct {
	RiderID string  `json:"-"`
	Score   float64 `json:"score"`
	Reason  string  `json:"reason,omitempty"`
}

type PoolRouteStop struct {
	Sequence   int     `json:"sequence"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	IsYourStop bool    `json:"isYourStop"`
}

type PoolRoutePreview struct {
	Stops                    []PoolRouteStop `json:"stops"`
	DirectDistanceMeters     float64         `json:"directDistanceMeters"`
	DirectDurationSeconds    float64         `json:"directDurationSeconds"`
	SharedDistanceMeters     float64         `json:"sharedDistanceMeters"`
	SharedDurationSeconds    float64         `json:"sharedDurationSeconds"`
	PoolRouteDistanceMeters  float64         `json:"poolRouteDistanceMeters"`
	PoolRouteDurationSeconds float64         `json:"poolRouteDurationSeconds"`
	DistanceDeviationMeters  float64         `json:"distanceDeviationMeters"`
	TimeWastedSeconds        float64         `json:"timeWastedSeconds"`
	Geometry                 json.RawMessage `json:"geometry"`
}

type FareQuote struct {
	TripID                    string          `json:"tripId,omitempty"`
	BaseFare                  float64         `json:"baseFare"`
	PoolDiscount              float64         `json:"poolDiscount"`
	Total                     float64         `json:"total"`
	BaseCents                 int64           `json:"basePaise"`
	DistanceCents             int64           `json:"distancePaise"`
	DiscountCents             int64           `json:"discountPaise"`
	SubscriptionDiscountPaise int64           `json:"subscriptionDiscountPaise"`
	TotalCents                int64           `json:"totalPaise"`
	Currency                  string          `json:"currency"`
	DistanceMeters            float64         `json:"distanceMeters"`
	DurationSeconds           float64         `json:"durationSeconds"`
	Geometry                  json.RawMessage `json:"geometry"`
}

type WalletEntry struct {
	ID             string    `json:"id"`
	UserID         string    `json:"userId"`
	TripID         string    `json:"tripId,omitempty"`
	IdempotencyKey string    `json:"idempotencyKey"`
	AmountCents    int64     `json:"amountPaise"`
	EntryType      string    `json:"entryType"`
	Description    string    `json:"description"`
	CreatedAt      time.Time `json:"createdAt"`
}

type SupportTicket struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Category  string    `json:"category"`
	Subject   string    `json:"subject"`
	Message   string    `json:"message"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type TripRating struct {
	ID        string    `json:"id"`
	TripID    string    `json:"tripId"`
	RiderID   string    `json:"riderId"`
	DriverID  string    `json:"driverId"`
	Stars     int       `json:"stars"`
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"createdAt"`
}

type DriverLocation struct {
	DriverID       string    `json:"driverId"`
	TripID         string    `json:"tripId,omitempty"`
	Latitude       float64   `json:"latitude"`
	Longitude      float64   `json:"longitude"`
	AccuracyMeters float64   `json:"accuracyMeters"`
	SpeedMPS       *float64  `json:"speedMps,omitempty"`
	HeadingDegrees *float64  `json:"headingDegrees,omitempty"`
	RecordedAt     time.Time `json:"recordedAt"`
}

type DriverDocument struct {
	ID           string     `json:"id"`
	DriverID     string     `json:"driverId"`
	DocumentType string     `json:"documentType"`
	FileName     string     `json:"fileName"`
	ContentType  string     `json:"contentType"`
	Status       string     `json:"status"`
	ReviewNote   string     `json:"reviewNote,omitempty"`
	UploadedAt   time.Time  `json:"uploadedAt"`
	ReviewedAt   *time.Time `json:"reviewedAt,omitempty"`
}

type StationBay struct {
	ID            string  `json:"id"`
	StationID     string  `json:"stationId"`
	Label         string  `json:"label"`
	Capacity      int     `json:"capacity"`
	QueueLength   int     `json:"queueLength"`
	WalkingMeters float64 `json:"walkingMeters"`
	Accessible    bool    `json:"accessible"`
	Open          bool    `json:"open"`
}

type BayAssignment struct {
	ID        string     `json:"id"`
	TripID    string     `json:"tripId"`
	BayID     string     `json:"bayId"`
	StationID string     `json:"stationId"`
	Label     string     `json:"label"`
	ArrivedAt *time.Time `json:"arrivedAt,omitempty"`
}

type SubscriptionPlan struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	PricePaise       int64  `json:"pricePaise"`
	MonthlyRideLimit int    `json:"monthlyRideLimit"`
	DiscountBPS      int    `json:"discountBps"`
	Active           bool   `json:"active"`
}

type UserSubscription struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	PlanID    string    `json:"planId"`
	PlanName  string    `json:"planName"`
	StartsAt  time.Time `json:"startsAt"`
	EndsAt    time.Time `json:"endsAt"`
	Cancelled bool      `json:"cancelled"`
}

type DriverEarnings struct {
	TripsCompleted  int   `json:"tripsCompleted"`
	GrossPaise      int64 `json:"grossPaise"`
	CommissionPaise int64 `json:"commissionPaise"`
	NetPaise        int64 `json:"netPaise"`
}

type DriverRatingSummary struct {
	Average float64 `json:"average"`
	Count   int     `json:"count"`
}

type HealthStatus struct {
	Status   string            `json:"status"`
	Services map[string]string `json:"services"`
}

type ApiResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}
