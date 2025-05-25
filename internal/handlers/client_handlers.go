package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"ps_club_backend/internal/services"
	"ps_club_backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

// ClientHandler holds the client service.
type ClientHandler struct {
	clientService services.ClientService
}

// NewClientHandler creates a new ClientHandler.
func NewClientHandler(cs services.ClientService) *ClientHandler {
	return &ClientHandler{clientService: cs}
}

// CreateClient handles the creation of a new client.
func (h *ClientHandler) CreateClient(c *gin.Context) {
	var req services.CreateClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.LogError(err, "CreateClient: Failed to bind JSON")
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid request payload: "+err.Error(), err.Error()))
		return
	}

	client, err := h.clientService.CreateClient(req)
	if err != nil {
		utils.LogError(err, "CreateClient: Error from clientService.CreateClient")
		if errors.Is(err, services.ErrPhoneNumberExists) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusConflict, utils.ErrCodeConflict, "Phone number already exists.", err.Error()))
		} else if errors.Is(err, services.ErrEmailExists) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusConflict, utils.ErrCodeConflict, "Email already exists.", err.Error()))
		} else if errors.Is(err, services.ErrClientValidation) || errors.Is(err, services.ErrDateFormat){
			utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Validation failed: "+err.Error(), err.Error()))
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to create client.", "Internal error"))
		}
		return
	}
	c.JSON(http.StatusCreated, client)
}

// GetClients handles fetching all clients with pagination and search.
func (h *ClientHandler) GetClients(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	searchTerm := c.Query("search")

	if page <= 0 { page = 1 }
	if pageSize <= 0 { pageSize = 10 }
	
	var pSearchTerm *string
	if searchTerm != "" {
		pSearchTerm = &searchTerm
	}

	clients, totalCount, err := h.clientService.GetClients(page, pageSize, pSearchTerm)
	if err != nil {
		utils.LogError(err, "GetClients: Error from clientService.GetClients")
		utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to fetch clients.", "Internal error"))
		return
	}
	
	if clients == nil {
	    clients = []models.Client{}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  clients,
		"total": totalCount,
		"page":  page,
		"page_size": pageSize,
	})
}

// GetClientByID handles fetching a single client by ID.
func (h *ClientHandler) GetClientByID(c *gin.Context) {
	idStr := c.Param("id")
	clientID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid client ID format.", err.Error()))
		return
	}

	client, err := h.clientService.GetClientByID(clientID)
	if err != nil {
		utils.LogError(err, "GetClientByID: Error from clientService.GetClientByID for ID "+idStr)
		if errors.Is(err, services.ErrClientNotFound) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusNotFound, utils.ErrCodeNotFound, "Client not found.", err.Error()))
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to fetch client.", "Internal error"))
		}
		return
	}
	c.JSON(http.StatusOK, client)
}

// UpdateClient handles updating a client.
func (h *ClientHandler) UpdateClient(c *gin.Context) {
	idStr := c.Param("id")
	clientID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid client ID format.", err.Error()))
		return
	}

	var req services.UpdateClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.LogError(err, "UpdateClient: Failed to bind JSON for ID "+idStr)
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid request payload: "+err.Error(), err.Error()))
		return
	}

	client, err := h.clientService.UpdateClient(clientID, req)
	if err != nil {
		utils.LogError(err, "UpdateClient: Error from clientService.UpdateClient for ID "+idStr)
		if errors.Is(err, services.ErrClientNotFound) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusNotFound, utils.ErrCodeNotFound, "Client not found to update.", err.Error()))
		} else if errors.Is(err, services.ErrPhoneNumberExists) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusConflict, utils.ErrCodeConflict, "Phone number already exists.", err.Error()))
		} else if errors.Is(err, services.ErrEmailExists) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusConflict, utils.ErrCodeConflict, "Email already exists.", err.Error()))
		} else if errors.Is(err, services.ErrClientValidation) || errors.Is(err, services.ErrDateFormat){
			utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Validation failed: "+err.Error(), err.Error()))
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to update client.", "Internal error"))
		}
		return
	}
	c.JSON(http.StatusOK, client)
}

// DeleteClient handles deleting a client.
func (h *ClientHandler) DeleteClient(c *gin.Context) {
	idStr := c.Param("id")
	clientID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid client ID format.", err.Error()))
		return
	}

	err = h.clientService.DeleteClient(clientID)
	if err != nil {
		utils.LogError(err, "DeleteClient: Error from clientService.DeleteClient for ID "+idStr)
		if errors.Is(err, services.ErrClientNotFound) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusNotFound, utils.ErrCodeNotFound, "Client not found to delete.", err.Error()))
		} else if errors.Is(err, services.ErrClientInUse) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusConflict, utils.ErrCodeConflict, "Client cannot be deleted as they are referenced in other records.", err.Error()))
		}else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to delete client.", "Internal error"))
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Client deleted successfully"})
}

// Remove or comment out old standalone functions if they existed, e.g.:
// func CreateClient(c *gin.Context) { /* ... */ }
// func GetClients(c *gin.Context) { /* ... */ }
// ... etc. ...
