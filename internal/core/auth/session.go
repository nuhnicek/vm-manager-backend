package auth

import (
	"context"
	"vm-manager-backend/types"
)

func Session(token string, conn types.Conn) (error, string) {
	email := conn.Redis.Get(context.TODO(), token)
	if email.Err() != nil {
		println("nigga")
		println(email.Err())
		return email.Err(), ""
	}
	print(email.Val())
	return nil, email.Val()
}
