package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"ps_club_backend/internal/services"
	"ps_club_backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

// PricelistHandler holds the pricelist service.
type PricelistHandler struct {
	pricelistService services.PricelistService
}

// NewPricelistHandler creates a new PricelistHandler.
func NewPricelistHandler(ps services.PricelistService) *PricelistHandler {
	return &PricelistHandler{pricelistService: ps}
}

// --- PricelistCategory Handler Methods ---

// CreatePricelistCategory handles the creation of a new pricelist category.
func (h *PricelistHandler) CreatePricelistCategory(c *gin.Context) {
	var req services.CreatePricelistCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.LogError(err, "CreatePricelistCategory: Failed to bind JSON")
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid request payload: "+err.Error(), err.Error()))
		return
	}

	category, err := h.pricelistService.CreateCategory(req)
	if err != nil {
		utils.LogError(err, "CreatePricelistCategory: Error from pricelistService.CreateCategory")
		if errors.Is(err, services.ErrCategoryNameExists) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusConflict, utils.ErrCodeConflict, "Category name already exists.", err.Error()))
		} else if errors.Is(err, services.ErrValidation) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Validation failed: "+err.Error(), err.Error()))
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to create category.", "Internal error"))
		}
		return
	}
	c.JSON(http.StatusCreated, category)
}

// GetPricelistCategories handles fetching all pricelist categories with pagination.
func (h *PricelistHandler) GetPricelistCategories(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page <= 0 { page = 1 }
	if pageSize <= 0 { pageSize = 10 }


	categories, totalCount, err := h.pricelistService.GetCategories(page, pageSize)
	if err != nil {
		utils.LogError(err, "GetPricelistCategories: Error from pricelistService.GetCategories")
		utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to fetch categories.", "Internal error"))
		return
	}
	
	if categories == nil { // Ensure response is an empty list, not null
	    categories = []models.PricelistCategory{}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  categories,
		"total": totalCount,
		"page":  page,
		"page_size": pageSize,
	})
}

// GetPricelistCategoryByID handles fetching a single pricelist category by ID.
func (h *PricelistHandler) GetPricelistCategoryByID(c *gin.Context) {
	idStr := c.Param("id")
	categoryID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid category ID format.", err.Error()))
		return
	}

	category, err := h.pricelistService.GetCategoryByID(categoryID)
	if err != nil {
		utils.LogError(err, "GetPricelistCategoryByID: Error from pricelistService.GetCategoryByID for ID "+idStr)
		if errors.Is(err, services.ErrCategoryNotFound) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusNotFound, utils.ErrCodeNotFound, "Category not found.", err.Error()))
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to fetch category.", "Internal error"))
		}
		return
	}
	c.JSON(http.StatusOK, category)
}

// UpdatePricelistCategory handles updating a pricelist category.
func (h *PricelistHandler) UpdatePricelistCategory(c *gin.Context) {
	idStr := c.Param("id")
	categoryID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid category ID format.", err.Error()))
		return
	}

	var req services.UpdatePricelistCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.LogError(err, "UpdatePricelistCategory: Failed to bind JSON for ID "+idStr)
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid request payload: "+err.Error(), err.Error()))
		return
	}

	category, err := h.pricelistService.UpdateCategory(categoryID, req)
	if err != nil {
		utils.LogError(err, "UpdatePricelistCategory: Error from pricelistService.UpdateCategory for ID "+idStr)
		if errors.Is(err, services.ErrCategoryNotFound) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusNotFound, utils.ErrCodeNotFound, "Category not found to update.", err.Error()))
		} else if errors.Is(err, services.ErrCategoryNameExists) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusConflict, utils.ErrCodeConflict, "Category name already exists.", err.Error()))
		} else if errors.Is(err, services.ErrValidation) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Validation failed: "+err.Error(), err.Error()))
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to update category.", "Internal error"))
		}
		return
	}
	c.JSON(http.StatusOK, category)
}

// DeletePricelistCategory handles deleting a pricelist category.
func (h *PricelistHandler) DeletePricelistCategory(c *gin.Context) {
	idStr := c.Param("id")
	categoryID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid category ID format.", err.Error()))
		return
	}

	err = h.pricelistService.DeleteCategory(categoryID)
	if err != nil {
		utils.LogError(err, "DeletePricelistCategory: Error from pricelistService.DeleteCategory for ID "+idStr)
		if errors.Is(err, services.ErrCategoryNotFound) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusNotFound, utils.ErrCodeNotFound, "Category not found to delete.", err.Error()))
		} else if errors.Is(err, services.ErrPricelistForeignKey) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusConflict, utils.ErrCodeConflict, "Cannot delete category: it is currently in use.", err.Error()))
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to delete category.", "Internal error"))
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Pricelist category deleted successfully"})
}

// --- PricelistItem Handler Methods ---

