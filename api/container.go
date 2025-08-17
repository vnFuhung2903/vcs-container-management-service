package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/vnFuhung2903/vcs-container-management-service/dto"
	"github.com/vnFuhung2903/vcs-container-management-service/pkg/middlewares"
	"github.com/vnFuhung2903/vcs-container-management-service/usecases/services"
)

type ContainerHandler struct {
	containerService services.IContainerService
	jwtMiddleware    middlewares.IJWTMiddleware
}

func NewContainerHandler(containerService services.IContainerService, jwtMiddleware middlewares.IJWTMiddleware) *ContainerHandler {
	return &ContainerHandler{containerService, jwtMiddleware}
}

func (h *ContainerHandler) SetupRoutes(r *gin.Engine) {
	containerRoutes := r.Group("/containers")
	{
		createGroup := containerRoutes.Group("", h.jwtMiddleware.RequireScope("container:create"))
		{
			createGroup.POST("/create", h.Create)
			createGroup.POST("/import", h.Import)
		}

		viewGroup := containerRoutes.Group("", h.jwtMiddleware.RequireScope("container:view"))
		{
			viewGroup.GET("/view", h.View)
			viewGroup.GET("/export", h.Export)
		}

		modifyGroup := containerRoutes.Group("", h.jwtMiddleware.RequireScope("container:update"))
		{
			modifyGroup.PUT("/update/:id", h.Update)
		}

		deleteGroup := containerRoutes.Group("", h.jwtMiddleware.RequireScope("container:delete"))
		{
			deleteGroup.DELETE("/delete/:id", h.Delete)
		}
	}
}

// Create godoc
// @Summary Create a new container
// @Description Create a container with name and image
// @Tags containers
// @Accept json
// @Produce json
// @Param body body dto.CreateRequest true "Container creation request"
// @Success 201 {object} dto.APIResponse "Container created successfully"
// @Failure 400 {object} dto.APIResponse "Bad request"
// @Failure 500 {object} dto.APIResponse "Internal server error"
// @Security BearerAuth
// @Router /containers/create [post]
func (h *ContainerHandler) Create(c *gin.Context) {
	var req dto.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Code:    "BAD_REQUEST",
			Message: "Invalid request data",
			Error:   err.Error(),
		})
		return
	}

	_, err := h.containerService.Create(c.Request.Context(), req.ContainerName, req.ImageName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to create container",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, dto.APIResponse{
		Success: true,
		Code:    "CONTAINER_CREATED",
		Message: "Container created successfully",
	})
}

// View godoc
// @Summary View containers
// @Description Retrieve a list of containers with optional pagination, filtering, and sorting
// @Tags containers
// @Produce json
// @Param from query int false "From index (default 1)" default(1)
// @Param to query int false "To index (default -1 for all)" default(-1)
// @Param container_id query string false "Filter by ContainerId"
// @Param container_name query string false "Filter by ContainerName"
// @Param status query string false "Filter by Status" Enums(ON, OFF)
// @Param ipv4 query string false "Filter by IPv4"
// @Param field query string true "Sort by field" Enums(container_id, container_name, status, ipv4, created_at, updated_at)
// @Param order query string true "Sort order" Enums(asc, desc)
// @Success 200 {object} dto.APIResponse "Successful response with container list"
// @Failure 400 {object} dto.APIResponse "Bad request"
// @Failure 500 {object} dto.APIResponse "Internal server error"
// @Security BearerAuth
// @Router /containers/view [get]
func (h *ContainerHandler) View(c *gin.Context) {
	from, err := strconv.Atoi(c.DefaultQuery("from", "1"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Code:    "BAD_REQUEST",
			Message: "Invalid request data",
			Error:   err.Error(),
		})
		return
	}
	to, err := strconv.Atoi(c.DefaultQuery("to", "-1"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Code:    "BAD_REQUEST",
			Message: "Invalid request data",
			Error:   err.Error(),
		})
		return
	}

	var filter dto.ContainerFilter
	var sort dto.ContainerSort
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Code:    "BAD_REQUEST",
			Message: "Invalid request data",
			Error:   err.Error(),
		})
		return
	}

	if err := c.ShouldBindQuery(&sort); err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Code:    "BAD_REQUEST",
			Message: "Invalid request data",
			Error:   err.Error(),
		})
		return
	}

	containers, total, err := h.containerService.View(c.Request.Context(), filter, from, to, sort)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to retrieve containers",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Code:    "CONTAINERS_RETRIEVED",
		Message: "Containers retrieved successfully",
		Data: dto.ViewResponse{
			Data:  containers,
			Total: total,
		},
	})
}

