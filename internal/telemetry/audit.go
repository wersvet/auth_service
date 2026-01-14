package telemetry

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"auth-service/internal/rabbitmq"
)

type Emitter interface {
	EmitAudit(ctx context.Context, level, text, requestID string, userID *int64) error
}

type AuditEmitter struct {
	publisher   rabbitmq.Publisher
	service     string
	environment string
}

type auditPayload struct {
	Level string `json:"level"`
	Text  string `json:"text"`
}

type auditEnvelope struct {
	SchemaVersion int          `json:"schema_version"`
	EventID       string       `json:"event_id"`
	EventType     string       `json:"event_type"`
	OccurredAt    string       `json:"occurred_at"`
	Service       string       `json:"service"`
	Environment   string       `json:"environment"`
	RequestID     string       `json:"request_id"`
	UserID        *int64       `json:"user_id,omitempty"`
	Payload       auditPayload `json:"payload"`
}

func NewAuditEmitter(publisher rabbitmq.Publisher, service, environment string) *AuditEmitter {
	return &AuditEmitter{
		publisher:   publisher,
		service:     service,
		environment: environment,
	}
}

func (e *AuditEmitter) EmitAudit(ctx context.Context, level, text, requestID string, userID *int64) error {
	if e == nil || e.publisher == nil {
		log.Printf("audit emitter is not configured")
		return nil
	}

	if requestID == "" {
		requestID = uuid.NewString()
	}

	envelope := auditEnvelope{
		SchemaVersion: 1,
		EventID:       uuid.NewString(),
		EventType:     "audit_log",
		OccurredAt:    time.Now().UTC().Format(time.RFC3339Nano),
		Service:       e.service,
		Environment:   e.environment,
		RequestID:     requestID,
		UserID:        userID,
		Payload: auditPayload{
			Level: level,
			Text:  text,
		},
	}

	routingKey := fmt.Sprintf("%s.audit", e.service)
	return e.publisher.Publish(ctx, routingKey, envelope)
}
