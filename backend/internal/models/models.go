package models

import "time"

type User struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Role         string    `json:"role"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
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
}

type Vehicle struct {
	ID             string `json:"id"`
	Type           string `json:"type"`
	Capacity       int    `json:"capacity"`
	CurrentStation string `json:"currentStation,omitempty"`
	Available      bool   `json:"available"`
}

type Station struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Zone      string  `json:"zone,omitempty"`
}

type Trip struct {
	ID             string    `json:"id"`
	RiderID        string    `json:"riderId"`
	DriverID       string    `json:"driverId,omitempty"`
	Status         string    `json:"status"`
	OriginID       string    `json:"originId"`
	Destination    string    `json:"destination"`
	PoolMode       string    `json:"poolMode"`
	PoolID         string    `json:"poolId,omitempty"`
	DistanceMeters float64   `json:"distanceMeters"`
	FareAmount     float64   `json:"fareAmount,omitempty"`
	FareCents      int64     `json:"fareCents"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type PoolCandidate struct {
	RiderID string  `json:"riderId"`
	Score   float64 `json:"score"`
	Reason  string  `json:"reason,omitempty"`
}

type FareQuote struct {
	TripID        string  `json:"tripId,omitempty"`
	BaseFare      float64 `json:"baseFare"`
	PoolDiscount  float64 `json:"poolDiscount"`
	Total         float64 `json:"total"`
	BaseCents     int64   `json:"baseCents"`
	DistanceCents int64   `json:"distanceCents"`
	DiscountCents int64   `json:"discountCents"`
	TotalCents    int64   `json:"totalCents"`
	Currency      string  `json:"currency"`
}

type WalletEntry struct {
	ID             string    `json:"id"`
	UserID         string    `json:"userId"`
	TripID         string    `json:"tripId,omitempty"`
	IdempotencyKey string    `json:"idempotencyKey"`
	AmountCents    int64     `json:"amountCents"`
	EntryType      string    `json:"entryType"`
	Description    string    `json:"description"`
	CreatedAt      time.Time `json:"createdAt"`
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
