package storage

import (
	"context"
	"errors"
	"fmt"
	"math"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	auditdomain "t11-unstoppables/internal/audit"
	"t11-unstoppables/internal/bays"
	"t11-unstoppables/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrInvalidLocation = errors.New("location is invalid")
var ErrInvalidDocument = errors.New("document type or content is invalid")
var ErrInvalidCode = errors.New("verification code is invalid or expired")
var ErrInvalidBay = errors.New("bay is invalid or unavailable")

func (s *PostgresStore) VerifyUserPhone(ctx context.Context, userID, phone string) error {
	id, err := uuid.Parse(userID)
	if err != nil {
		return ErrNotFound
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin phone verification update: %w", err)
	}
	defer tx.Rollback(ctx)
	result, err := tx.Exec(ctx, "UPDATE users SET phone = $2, phone_verified = TRUE WHERE id = $1", id, phone)
	if err != nil {
		return fmt.Errorf("save verified phone number: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	if _, err := tx.Exec(ctx, "UPDATE rider_profiles SET phone = $2 WHERE user_id = $1", id, phone); err != nil {
		return fmt.Errorf("save verified rider phone number: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit phone verification update: %w", err)
	}
	return nil
}

func (s *PostgresStore) SaveDriverLocation(ctx context.Context, driverID string, location models.DriverLocation) error {
	id, err := uuid.Parse(driverID)
	if err != nil {
		return ErrNotFound
	}
	if math.IsNaN(location.Latitude) || math.IsInf(location.Latitude, 0) || location.Latitude < -90 || location.Latitude > 90 ||
		math.IsNaN(location.Longitude) || math.IsInf(location.Longitude, 0) || location.Longitude < -180 || location.Longitude > 180 ||
		math.IsNaN(location.AccuracyMeters) || math.IsInf(location.AccuracyMeters, 0) || location.AccuracyMeters < 0 || location.AccuracyMeters > 10000 ||
		location.SpeedMPS != nil && (*location.SpeedMPS < 0 || math.IsNaN(*location.SpeedMPS) || math.IsInf(*location.SpeedMPS, 0)) ||
		location.HeadingDegrees != nil && (*location.HeadingDegrees < 0 || *location.HeadingDegrees >= 360 || math.IsNaN(*location.HeadingDegrees) || math.IsInf(*location.HeadingDegrees, 0)) {
		return ErrInvalidLocation
	}
	tripUUID, err := uuid.Parse(location.TripID)
	if err != nil {
		return ErrNotFound
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin driver location update: %w", err)
	}
	defer tx.Rollback(ctx)
	var active bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM trip_requests
			WHERE id = $1 AND driver_id = $2 AND status IN ('matched', 'in_progress')
		)
	`, tripUUID, id).Scan(&active)
	if err != nil {
		return fmt.Errorf("validate driver location trip: %w", err)
	}
	if !active {
		return ErrNotFound
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO driver_locations (
			driver_id, trip_id, latitude, longitude, accuracy_meters, speed_mps, heading_degrees, recorded_at
		)
		SELECT $1, t.id, $2, $3, $4, $5, $6, NOW()
		FROM trip_requests t
		WHERE t.driver_id = $1 AND t.status IN ('matched', 'in_progress')
		ON CONFLICT (driver_id, trip_id) DO UPDATE SET
			latitude = EXCLUDED.latitude,
			longitude = EXCLUDED.longitude,
			accuracy_meters = EXCLUDED.accuracy_meters,
			speed_mps = EXCLUDED.speed_mps,
			heading_degrees = EXCLUDED.heading_degrees,
			recorded_at = NOW()
	`, id, location.Latitude, location.Longitude, location.AccuracyMeters, location.SpeedMPS, location.HeadingDegrees)
	if err != nil {
		return fmt.Errorf("save driver location: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit driver location: %w", err)
	}
	return nil
}

