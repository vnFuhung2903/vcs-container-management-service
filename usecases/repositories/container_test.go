package repositories

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/vnFuhung2903/vcs-container-management-service/dto"
	"github.com/vnFuhung2903/vcs-container-management-service/entities"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type ContainerRepoSuite struct {
	suite.Suite
	db   *gorm.DB
	repo IContainerRepository
}

func (suite *ContainerRepoSuite) SetupTest() {
	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	assert.NoError(suite.T(), err)
	err = gormDB.AutoMigrate(&entities.Container{})
	assert.NoError(suite.T(), err)
	suite.db = gormDB
	suite.repo = NewContainerRepository(gormDB)
}

func (suite *ContainerRepoSuite) TearDownTest() {
	sqlDB, err := suite.db.DB()
	assert.NoError(suite.T(), err)
	sqlDB.Close()
}

func TestContainerRepoSuite(t *testing.T) {
	suite.Run(t, new(ContainerRepoSuite))
}

func (suite *ContainerRepoSuite) TestCreateDuplicateContainerId() {
	_, err := suite.repo.Create("dup-id", "Name1", entities.ContainerOn, "10.0.1.1")
	assert.NoError(suite.T(), err)
	_, err = suite.repo.Create("dup-id", "Name2", entities.ContainerOff, "10.0.1.2")
	assert.Error(suite.T(), err)
}

func (suite *ContainerRepoSuite) TestCreateDuplicateContainerName() {
	_, err := suite.repo.Create("id1", "dup-name", entities.ContainerOn, "10.0.2.1")
	assert.NoError(suite.T(), err)
	_, err = suite.repo.Create("id2", "dup-name", entities.ContainerOff, "10.0.2.2")
	assert.Error(suite.T(), err)
}

func (suite *ContainerRepoSuite) TestCreateInBatches() {
	container1 := &entities.Container{ContainerId: "id1", ContainerName: "Name1", Status: entities.ContainerOn, Ipv4: "10.0.3.1"}
	container2 := &entities.Container{ContainerId: "id2", ContainerName: "Name2", Status: entities.ContainerOff, Ipv4: ""}

	containers := []*entities.Container{
		container1,
		container2,
	}
	err := suite.repo.CreateInBatches(containers)
	assert.NoError(suite.T(), err)
}

func (suite *ContainerRepoSuite) TestCreateAndFindById() {
	c, err := suite.repo.Create("cid-1", "Alpha", entities.ContainerOn, "10.0.0.1")
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), c)
	found, err := suite.repo.FindById("cid-1")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "cid-1", found.ContainerId)
}

func (suite *ContainerRepoSuite) TestFindByIdNotFound() {
	_, err := suite.repo.FindById("not-exist-id")
	assert.Error(suite.T(), err)
}

func (suite *ContainerRepoSuite) TestFindByName() {
	_, err := suite.repo.Create("cid-2", "Beta", entities.ContainerOff, "10.0.0.2")
	assert.NoError(suite.T(), err)
	found, err := suite.repo.FindByName("Beta")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Beta", found.ContainerName)
}

func (suite *ContainerRepoSuite) TestFindByNameNotFound() {
	_, err := suite.repo.FindByName("not-exist-name")
	assert.Error(suite.T(), err)
}

func (suite *ContainerRepoSuite) TestViewWithFilters() {
	_, _ = suite.repo.Create("cid-3", "Gamma", entities.ContainerOn, "10.0.0.3")
	_, _ = suite.repo.Create("cid-4", "Delta", entities.ContainerOff, "10.0.0.4")

	// ContainerId filter
	filter := dto.ContainerFilter{ContainerId: "cid-3"}
	sort := dto.ContainerSort{Field: "container_id", Order: "asc"}
	result, total, err := suite.repo.View(filter, 1, 10, sort)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(1), total)
	assert.Equal(suite.T(), "cid-3", result[0].ContainerId)

	// Status filter
	filter = dto.ContainerFilter{Status: entities.ContainerOff}
	result, total, err = suite.repo.View(filter, 1, 10, sort)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(1), total)
	assert.Equal(suite.T(), "cid-4", result[0].ContainerId)

	// ContainerName filter
	filter = dto.ContainerFilter{ContainerName: "Gamma"}
	result, total, err = suite.repo.View(filter, 1, 10, sort)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(1), total)
	assert.Equal(suite.T(), "cid-3", result[0].ContainerId)

	// Ipv4 filter
	filter = dto.ContainerFilter{Ipv4: "10.0.0.4"}
	result, total, err = suite.repo.View(filter, 1, 10, sort)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(1), total)
	assert.Equal(suite.T(), "cid-4", result[0].ContainerId)

	// Multiple filters
	filter = dto.ContainerFilter{ContainerId: "cid-3", Ipv4: "10.0.0.4"}
	_, total, err = suite.repo.View(filter, 1, 10, sort)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(0), total)
}

