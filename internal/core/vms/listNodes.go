package vms

import (
	"context"
	"time"
	"vm-manager-backend/types"
)

func ListNodes(conn types.Conn) []types.ProxmoxNodeInfo {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return conn.GetNodesInfo(ctx)
}
