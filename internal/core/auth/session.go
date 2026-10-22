package auth

import (
	"context"
	"strings"
	"vm-manager-backend/types"
)

func Session(token string, conn types.Conn) (error, string, string) {
	val := conn.Redis.Get(context.TODO(), token)
	if val.Err() != nil {
		return val.Err(), "", ""
	}
	parts := strings.Split(val.Val(), "|")
	uuid := parts[0]
	role := "student"
	if len(parts) > 1 && parts[1] != "" {
		role = parts[1]
	}
	return nil, uuid, role
}
