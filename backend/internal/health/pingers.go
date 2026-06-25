package health

import (
	"context"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// NewDefaultPingers builds Pinger implementations for PostgreSQL and Redis.
func NewDefaultPingers(db *gorm.DB, redisClient *redis.Client) map[string]Pinger {
	return map[string]Pinger{
		"postgres": &postgresPinger{db: db},
		"redis":    &redisPinger{client: redisClient},
	}
}

type postgresPinger struct {
	db *gorm.DB
}

func (p *postgresPinger) Ping(ctx context.Context) error {
	sqlDB, err := p.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

type redisPinger struct {
	client *redis.Client
}

func (p *redisPinger) Ping(ctx context.Context) error {
	return p.client.Ping(ctx).Err()
}