func (s *PostgresStore) TripLocation(ctx context.Context, tripID, userID, role string) (models.DriverLocation, error) {
	tripUUID, err := uuid.Parse(tripID)
	if err != nil {
		return models.DriverLocation{}, ErrNotFound
	}
	actorUUID, err := uuid.Parse(userID)
	if err != nil {
		return models.DriverLocation{}, ErrNotFound
	}
	var result models.DriverLocation
	var driverID uuid.UUID
	err = s.Pool.QueryRow(ctx, `
		SELECT l.driver_id, l.trip_id, l.latitude, l.longitude, l.accuracy_meters, l.speed_mps,
		       l.heading_degrees, l.recorded_at
		FROM driver_locations l
		JOIN trip_requests t ON t.id = l.trip_id
		WHERE t.id = $1
		  AND (t.status IN ('matched', 'in_progress'))
		  AND (($3 = 'admin') OR ($3 = 'rider' AND t.rider_id = $2) OR ($3 = 'driver' AND t.driver_id = $2))
	`, tripUUID, actorUUID, role).Scan(
		&driverID, &result.TripID, &result.Latitude, &result.Longitude, &result.AccuracyMeters,
		&result.SpeedMPS, &result.HeadingDegrees, &result.RecordedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.DriverLocation{}, ErrNotFound
	}
	if err != nil {
		return models.DriverLocation{}, fmt.Errorf("load trip location: %w", err)
	}
	result.DriverID = driverID.String()
	return result, nil
}

