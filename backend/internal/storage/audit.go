package storage

import (
	"context"
	"encoding/json"
	"fmt"

	auditdomain "t11-unstoppables/internal/audit"
	"t11-unstoppables/queries"

	"github.com/jackc/pgx/v5"
)

func writeAudit(ctx context.Context, tx pgx.Tx, event auditdomain.Event) error {
	if err := event.Validate(); err != nil {
		return err
	}
	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("encode audit metadata: %w", err)
	}
	_, err = tx.Exec(ctx, queries.Must("insert_audit_event.sql"),
		event.ID, event.ActorID, event.Action, event.Resource, event.ResourceID, metadata, event.OccurredAt,
	)
	return err
}
