package email

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/dogfood-platform/dogfood/internal/auth/port"
	"github.com/redis/go-redis/v9"
)

const queueName = "email_queue"

type emailTask struct {
	Type  string `json:"type"`
	Email string `json:"email"`
	Token string `json:"token"`
}

type QueueSender struct {
	rdb *redis.Client
}

func NewQueueSender(rdb *redis.Client) *QueueSender {
	return &QueueSender{rdb: rdb}
}

func (q *QueueSender) SendVerificationEmail(ctx context.Context, email, token string) error {
	task := emailTask{
		Type:  "verify_email",
		Email: email,
		Token: token,
	}

	payload, err := json.Marshal(task)
	if err != nil {
		return err
	}

	return q.rdb.LPush(ctx, queueName, payload).Err()
}

func RunWorker(ctx context.Context, rdb *redis.Client, sender port.EmailSender) {
	slog.Info("starting email queue worker")

	for {
		select {
		case <-ctx.Done():
			slog.Info("stopping email queue worker")
			return
		default:
			// Block for 2 seconds waiting for a new item
			res, err := rdb.BRPop(ctx, 2*time.Second, queueName).Result()
			if err == redis.Nil {
				continue // Timeout, try again
			} else if err != nil {
				// Context cancelled or connection issue
				slog.Error("error popping from email queue", "error", err)
				time.Sleep(1 * time.Second)
				continue
			}

			// res[0] is list name, res[1] is the value
			if len(res) == 2 {
				var task emailTask
				if err := json.Unmarshal([]byte(res[1]), &task); err != nil {
					slog.Error("failed to unmarshal email task", "error", err, "payload", res[1])
					continue
				}

				if task.Type == "verify_email" {
					if err := sender.SendVerificationEmail(ctx, task.Email, task.Token); err != nil {
						slog.Error("failed to send verification email", "error", err, "email", task.Email)
						// In a robust system, we would put it in a dead letter queue or retry later
					} else {
						slog.Info("successfully processed verification email", "email", task.Email)
					}
				}
			}
		}
	}
}