func (s *PostgresStore) SaveDriverDocument(ctx context.Context, driverID, documentType, filename, contentType, directory string, data, encryptionKey []byte) (models.DriverDocument, error) {
	driverUUID, err := uuid.Parse(driverID)
	if err != nil {
		return models.DriverDocument{}, ErrNotFound
	}
	if documentType != "driving_license" && documentType != "vehicle_registration" && documentType != "insurance" && documentType != "identity" {
		return models.DriverDocument{}, ErrInvalidDocument
	}
	detected := http.DetectContentType(data)
	if detected == "application/octet-stream" && strings.HasPrefix(string(data), "%PDF-") {
		detected = "application/pdf"
	}
	if detected != contentType || contentType != "application/pdf" && contentType != "image/jpeg" && contentType != "image/png" {
		return models.DriverDocument{}, ErrInvalidDocument
	}
	filename = filepath.Base(strings.ReplaceAll(strings.TrimSpace(filename), `\`, "/"))
	if filename == "." || filename == "" || len(filename) > 180 {
		return models.DriverDocument{}, ErrInvalidDocument
	}
	extension := map[string]string{"application/pdf": ".pdf", "image/jpeg": ".jpg", "image/png": ".png"}[contentType]
	if ext := strings.ToLower(filepath.Ext(filename)); ext != "" && ext != extension && !(contentType == "image/jpeg" && ext == ".jpeg") {
		return models.DriverDocument{}, ErrInvalidDocument
	}
	if _, err := mime.ExtensionsByType(contentType); err != nil {
		return models.DriverDocument{}, ErrInvalidDocument
	}
	storedData, err := encryptDriverDocument(data, encryptionKey)
	if err != nil {
		return models.DriverDocument{}, fmt.Errorf("encrypt driver document: %w", err)
	}
	if err := os.MkdirAll(directory, 0700); err != nil {
		return models.DriverDocument{}, fmt.Errorf("create private document directory: %w", err)
	}
	key := uuid.NewString() + extension
	path := filepath.Join(directory, key)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return models.DriverDocument{}, fmt.Errorf("create driver document: %w", err)
	}
	if _, err := file.Write(storedData); err != nil {
		file.Close()
		_ = os.Remove(path)
		return models.DriverDocument{}, fmt.Errorf("write driver document: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return models.DriverDocument{}, fmt.Errorf("close driver document: %w", err)
	}
	var document models.DriverDocument
	var documentID, ownerID uuid.UUID
	err = s.Pool.QueryRow(ctx, `
		INSERT INTO driver_documents (driver_id, document_type, file_name, content_type, storage_key)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, driver_id, document_type, file_name, content_type, status, uploaded_at
	`, driverUUID, documentType, filename, contentType, key).Scan(
		&documentID, &ownerID, &document.DocumentType, &document.FileName, &document.ContentType, &document.Status, &document.UploadedAt,
	)
	if err != nil {
		_ = os.Remove(path)
		return models.DriverDocument{}, fmt.Errorf("save driver document metadata: %w", err)
	}
	document.ID = documentID.String()
	document.DriverID = ownerID.String()
	return document, nil
}

func (s *PostgresStore) ListDriverDocuments(ctx context.Context, driverID string, all bool) ([]models.DriverDocument, error) {
	var driverUUID *uuid.UUID
	if !all {
		parsed, err := uuid.Parse(driverID)
		if err != nil {
			return nil, ErrNotFound
		}
		driverUUID = &parsed
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT id, driver_id, document_type, file_name, content_type, status, review_note, uploaded_at, reviewed_at
		FROM driver_documents
		WHERE $1::uuid IS NULL OR driver_id = $1
		ORDER BY uploaded_at DESC
	`, driverUUID)
	if err != nil {
		return nil, fmt.Errorf("list driver documents: %w", err)
	}
	defer rows.Close()
	documents := make([]models.DriverDocument, 0)
	for rows.Next() {
		var id, owner uuid.UUID
		var item models.DriverDocument
		if err := rows.Scan(&id, &owner, &item.DocumentType, &item.FileName, &item.ContentType, &item.Status, &item.ReviewNote, &item.UploadedAt, &item.ReviewedAt); err != nil {
			return nil, fmt.Errorf("scan driver document: %w", err)
		}
		item.ID, item.DriverID = id.String(), owner.String()
		documents = append(documents, item)
	}
	return documents, rows.Err()
}

func (s *PostgresStore) ReviewDriverDocument(ctx context.Context, adminID, documentID, status, note string) (models.DriverDocument, error) {
	adminUUID, err := uuid.Parse(adminID)
	if err != nil {
		return models.DriverDocument{}, ErrNotFound
	}
	id, err := uuid.Parse(documentID)
	if err != nil {
		return models.DriverDocument{}, ErrNotFound
	}
	if status != "approved" && status != "rejected" || len(note) > 500 {
		return models.DriverDocument{}, ErrInvalidDocument
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return models.DriverDocument{}, fmt.Errorf("begin document review: %w", err)
	}
	defer tx.Rollback(ctx)
	var document models.DriverDocument
	var actualID, owner uuid.UUID
	err = tx.QueryRow(ctx, `
		UPDATE driver_documents SET status = $2, review_note = $3, reviewed_at = NOW()
		WHERE id = $1
		RETURNING id, driver_id, document_type, file_name, content_type, status, review_note, uploaded_at, reviewed_at
	`, id, status, strings.TrimSpace(note)).Scan(
		&actualID, &owner, &document.DocumentType, &document.FileName, &document.ContentType,
		&document.Status, &document.ReviewNote, &document.UploadedAt, &document.ReviewedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.DriverDocument{}, ErrNotFound
	}
	if err != nil {
		return models.DriverDocument{}, fmt.Errorf("review driver document: %w", err)
	}
	document.ID, document.DriverID = actualID.String(), owner.String()
	if err := writeAudit(ctx, tx, auditdomain.Event{
		ID: uuid.NewString(), ActorID: adminUUID.String(), Action: "driver.document_reviewed",
		Resource: "driver_document", ResourceID: actualID.String(), OccurredAt: time.Now().UTC(),
		Metadata: map[string]string{"status": status},
	}); err != nil {
		return models.DriverDocument{}, fmt.Errorf("audit document review: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return models.DriverDocument{}, fmt.Errorf("commit document review: %w", err)
	}
	return document, nil
}

func (s *PostgresStore) ReadDriverDocument(ctx context.Context, documentID, directory string, encryptionKey []byte) (models.DriverDocument, []byte, error) {
	id, err := uuid.Parse(documentID)
	if err != nil {
		return models.DriverDocument{}, nil, ErrNotFound
	}
	var document models.DriverDocument
	var actualID, owner uuid.UUID
	var key string
	err = s.Pool.QueryRow(ctx, `
		SELECT id, driver_id, document_type, file_name, content_type, status, review_note, uploaded_at, reviewed_at, storage_key
		FROM driver_documents WHERE id = $1
	`, id).Scan(&actualID, &owner, &document.DocumentType, &document.FileName, &document.ContentType, &document.Status,
		&document.ReviewNote, &document.UploadedAt, &document.ReviewedAt, &key)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.DriverDocument{}, nil, ErrNotFound
	}
	if err != nil {
		return models.DriverDocument{}, nil, fmt.Errorf("find driver document: %w", err)
	}
	if filepath.Base(key) != key {
		return models.DriverDocument{}, nil, errors.New("invalid stored document key")
	}
	data, err := os.ReadFile(filepath.Join(directory, key))
	if err != nil {
		return models.DriverDocument{}, nil, fmt.Errorf("read driver document: %w", err)
	}
	data, err = decryptDriverDocument(data, encryptionKey)
	if err != nil {
		return models.DriverDocument{}, nil, fmt.Errorf("decrypt driver document: %w", err)
	}
	document.ID, document.DriverID = actualID.String(), owner.String()
	return document, data, nil
}

func (s *PostgresStore) DriverEarnings(ctx context.Context, driverID string) (models.DriverEarnings, error) {
	id, err := uuid.Parse(driverID)
	if err != nil {
		return models.DriverEarnings{}, ErrNotFound
	}
	var result models.DriverEarnings
	err = s.Pool.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(SUM(fare_paise), 0), COALESCE(SUM(fare_paise * 15 / 100), 0)
		FROM trip_requests
		WHERE driver_id = $1 AND status = 'completed'
	`, id).Scan(&result.TripsCompleted, &result.GrossPaise, &result.CommissionPaise)
	if err != nil {
		return models.DriverEarnings{}, fmt.Errorf("calculate driver earnings: %w", err)
	}
	result.NetPaise = result.GrossPaise - result.CommissionPaise
	return result, nil
}

func (s *PostgresStore) DriverRatingSummary(ctx context.Context, driverID string) (models.DriverRatingSummary, error) {
	id, err := uuid.Parse(driverID)
	if err != nil {
		return models.DriverRatingSummary{}, ErrNotFound
	}
	var result models.DriverRatingSummary
	err = s.Pool.QueryRow(ctx, `
		SELECT COALESCE(AVG(stars), 0), COUNT(*) FROM trip_ratings WHERE driver_id = $1
	`, id).Scan(&result.Average, &result.Count)
	if err != nil {
		return models.DriverRatingSummary{}, fmt.Errorf("calculate driver rating summary: %w", err)
	}
	return result, nil
}

func (s *PostgresStore) ListBays(ctx context.Context) ([]models.StationBay, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT b.id, b.station_id, b.label, b.capacity,
		       b.queue_length + (SELECT COUNT(*) FROM bay_assignments a WHERE a.bay_id = b.id AND a.arrived_at IS NULL),
		       b.walking_meters, b.accessible, b.is_open
		FROM station_bays b ORDER BY b.station_id, b.label
	`)
	if err != nil {
		return nil, fmt.Errorf("list station bays: %w", err)
	}
	defer rows.Close()
	result := make([]models.StationBay, 0)
	for rows.Next() {
		var bay models.StationBay
		var id uuid.UUID
		if err := rows.Scan(&id, &bay.StationID, &bay.Label, &bay.Capacity, &bay.QueueLength, &bay.WalkingMeters, &bay.Accessible, &bay.Open); err != nil {
			return nil, fmt.Errorf("scan station bay: %w", err)
		}
		bay.ID = id.String()
		result = append(result, bay)
	}
	return result, rows.Err()
}

