package entities

import (
	"time"
)

type Container struct {
	ContainerId   string          `gorm:"primaryKey"`
	Status        ContainerStatus `gorm:"type:varchar(10);not null;index:idx_status,priority:1"`
	CreatedAt     time.Time       `gorm:"autoCreateTime"`
	UpdatedAt     time.Time       `gorm:"autoUpdateTime"`
	ContainerName string          `gorm:"index:idx_container_name,priority:2;not null;unique"`
	Ipv4          string          `gorm:"not null"`
}

type ContainerStatus string

const (
	ContainerOn  ContainerStatus = "ON"
	ContainerOff ContainerStatus = "OFF"
)

type ContainerWithStatus struct {
	ContainerId string          `json:"container_id"`
	Status      ContainerStatus `json:"status"`
}
