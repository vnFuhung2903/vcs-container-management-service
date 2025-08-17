package repositories

import (
	"context"
	"fmt"

	"github.com/vnFuhung2903/vcs-container-management-service/dto"
	"github.com/vnFuhung2903/vcs-container-management-service/entities"
	"gorm.io/gorm"
)

type IContainerRepository interface {
	FindById(containerId string) (*entities.Container, error)
	FindByName(containerName string) (*entities.Container, error)
	View(filter dto.ContainerFilter, from int, limit int, sort dto.ContainerSort) ([]*entities.Container, int64, error)
	Create(containerId string, containerName string, status entities.ContainerStatus, ipv4 string) (*entities.Container, error)
	CreateInBatches(containers []*entities.Container) error
	Update(containerId string, status entities.ContainerStatus, ipv4 string) error
	Delete(containerId string) error
	BeginTransaction(ctx context.Context) (*gorm.DB, error)
	WithTransaction(tx *gorm.DB) IContainerRepository
}

type containerRepository struct {
	db *gorm.DB
}

func NewContainerRepository(db *gorm.DB) IContainerRepository {
	return &containerRepository{db: db}
}

func (r *containerRepository) FindById(containerId string) (*entities.Container, error) {
	var container entities.Container
	res := r.db.First(&container, entities.Container{ContainerId: containerId})
	if res.Error != nil {
		return nil, res.Error
	}
	return &container, nil
}

func (r *containerRepository) FindByName(containerName string) (*entities.Container, error) {
	var container entities.Container
	res := r.db.First(&container, entities.Container{ContainerName: containerName})
	if res.Error != nil {
		return nil, res.Error
	}
	return &container, nil
}

func (r *containerRepository) View(filter dto.ContainerFilter, from int, limit int, sort dto.ContainerSort) ([]*entities.Container, int64, error) {
	query := r.db.Model(entities.Container{})

	if filter.ContainerId != "" {
		query = query.Where("container_id = ?", filter.ContainerId)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.ContainerName != "" {
		query = query.Where("container_name LIKE ?", "%"+filter.ContainerName+"%")
	}
	if filter.Ipv4 != "" {
		query = query.Where("ipv4 = ?", filter.Ipv4)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query = query.Order(fmt.Sprintf("%s %s", sort.Field, sort.Order))

	var containers []*entities.Container
	if err := query.Limit(limit).Offset(from - 1).Find(&containers).Error; err != nil {
		return nil, 0, err
	}
	return containers, total, nil
}

func (r *containerRepository) Create(containerId string, containerName string, status entities.ContainerStatus, ipv4 string) (*entities.Container, error) {
	newContainer := &entities.Container{
		ContainerId:   containerId,
		Status:        status,
		ContainerName: containerName,
		Ipv4:          ipv4,
	}
	res := r.db.Create(newContainer)
	if res.Error != nil {
		return nil, res.Error
	}
	return newContainer, nil
}

func (r *containerRepository) CreateInBatches(containers []*entities.Container) error {
	res := r.db.CreateInBatches(&containers, 10000)
	return res.Error
}

func (r *containerRepository) Update(containerId string, status entities.ContainerStatus, ipv4 string) error {
	updateData := map[string]interface{}{
		"status": status,
		"ipv4":   ipv4,
	}
	res := r.db.Model(&entities.Container{}).Where("container_id = ?", containerId).Updates(updateData)
	return res.Error
}

func (r *containerRepository) Delete(containerId string) error {
	res := r.db.Where("container_id = ?", containerId).Delete(&entities.Container{})
	return res.Error
}

func (r *containerRepository) BeginTransaction(ctx context.Context) (*gorm.DB, error) {
	tx := r.db.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	return tx, nil
}

func (r *containerRepository) WithTransaction(tx *gorm.DB) IContainerRepository {
	return &containerRepository{db: tx}
}
