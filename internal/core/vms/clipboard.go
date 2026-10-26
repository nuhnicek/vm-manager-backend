package vms

import (
	"context"
	"fmt"
	"strings"
	"time"
	"vm-manager-backend/types"
)

func GetVMClipboard(conn types.Conn, userUUID string, role string, vmid int, targetNodes ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	vm, _, err := conn.FindVM(ctx, vmid, targetNodes...)
	if err != nil || vm == nil {
		return "", fmt.Errorf("VM %d not found", vmid)
	}

	if !VerifyVMOwner(vm, userUUID, role) {
		return "", fmt.Errorf("unauthorized: VM %d does not belong to user", vmid)
	}

	if !strings.EqualFold(vm.Status, "running") {
		return "", fmt.Errorf("VM %d is stopped", vmid)
	}

	cmd := []string{
		"bash",
		"-c",
		`XAUTH=$(ps aux | grep -oE -- '-auth [^ ]+' | awk '{print $2}' | head -n1); [ -z "$XAUTH" ] && XAUTH=$(find /run /var/run /home /root -name "*auth*" -o -name ".Xauthority" 2>/dev/null | head -n1); export XAUTHORITY="$XAUTH"; export DISPLAY=:0; timeout 2 xclip -o -selection clipboard 2>/dev/null || timeout 2 xclip -o -selection primary 2>/dev/null || timeout 2 xclip -o 2>/dev/null || timeout 2 xsel -b -o 2>/dev/null || timeout 2 wl-paste 2>/dev/null || cat /tmp/.guest_clip.txt 2>/dev/null`,
	}

	pid, err := vm.AgentExec(ctx, cmd, "")
	if err != nil {
		return "", err
	}

	status, err := vm.WaitForAgentExecExit(ctx, pid, 3)
	if err != nil {
		return "", err
	}
	if status != nil && status.OutData != "" {
		return status.OutData, nil
	}
	return "", nil
}

func SetVMClipboard(conn types.Conn, userUUID string, role string, vmid int, text string, targetNodes ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	vm, _, err := conn.FindVM(ctx, vmid, targetNodes...)
	if err != nil || vm == nil {
		return fmt.Errorf("VM %d not found", vmid)
	}

	if !VerifyVMOwner(vm, userUUID, role) {
		return fmt.Errorf("unauthorized: VM %d does not belong to user", vmid)
	}

	if !strings.EqualFold(vm.Status, "running") {
		return fmt.Errorf("VM %d is stopped", vmid)
	}

	cmd := []string{
		"bash",
		"-c",
		`XAUTH=$(ps aux | grep -oE -- '-auth [^ ]+' | awk '{print $2}' | head -n1); [ -z "$XAUTH" ] && XAUTH=$(find /run /var/run /home /root -name "*auth*" -o -name ".Xauthority" 2>/dev/null | head -n1); export XAUTHORITY="$XAUTH"; export DISPLAY=:0; cat > /tmp/.guest_clip.txt && (nohup xclip -selection clipboard -i /tmp/.guest_clip.txt >/dev/null 2>&1 &) || (nohup xsel -b -i /tmp/.guest_clip.txt >/dev/null 2>&1 &) || (nohup wl-copy </tmp/.guest_clip.txt >/dev/null 2>&1 &)`,
	}

	pid, err := vm.AgentExec(ctx, cmd, text)
	if err != nil {
		return err
	}
	_, _ = vm.WaitForAgentExecExit(ctx, pid, 3)
	return nil
}
