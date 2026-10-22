package types

import "time"

type LabVMTemplate struct {
	Name        string `bson:"name" json:"name"`
	TemplateID  int    `bson:"template_id" json:"template_id"`
	Node        string `bson:"node,omitempty" json:"node,omitempty"`
	Description string `bson:"description" json:"description"`
	RAM         int    `bson:"ram_mb,omitempty" json:"ram_mb"`
	DefaultUser string `bson:"default_user,omitempty" json:"default_user"`
	DefaultPass string `bson:"default_password,omitempty" json:"default_password"`
}

type LabNetworkLink struct {
	FromVM  string `bson:"from_vm" json:"from_vm"`
	FromNIC string `bson:"from_nic,omitempty" json:"from_nic"`
	ToVM    string `bson:"to_vm" json:"to_vm"`
	ToNIC   string `bson:"to_nic,omitempty" json:"to_nic"`
	Label   string `bson:"label,omitempty" json:"label"`
}

type LabTemplate struct {
	ID          string           `bson:"_id,omitempty" json:"id"`
	Name        string           `bson:"name" json:"name"`
	Category    string           `bson:"category" json:"category"`
	Difficulty  string           `bson:"difficulty" json:"difficulty"`
	Description string           `bson:"description" json:"description"`
	Objectives  []string         `bson:"objectives" json:"objectives"`
	VMs         []LabVMTemplate  `bson:"vms" json:"vms"`
	Networks    []LabNetworkLink `bson:"networks" json:"networks"`
	CreatedBy   string           `bson:"created_by,omitempty" json:"created_by"`
	CreatedAt   time.Time        `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time        `bson:"updated_at" json:"updated_at"`
}

type LabInstanceVM struct {
	TemplateVMName string `bson:"template_vm_name" json:"template_vm_name"`
	VMID           int    `bson:"vmid" json:"vmid"`
	Name           string `bson:"name" json:"name"`
	DefaultUser    string `bson:"default_user,omitempty" json:"default_user"`
	DefaultPass    string `bson:"default_password,omitempty" json:"default_password"`
}

type LabInstance struct {
	ID         string          `bson:"_id,omitempty" json:"id"`
	TemplateID string          `bson:"template_id" json:"template_id"`
	LabName    string          `bson:"lab_name" json:"lab_name"`
	OwnerUUID  string          `bson:"owner_uuid" json:"owner_uuid"`
	OwnerEmail string          `bson:"owner_email,omitempty" json:"owner_email"`
	Status     string          `bson:"status" json:"status"`
	VMs        []LabInstanceVM `bson:"vms" json:"vms"`
	CreatedAt  time.Time       `bson:"created_at" json:"created_at"`
}
