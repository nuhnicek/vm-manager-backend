package auth

import (
	"context"
	"vm-manager-backend/types"
)

func Session(token string, conn types.Conn) (error, string) {
	uuid := conn.Redis.Get(context.TODO(), token)
	if uuid.Err() != nil {
		return uuid.Err(), ""
	}
	return nil, uuid.Val()
}
