package types

import "time"

type Project struct {
	ID          string    `bson:"_id,omitempty" json:"id"`
	Name        string    `bson:"name" json:"name"`
	Description string    `bson:"description" json:"description"`
	Color       string    `bson:"color,omitempty" json:"color"`
	OwnerUUID   string    `bson:"owner_uuid" json:"owner_uuid"`
	OwnerEmail  string    `bson:"owner_email,omitempty" json:"owner_email"`
	VMIDs       []int     `bson:"vm_ids" json:"vm_ids"`
	CreatedAt   time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time `bson:"updated_at" json:"updated_at"`
}