// CreatePricelistItem handles the creation of a new pricelist item.
func (h *PricelistHandler) CreatePricelistItem(c *gin.Context) {
	var req services.CreatePricelistItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.LogError(err, "CreatePricelistItem: Failed to bind JSON")
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid request payload: "+err.Error(), err.Error()))
		return
	}

	item, err := h.pricelistService.CreateItem(req)
	if err != nil {
		utils.LogError(err, "CreatePricelistItem: Error from pricelistService.CreateItem")
		if errors.Is(err, services.ErrItemNameConflict) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusConflict, utils.ErrCodeConflict, "Item name or SKU already exists or conflicts.", err.Error()))
		} else if errors.Is(err, services.ErrCategoryNotFound) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeBadRequest, "Invalid category ID provided.", err.Error()))
		} else if errors.Is(err, services.ErrValidation) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Validation failed: "+err.Error(), err.Error()))
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to create item.", "Internal error"))
		}
		return
	}
	c.JSON(http.StatusCreated, item)
}

// GetPricelistItems handles fetching all pricelist items with filters and pagination.
func (h *PricelistHandler) GetPricelistItems(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	
	if page <= 0 { page = 1 }
	if pageSize <= 0 { pageSize = 10 }


	var categoryID *int64
	if categoryIDStr := c.Query("category_id"); categoryIDStr != "" {
		id, err := strconv.ParseInt(categoryIDStr, 10, 64)
		if err == nil {
			categoryID = &id
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid category_id format.", err.Error()))
			return
		}
	}

	var itemType *string
	if itemTypeStr := c.Query("item_type"); itemTypeStr != "" {
		itemType = &itemTypeStr
	}

	items, totalCount, err := h.pricelistService.GetItems(categoryID, itemType, page, pageSize)
	if err != nil {
		utils.LogError(err, "GetPricelistItems: Error from pricelistService.GetItems")
		utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to fetch items.", "Internal error"))
		return
	}
	
	if items == nil { // Ensure response is an empty list, not null
	    items = []models.PricelistItem{}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  items,
		"total": totalCount,
		"page": page,
		"page_size": pageSize,
	})
}

// GetPricelistItemByID handles fetching a single pricelist item by ID.
func (h *PricelistHandler) GetPricelistItemByID(c *gin.Context) {
	idStr := c.Param("id")
	itemID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid item ID format.", err.Error()))
		return
	}

	item, err := h.pricelistService.GetItemByID(itemID)
	if err != nil {
		utils.LogError(err, "GetPricelistItemByID: Error from pricelistService.GetItemByID for ID "+idStr)
		if errors.Is(err, services.ErrItemNotFound) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusNotFound, utils.ErrCodeNotFound, "Item not found.", err.Error()))
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to fetch item.", "Internal error"))
		}
		return
	}
	c.JSON(http.StatusOK, item)
}

// UpdatePricelistItem handles updating a pricelist item.
func (h *PricelistHandler) UpdatePricelistItem(c *gin.Context) {
	idStr := c.Param("id")
	itemID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid item ID format.", err.Error()))
		return
	}

	var req services.UpdatePricelistItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.LogError(err, "UpdatePricelistItem: Failed to bind JSON for ID "+idStr)
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid request payload: "+err.Error(), err.Error()))
		return
	}

	item, err := h.pricelistService.UpdateItem(itemID, req)
	if err != nil {
		utils.LogError(err, "UpdatePricelistItem: Error from pricelistService.UpdateItem for ID "+idStr)
		if errors.Is(err, services.ErrItemNotFound) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusNotFound, utils.ErrCodeNotFound, "Item not found to update.", err.Error()))
		} else if errors.Is(err, services.ErrItemNameConflict) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusConflict, utils.ErrCodeConflict, "Item name or SKU already exists or conflicts.", err.Error()))
		} else if errors.Is(err, services.ErrCategoryNotFound) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeBadRequest, "Invalid category ID provided.", err.Error()))
		} else if errors.Is(err, services.ErrValidation) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Validation failed: "+err.Error(), err.Error()))
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to update item.", "Internal error"))
		}
		return
	}
	c.JSON(http.StatusOK, item)
}

// DeletePricelistItem handles deleting a pricelist item.
func (h *PricelistHandler) DeletePricelistItem(c *gin.Context) {
	idStr := c.Param("id")
	itemID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid item ID format.", err.Error()))
		return
	}

	err = h.pricelistService.DeleteItem(itemID)
	if err != nil {
		utils.LogError(err, "DeletePricelistItem: Error from pricelistService.DeleteItem for ID "+idStr)
		if errors.Is(err, services.ErrItemNotFound) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusNotFound, utils.ErrCodeNotFound, "Item not found to delete.", err.Error()))
		} else if errors.Is(err, services.ErrPricelistForeignKey) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusConflict, utils.ErrCodeConflict, "Cannot delete item: it is currently in use (e.g., by an order).", err.Error()))
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to delete item.", "Internal error"))
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Pricelist item deleted successfully"})
}

// Remove or comment out old standalone functions if they existed:
// func CreatePricelistCategory(c *gin.Context) { ... }
// func GetPricelistCategories(c *gin.Context) { ... }
// ... and so on for all category and item handlers ...
// func CreatePricelistItem(c *gin.Context) { ... }
// func GetPricelistItems(c *gin.Context) { ... }
// ... etc. ...
