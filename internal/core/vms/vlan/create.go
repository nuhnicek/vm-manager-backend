package vlan

import (
	"context"
	"fmt"
	"strconv"
	"time"
	"vm-manager-backend/types"
)

func ConnectVLAN(vm1id string, nic1 string, vm2id string, nic2 string, conn types.Conn, userUUID string, roles ...string) error {
	id1, err := strconv.Atoi(vm1id)
	if err != nil {
		return fmt.Errorf("invalid vm1id: %w", err)
	}
	id2, err := strconv.Atoi(vm2id)
	if err != nil {
		return fmt.Errorf("invalid vm2id: %w", err)
	}

	vlanTag, err := GetNextVLAN(conn)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	m1, err := FindEndpoint(ctx, conn, id1)
	if err != nil {
		return err
	}

	m2, err := FindEndpoint(ctx, conn, id2)
	if err != nil {
		return err
	}

	if !verifyOwner(m1.Tags, userUUID, roles...) || !verifyOwner(m2.Tags, userUUID, roles...) {
		return fmt.Errorf("unauthorized: machines do not belong to user")
	}

	vm1Net := nic1
	if vm1Net == "" {
		vm1Net, err = m1.NextAvailableNet()
		if err != nil {
			return err
		}
	}

	vm2Net := nic2
	if vm2Net == "" {
		vm2Net, err = m2.NextAvailableNet()
		if err != nil {
			return err
		}
	}

	err = m1.ApplyVLAN(ctx, vm1Net, vlanTag)
	if err != nil {
		return fmt.Errorf("failed to connect machine %d to network %d: %w", id1, vlanTag, err)
	}

	err = m2.ApplyVLAN(ctx, vm2Net, vlanTag)
	if err != nil {
		return fmt.Errorf("failed to connect machine %d to network %d: %w", id2, vlanTag, err)
	}

	return nil
}

func CreateVLAN(vm1id string, vm2id string, conn types.Conn) error {
	return ConnectVLAN(vm1id, "", vm2id, "", conn, "")
}
