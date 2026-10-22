package teacher

import (
	"fmt"
	"strings"
	"vm-manager-backend/internal/core/vms"
	"vm-manager-backend/types"
)

func TeacherBulkAction(conn types.Conn, action string) error {
	allVMs, err := vms.List(conn, "")
	if err != nil {
		return fmt.Errorf("failed to list VMs: %w", err)
	}

	switch action {
	case "stop_all":
		for _, vm := range allVMs {
			if strings.EqualFold(vm.Status, "running") {
				_ = vms.StopVM(conn, "", "teacher", int(vm.Id))
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported bulk action: %s", action)
	}
}