func (suite *ContainerRepoSuite) TestViewDefaultNoLimit() {
	_, _ = suite.repo.Create("cid-5", "Epsilon", entities.ContainerOn, "10.0.0.5")
	_, _ = suite.repo.Create("cid-6", "Stigma", entities.ContainerOff, "10.0.0.6")

	filter := dto.ContainerFilter{}
	sort := dto.ContainerSort{Field: "container_id", Order: "asc"}
	results, total, err := suite.repo.View(filter, 1, -1, sort)

	assert.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), total, int64(2))
	assert.Len(suite.T(), results, int(total))
}

func (suite *ContainerRepoSuite) TestViewWithInvalidSort() {
	_, _, err := suite.repo.View(dto.ContainerFilter{}, 1, 10, dto.ContainerSort{Field: "not_a_field", Order: "asc"})
	assert.Error(suite.T(), err)
	_, _, err = suite.repo.View(dto.ContainerFilter{}, 1, 10, dto.ContainerSort{Field: "container_id", Order: "invalid_order"})
	assert.Error(suite.T(), err)
	_, _, err = suite.repo.View(dto.ContainerFilter{}, 1, 10, dto.ContainerSort{})
	assert.Error(suite.T(), err)
}

func (suite *ContainerRepoSuite) TestViewWhileDbClose() {
	sqlDB, err := suite.db.DB()
	assert.NoError(suite.T(), err)
	sqlDB.Close()

	filter := dto.ContainerFilter{}
	sort := dto.ContainerSort{Field: "container_id", Order: "asc"}

	_, _, err = suite.repo.View(filter, 1, 10, sort)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "database is closed")
}

func (suite *ContainerRepoSuite) TestUpdate() {
	_, _ = suite.repo.Create("cid-7", "Zeta", entities.ContainerOn, "10.0.0.7")
	err := suite.repo.Update("cid-7", entities.ContainerOff, "")
	assert.NoError(suite.T(), err)
	found, _ := suite.repo.FindById("cid-7")
	assert.Equal(suite.T(), entities.ContainerOff, found.Status)
	assert.Equal(suite.T(), "", found.Ipv4)
	assert.Equal(suite.T(), "Zeta", found.ContainerName)
}

func (suite *ContainerRepoSuite) TestUpdateAndDeleteNonExistent() {
	err := suite.repo.Update("not-exist", entities.ContainerOff, "")
	assert.NoError(suite.T(), err)
	err = suite.repo.Delete("not-exist")
	assert.NoError(suite.T(), err)
}

func (suite *ContainerRepoSuite) TestDelete() {
	_, _ = suite.repo.Create("cid-9", "Theta", entities.ContainerOn, "10.0.0.9")
	err := suite.repo.Delete("cid-9")
	assert.NoError(suite.T(), err)
	_, err = suite.repo.FindById("cid-9")
	assert.Error(suite.T(), err)
}

func (suite *ContainerRepoSuite) TestBeginAndWithTransaction() {
	tx, err := suite.repo.BeginTransaction(suite.T().Context())
	assert.NoError(suite.T(), err)
	txRepo := suite.repo.WithTransaction(tx)
	_, err = txRepo.Create("cid-10", "Iota", entities.ContainerOn, "10.0.0.10")
	assert.NoError(suite.T(), err)
	tx.Rollback()
	_, err = suite.repo.FindById("cid-10")
	assert.Error(suite.T(), err)
}

func (suite *ContainerRepoSuite) TestBeginTransactionError() {
	sqlDB, _ := suite.db.DB()
	sqlDB.Close()
	_, err := suite.repo.BeginTransaction(suite.T().Context())
	assert.Error(suite.T(), err)
}
