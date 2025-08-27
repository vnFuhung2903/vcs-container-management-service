package dto

import (
	"github.com/vnFuhung2903/vcs-container-management-service/entities"
)

type CreateRequest struct {
	ContainerName string `json:"container_name" binding:"required"`
	ImageName     string `json:"image_name" binding:"required"`
}

type ViewResponse struct {
	Data  []*entities.Container `json:"data"`
	Total int64                 `json:"total"`
}

type ImportResponse struct {
	SuccessCount      int      `json:"success_count"`
	SuccessContainers []string `json:"success_containers"`
	FailedCount       int      `json:"failed_count"`
	FailedContainers  []string `json:"failed_containers"`
}

type ContainerUpdate struct {
	Status entities.ContainerStatus `json:"status" binding:"required,oneof=ON OFF"`
}

type ContainerFilter struct {
	ContainerId   string                   `form:"container_id" binding:"omitempty"`
	Status        entities.ContainerStatus `form:"status" binding:"omitempty,oneof=ON OFF"`
	ContainerName string                   `form:"container_name" binding:"omitempty"`
	Ipv4          string                   `form:"ipv4" binding:"omitempty"`
}

type ContainerSort struct {
	Field string    `form:"field" binding:"required,oneof=container_id container_name status ipv4 created_at updated_at"`
	Order SortOrder `form:"order" binding:"required,oneof=asc desc"`
}

var SortField = map[string]bool{
	"container_id":   true,
	"container_name": true,
	"status":         true,
	"ipv4":           true,
	"created_at":     true,
	"updated_at":     true,
}

type SortOrder string

const (
	Asc SortOrder = "asc"
	Dsc SortOrder = "desc"
)

type KafkaStatusUpdate struct {
	ContainerId string                   `json:"container_id"`
	Status      entities.ContainerStatus `json:"status"`
	Ipv4        string                   `json:"ipv4"`
}