// Update godoc
// @Summary Update a container
// @Description Update container information by container ID
// @Tags containers
// @Accept json
// @Produce json
// @Param id path string true "Container ID"
// @Param body body dto.ContainerUpdate true "Container update payload"
// @Success 200 {object} dto.APIResponse "Container updated successfully"
// @Failure 400 {object} dto.APIResponse "Bad request"
// @Failure 500 {object} dto.APIResponse "Internal server error"
// @Security BearerAuth
// @Router /containers/update/{id} [put]
func (h *ContainerHandler) Update(c *gin.Context) {
	containerId := c.Param("id")

	var updateData dto.ContainerUpdate
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Code:    "BAD_REQUEST",
			Message: "Invalid request data",
			Error:   err.Error(),
		})
		return
	}

	err := h.containerService.Update(c.Request.Context(), containerId, updateData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to update container",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Code:    "CONTAINER_UPDATED",
		Message: "Container updated successfully",
	})
}

// Delete godoc
// @Summary Delete a container
// @Description Delete a container by its ID
// @Tags containers
// @Produce json
// @Param id path string true "Container ID"
// @Success 200 {object} dto.APIResponse "Container deleted successfully"
// @Failure 500 {object} dto.APIResponse "Internal server error"
// @Security BearerAuth
// @Router /containers/delete/{id} [delete]
func (h *ContainerHandler) Delete(c *gin.Context) {
	containerId := c.Param("id")
	err := h.containerService.Delete(c.Request.Context(), containerId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to delete container",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Code:    "CONTAINER_DELETED",
		Message: "Container deleted successfully",
	})
}

// Import godoc
// @Summary Import containers from Excel
// @Description Import containers using an Excel (.xlsx) file
// @Tags containers
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Excel file containing container data"
// @Success 200 {object} dto.APIResponse "Import result with success and failure counts"
// @Failure 400 {object} dto.APIResponse "Bad request"
// @Failure 500 {object} dto.APIResponse "Internal server error"
// @Security BearerAuth
// @Router /containers/import [post]
func (h *ContainerHandler) Import(c *gin.Context) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Code:    "BAD_REQUEST",
			Message: "Invalid request data",
			Error:   err.Error(),
		})
		return
	}
	defer file.Close()

	result, err := h.containerService.Import(c.Request.Context(), file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to import containers",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Code:    "CONTAINERS_IMPORTED",
		Message: "Containers imported successfully",
		Data:    dto.ImportResponse(*result),
	})
}

// Export godoc
// @Summary Export containers to Excel
// @Description Export containers with optional filters and sorting to an Excel file
// @Tags containers
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param from query int false "From index (default 1)" default(1)
// @Param to query int false "To index (default -1 for all)" default(-1)
// @Param container_id query string false "Filter by ContainerId"
// @Param container_name query string false "Filter by ContainerName"
// @Param status query string false "Filter by Status" Enums(ON, OFF)
// @Param ipv4 query string false "Filter by IPv4"
// @Param field query string true "Sort by field" Enums(container_id, container_name, status, ipv4, created_at, updated_at)
// @Param order query string true "Sort order" Enums(asc, desc)
// @Success 200 {file} file "Excel file containing container data"
// @Failure 400 {object} dto.APIResponse "Bad request"
// @Failure 500 {object} dto.APIResponse "Internal server error"
// @Security BearerAuth
// @Router /containers/export [get]
func (h *ContainerHandler) Export(c *gin.Context) {
	from, err := strconv.Atoi(c.DefaultQuery("from", "1"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Code:    "BAD_REQUEST",
			Message: "Invalid request data",
			Error:   err.Error(),
		})
		return
	}
	to, err := strconv.Atoi(c.DefaultQuery("to", "-1"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Code:    "BAD_REQUEST",
			Message: "Invalid request data",
			Error:   err.Error(),
		})
		return
	}

	var filter dto.ContainerFilter
	var sort dto.ContainerSort
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Code:    "BAD_REQUEST",
			Message: "Invalid request data",
			Error:   err.Error(),
		})
		return
	}

	if err := c.ShouldBindQuery(&sort); err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Code:    "BAD_REQUEST",
			Message: "Invalid request data",
			Error:   err.Error(),
		})
		return
	}

	data, err := h.containerService.Export(c.Request.Context(), filter, from, to, sort)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to export containers",
			Error:   err.Error(),
		})
		return
	}
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", `attachment; filename="containers.xlsx"`)
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}
