package keybuilder

import (
	"fmt"
	"github.com/google/uuid"
)

const (
	Redis        string = "redis"
	Notification string = "notification"
)

func RedisNotificationKeyBuild(id uuid.UUID) string {
	return fmt.Sprintf("%s:%s:%s", Redis, Notification, id)
}
