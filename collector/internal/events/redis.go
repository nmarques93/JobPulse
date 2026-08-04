package events

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
	"jobpulse/collector/internal/model"
)

type Publisher struct {
	client *redis.Client
	stream string
}

func NewPublisher(client *redis.Client, stream string) *Publisher {
	return &Publisher{client: client, stream: stream}
}

func (p *Publisher) Publish(ctx context.Context, posting model.JobPosting) error {
	payload, err := json.Marshal(posting)
	if err != nil {
		return err
	}
	return p.client.XAdd(ctx, &redis.XAddArgs{Stream: p.stream, Values: map[string]any{
		"event_type": "job.posting.discovered", "posting_id": posting.ID, "payload": string(payload),
	}}).Err()
}