func (s *PostgresStore) SaveBay(ctx context.Context, bay models.StationBay) (models.StationBay, error) {
	if bay.StationID == "" || strings.TrimSpace(bay.Label) == "" || len(bay.Label) > 80 || bay.Capacity < 1 ||
		bay.QueueLength < 0 || bay.WalkingMeters < 0 || math.IsNaN(bay.WalkingMeters) || math.IsInf(bay.WalkingMeters, 0) {
		return models.StationBay{}, ErrInvalidBay
	}
	var id uuid.UUID
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO station_bays (station_id, label, capacity, queue_length, walking_meters, accessible, is_open)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (station_id, label) DO UPDATE SET
			capacity = EXCLUDED.capacity, queue_length = EXCLUDED.queue_length,
			walking_meters = EXCLUDED.walking_meters, accessible = EXCLUDED.accessible, is_open = EXCLUDED.is_open
		RETURNING id
	`, bay.StationID, strings.TrimSpace(bay.Label), bay.Capacity, bay.QueueLength, bay.WalkingMeters, bay.Accessible, bay.Open).Scan(&id)
	if err != nil {
		return models.StationBay{}, fmt.Errorf("save station bay: %w", err)
	}
	bay.ID = id.String()
	return bay, nil
}

func (s *PostgresStore) AssignBay(ctx context.Context, driverID, tripID string) (models.BayAssignment, error) {
	driverUUID, err := uuid.Parse(driverID)
	if err != nil {
		return models.BayAssignment{}, ErrNotFound
	}
	tripUUID, err := uuid.Parse(tripID)
	if err != nil {
		return models.BayAssignment{}, ErrNotFound
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return models.BayAssignment{}, fmt.Errorf("begin bay assignment: %w", err)
	}
	defer tx.Rollback(ctx)
	var stationID string
	var tripDriverID *uuid.UUID
	var status string
	var partySize int
	var requiresAccessible bool
	err = tx.QueryRow(ctx, "SELECT origin_id, driver_id, status, party_size, requires_accessible FROM trip_requests WHERE id = $1 FOR UPDATE", tripUUID).
		Scan(&stationID, &tripDriverID, &status, &partySize, &requiresAccessible)
	if errors.Is(err, pgx.ErrNoRows) || err == nil && (tripDriverID == nil || *tripDriverID != driverUUID || status != "matched" && status != "in_progress") {
		return models.BayAssignment{}, ErrNotFound
	}
	if err != nil {
		return models.BayAssignment{}, fmt.Errorf("load trip for bay assignment: %w", err)
	}
	var prior models.BayAssignment
	var priorID, priorBayID uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT a.id, a.trip_id, a.bay_id, b.station_id, b.label, a.arrived_at
		FROM bay_assignments a JOIN station_bays b ON b.id = a.bay_id WHERE a.trip_id = $1
	`, tripUUID).Scan(&priorID, &prior.TripID, &priorBayID, &prior.StationID, &prior.Label, &prior.ArrivedAt)
	if err == nil {
		prior.ID, prior.BayID = priorID.String(), priorBayID.String()
		return prior, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return models.BayAssignment{}, fmt.Errorf("check existing bay assignment: %w", err)
	}
	rows, err := tx.Query(ctx, `
		SELECT b.id, b.station_id, b.label, b.capacity, b.queue_length +
		       (SELECT COUNT(*) FROM bay_assignments a WHERE a.bay_id = b.id AND a.arrived_at IS NULL),
		       b.walking_meters, b.accessible, b.is_open
		FROM station_bays b WHERE b.station_id = $1 ORDER BY b.id FOR UPDATE
	`, stationID)
	if err != nil {
		return models.BayAssignment{}, fmt.Errorf("load station bays: %w", err)
	}
	candidates := make([]bays.Bay, 0)
	for rows.Next() {
		var bay bays.Bay
		var id uuid.UUID
		if err := rows.Scan(&id, &bay.StationID, &bay.Label, &bay.Capacity, &bay.QueueLength, &bay.WalkingMeters, &bay.Accessible, &bay.Open); err != nil {
			rows.Close()
			return models.BayAssignment{}, fmt.Errorf("scan available bay: %w", err)
		}
		bay.ID = id.String()
		candidates = append(candidates, bay)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return models.BayAssignment{}, fmt.Errorf("read available bays: %w", err)
	}
	rows.Close()
	selected, err := bays.Assign(bays.Request{
		StationID: stationID, PartySize: partySize, MaxWalkMeters: 500, RequiresAccessible: requiresAccessible,
	}, candidates)
	if err != nil {
		return models.BayAssignment{}, ErrInvalidBay
	}
	var assignment models.BayAssignment
	var assignmentID, assignmentTripID, assignmentBayID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO bay_assignments (trip_id, bay_id) VALUES ($1, $2)
		RETURNING id, trip_id, bay_id
	`, tripUUID, selected.ID).Scan(&assignmentID, &assignmentTripID, &assignmentBayID)
	if err != nil {
		return models.BayAssignment{}, fmt.Errorf("create bay assignment: %w", err)
	}
	assignment.ID, assignment.TripID, assignment.BayID = assignmentID.String(), assignmentTripID.String(), assignmentBayID.String()
	assignment.StationID, assignment.Label = selected.StationID, selected.Label
	if err := tx.Commit(ctx); err != nil {
		return models.BayAssignment{}, fmt.Errorf("commit bay assignment: %w", err)
	}
	return assignment, nil
}

func (s *PostgresStore) MarkBayArrival(ctx context.Context, driverID, tripID string) (models.BayAssignment, error) {
	driverUUID, err := uuid.Parse(driverID)
	if err != nil {
		return models.BayAssignment{}, ErrNotFound
	}
	tripUUID, err := uuid.Parse(tripID)
	if err != nil {
		return models.BayAssignment{}, ErrNotFound
	}
	var assignment models.BayAssignment
	var id, storedTripID, bayID uuid.UUID
	err = s.Pool.QueryRow(ctx, `
		UPDATE bay_assignments a SET arrived_at = COALESCE(a.arrived_at, NOW())
		FROM trip_requests t, station_bays b
		WHERE a.trip_id = t.id AND a.bay_id = b.id AND a.trip_id = $1
		  AND t.driver_id = $2 AND t.status IN ('matched', 'in_progress')
		RETURNING a.id, a.trip_id, a.bay_id, b.station_id, b.label, a.arrived_at
	`, tripUUID, driverUUID).Scan(&id, &storedTripID, &bayID, &assignment.StationID, &assignment.Label, &assignment.ArrivedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.BayAssignment{}, ErrNotFound
	}
	if err != nil {
		return models.BayAssignment{}, fmt.Errorf("mark bay arrival: %w", err)
	}
	assignment.ID, assignment.TripID, assignment.BayID = id.String(), storedTripID.String(), bayID.String()
	return assignment, nil
}
