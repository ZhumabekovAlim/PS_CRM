package services

import (
	"database/sql"
	"errors"
	"fmt"
	"ps_club_backend/internal/models"
	"ps_club_backend/internal/repositories"
	"strings"
	// "time" // Not directly used in DTOs, but will be in method implementations
)

// --- Custom Service Errors for Pricelist ---
var (
	ErrCategoryNotFound    = errors.New("category not found")
	ErrCategoryNameExists  = errors.New("category name already exists")
	ErrItemNotFound        = errors.New("pricelist item not found")
	ErrItemNameConflict    = errors.New("item name/SKU conflict") // More generic for SKU or name within category
	ErrValidation          = errors.New("validation error")      // Generic validation error
	ErrPricelistForeignKey = errors.New("operation failed due to existing references (e.g., category in use by items, or item in use by orders)")
)

// --- Category DTOs ---
type CreatePricelistCategoryRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description"`
}
type UpdatePricelistCategoryRequest struct {
	Name        *string `json:"name"` // Pointer to distinguish between empty and not provided
	Description *string `json:"description"`
}

// --- Item DTOs ---
type CreatePricelistItemRequest struct {
	CategoryID        int64    `json:"category_id" binding:"required"`
	Name              string   `json:"name" binding:"required"`
	Description       *string  `json:"description"`
	Price             float64  `json:"price" binding:"required,gt=0"`
	SKU               *string  `json:"sku"`
	IsAvailable       bool     `json:"is_available"` // Defaults to false (Go default) if not in JSON
	ItemType          string   `json:"item_type" binding:"required"`
	TracksStock       bool     `json:"tracks_stock"` // Defaults to false (Go default) if not in JSON
	CurrentStock      *int     `json:"current_stock"`
	LowStockThreshold *int     `json:"low_stock_threshold"`
}
type UpdatePricelistItemRequest struct {
	CategoryID        *int64   `json:"category_id"`
	Name              *string  `json:"name"`
	Description       *string  `json:"description"`
	Price             *float64 `json:"price,omitempty,gt=0"`
	SKU               *string  `json:"sku"`
	IsAvailable       *bool    `json:"is_available"`
	ItemType          *string  `json:"item_type"`
	TracksStock       *bool    `json:"tracks_stock"`
	CurrentStock      *int     `json:"current_stock"`
	LowStockThreshold *int     `json:"low_stock_threshold"`
}

// --- PricelistService Interface ---
type PricelistService interface {
	CreateCategory(req CreatePricelistCategoryRequest) (*models.PricelistCategory, error)
	GetCategoryByID(categoryID int64) (*models.PricelistCategory, error)
	GetCategories(page, pageSize int) ([]models.PricelistCategory, int, error)
	UpdateCategory(categoryID int64, req UpdatePricelistCategoryRequest) (*models.PricelistCategory, error)
	DeleteCategory(categoryID int64) error

	CreateItem(req CreatePricelistItemRequest) (*models.PricelistItem, error)
	GetItemByID(itemID int64) (*models.PricelistItem, error)
	GetItems(categoryID *int64, itemType *string, page, pageSize int) ([]models.PricelistItem, int, error)
	UpdateItem(itemID int64, req UpdatePricelistItemRequest) (*models.PricelistItem, error)
	DeleteItem(itemID int64) error
}

// --- pricelistService Implementation ---
type pricelistService struct {
	pricelistRepo repositories.PricelistRepository
	db            *sql.DB
}

func NewPricelistService(repo repositories.PricelistRepository, db *sql.DB) PricelistService {
	return &pricelistService{
		pricelistRepo: repo,
		db:            db,
	}
}

// --- Category Method Implementations ---

func (s *pricelistService) CreateCategory(req CreatePricelistCategoryRequest) (*models.PricelistCategory, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("%w: category name cannot be empty", ErrValidation)
	}
	category := &models.PricelistCategory{
		Name:        req.Name,
		Description: req.Description,
	}
	id, err := s.pricelistRepo.CreateCategory(s.db, category)
	if err != nil {
		if errors.Is(err, repositories.ErrDuplicateKey) {
			return nil, fmt.Errorf("%w: %s", ErrCategoryNameExists, err.Error())
		}
		return nil, fmt.Errorf("failed to create category: %w", err)
	}
	// Fetch to get timestamps and confirm creation
	return s.pricelistRepo.GetCategoryByID(id)
}

func (s *pricelistService) GetCategoryByID(categoryID int64) (*models.PricelistCategory, error) {
	category, err := s.pricelistRepo.GetCategoryByID(categoryID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrCategoryNotFound
		}
		return nil, fmt.Errorf("failed to get category by ID: %w", err)
	}
	return category, nil
}

func (s *pricelistService) GetCategories(page, pageSize int) ([]models.PricelistCategory, int, error) {
	if page <= 0 { page = 1 }
	if pageSize <= 0 { pageSize = 10 }
	
	categories, totalCount, err := s.pricelistRepo.GetCategories(page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get categories: %w", err)
	}
	return categories, totalCount, nil
}

func (s *pricelistService) UpdateCategory(categoryID int64, req UpdatePricelistCategoryRequest) (*models.PricelistCategory, error) {
	category, err := s.pricelistRepo.GetCategoryByID(categoryID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrCategoryNotFound
		}
		return nil, fmt.Errorf("failed to find category for update: %w", err)
	}

	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			return nil, fmt.Errorf("%w: category name cannot be empty if provided", ErrValidation)
		}
		category.Name = *req.Name
	}
	if req.Description != nil { // Allows setting description to empty string if desired
		category.Description = req.Description
	}

	err = s.pricelistRepo.UpdateCategory(s.db, category)
	if err != nil {
		if errors.Is(err, repositories.ErrDuplicateKey) {
			return nil, fmt.Errorf("%w: %s", ErrCategoryNameExists, err.Error())
		}
		if errors.Is(err, repositories.ErrNotFound) { // Should be caught by GetCategoryByID
			return nil, ErrCategoryNotFound
		}
		return nil, fmt.Errorf("failed to update category: %w", err)
	}
	return s.pricelistRepo.GetCategoryByID(categoryID)
}

func (s *pricelistService) DeleteCategory(categoryID int64) error {
	_, err := s.pricelistRepo.GetCategoryByID(categoryID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ErrCategoryNotFound
		}
		return fmt.Errorf("failed to find category for deletion: %w", err)
	}

	err = s.pricelistRepo.DeleteCategory(s.db, categoryID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ErrCategoryNotFound
		}
		if strings.Contains(err.Error(), "is currently in use by") || strings.Contains(err.Error(), "violates foreign key constraint"){
			return fmt.Errorf("%w: category cannot be deleted as it's referenced by other records", ErrPricelistForeignKey)
		}
		return fmt.Errorf("failed to delete category: %w", err)
	}
	return nil
}

// --- Item Method Implementations ---

func (s *pricelistService) CreateItem(req CreatePricelistItemRequest) (*models.PricelistItem, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("%w: item name cannot be empty", ErrValidation)
	}
	if req.TracksStock && req.CurrentStock == nil {
		zeroStock := 0
		req.CurrentStock = &zeroStock // Default to 0 if tracking stock and not provided
	}
	if !req.TracksStock { // If not tracking, ensure stock fields are nil to avoid confusion/errors
		req.CurrentStock = nil
		req.LowStockThreshold = nil
	}
	if req.TracksStock && req.CurrentStock != nil && *req.CurrentStock < 0 {
		return nil, fmt.Errorf("%w: current stock cannot be negative", ErrValidation)
	}
	if req.LowStockThreshold != nil && *req.LowStockThreshold < 0 {
		return nil, fmt.Errorf("%w: low stock threshold cannot be negative", ErrValidation)
	}


	// Check if category exists
	_, err := s.pricelistRepo.GetCategoryByID(req.CategoryID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, fmt.Errorf("%w: category with ID %d not found", ErrCategoryNotFound, req.CategoryID)
		}
		return nil, fmt.Errorf("failed to validate category for item creation: %w", err)
	}


	item := &models.PricelistItem{
		CategoryID:        req.CategoryID,
		Name:              req.Name,
		Description:       req.Description,
		Price:             req.Price,
		SKU:               req.SKU,
		IsAvailable:       req.IsAvailable,
		ItemType:          req.ItemType,
		TracksStock:       req.TracksStock,
		CurrentStock:      req.CurrentStock,
		LowStockThreshold: req.LowStockThreshold,
	}

	id, err := s.pricelistRepo.CreateItem(s.db, item)
	if err != nil {
		if errors.Is(err, repositories.ErrDuplicateKey) {
			return nil, fmt.Errorf("%w: %s", ErrItemNameConflict, err.Error())
		}
		if strings.Contains(err.Error(), "pricelist_items_category_id_fkey") {
			return nil, fmt.Errorf("%w: category with ID %d not found for item", ErrCategoryNotFound, req.CategoryID)
		}
		return nil, fmt.Errorf("failed to create item: %w", err)
	}
	return s.pricelistRepo.GetItemByID(id)
}

func (s *pricelistService) GetItemByID(itemID int64) (*models.PricelistItem, error) {
	item, err := s.pricelistRepo.GetItemByID(itemID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrItemNotFound
		}
		return nil, fmt.Errorf("failed to get item by ID: %w", err)
	}
	return item, nil
}

func (s *pricelistService) GetItems(categoryID *int64, itemType *string, page, pageSize int) ([]models.PricelistItem, int, error) {
	if page <= 0 { page = 1 }
	if pageSize <= 0 { pageSize = 10 }

	items, totalCount, err := s.pricelistRepo.GetItems(categoryID, itemType, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get items: %w", err)
	}
	return items, totalCount, nil
}

func (s *pricelistService) UpdateItem(itemID int64, req UpdatePricelistItemRequest) (*models.PricelistItem, error) {
	item, err := s.pricelistRepo.GetItemByID(itemID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrItemNotFound
		}
		return nil, fmt.Errorf("failed to find item for update: %w", err)
	}

	if req.CategoryID != nil {
		// Validate new category if provided
		_, catErr := s.pricelistRepo.GetCategoryByID(*req.CategoryID)
		if catErr != nil {
			if errors.Is(catErr, repositories.ErrNotFound) {
				return nil, fmt.Errorf("%w: new category with ID %d not found", ErrCategoryNotFound, *req.CategoryID)
			}
			return nil, fmt.Errorf("failed to validate new category for item update: %w", catErr)
		}
		item.CategoryID = *req.CategoryID
	}

	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			return nil, fmt.Errorf("%w: item name cannot be empty if provided", ErrValidation)
		}
		item.Name = *req.Name
	}
	if req.Description != nil { item.Description = req.Description }
	if req.Price != nil { item.Price = *req.Price }
	if req.SKU != nil { item.SKU = req.SKU } // SKU can be set to empty string
	if req.IsAvailable != nil { item.IsAvailable = *req.IsAvailable }
	if req.ItemType != nil { item.ItemType = *req.ItemType }

	// Handle TracksStock logic
	if req.TracksStock != nil {
		item.TracksStock = *req.TracksStock
		if !item.TracksStock { // If changing to not track stock
			item.CurrentStock = nil
			item.LowStockThreshold = nil
		} else { // If changing to track stock (or already tracking)
			if req.CurrentStock != nil { // If new stock value is provided
				if *req.CurrentStock < 0 { return nil, fmt.Errorf("%w: current stock cannot be negative", ErrValidation) }
				item.CurrentStock = req.CurrentStock
			} else if item.CurrentStock == nil { // If now tracking and stock was nil, default to 0
				zeroStock := 0
				item.CurrentStock = &zeroStock
			}
		}
	} else if req.CurrentStock != nil { // TracksStock not in request, but CurrentStock is
		if !item.TracksStock {
			return nil, fmt.Errorf("%w: cannot update CurrentStock for an item that does not track stock, unless TracksStock is also updated to true", ErrValidation)
		}
		if *req.CurrentStock < 0 { return nil, fmt.Errorf("%w: current stock cannot be negative", ErrValidation) }
		item.CurrentStock = req.CurrentStock
	}

	if req.LowStockThreshold != nil {
		if !item.TracksStock {
			return nil, fmt.Errorf("%w: cannot set LowStockThreshold for an item that does not track stock", ErrValidation)
		}
		if *req.LowStockThreshold < 0 { return nil, fmt.Errorf("%w: low stock threshold cannot be negative", ErrValidation) }
		item.LowStockThreshold = req.LowStockThreshold
	} else if req.TracksStock != nil && !*req.TracksStock { // If TracksStock is being set to false
		item.LowStockThreshold = nil // Ensure it's cleared
	}


	err = s.pricelistRepo.UpdateItem(s.db, item)
	if err != nil {
		if errors.Is(err, repositories.ErrDuplicateKey) {
			return nil, fmt.Errorf("%w: %s", ErrItemNameConflict, err.Error())
		}
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrItemNotFound // Should have been caught by GetItemByID
		}
		if strings.Contains(err.Error(), "pricelist_items_category_id_fkey") {
			return nil, fmt.Errorf("%w: category with ID %d not found for item", ErrCategoryNotFound, item.CategoryID)
		}
		return nil, fmt.Errorf("failed to update item: %w", err)
	}
	return s.pricelistRepo.GetItemByID(itemID)
}

func (s *pricelistService) DeleteItem(itemID int64) error {
	_, err := s.pricelistRepo.GetItemByID(itemID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ErrItemNotFound
		}
		return fmt.Errorf("failed to find item for deletion: %w", err)
	}

	err = s.pricelistRepo.DeleteItem(s.db, itemID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ErrItemNotFound
		}
		if strings.Contains(err.Error(), "is referenced by other records") || strings.Contains(err.Error(), "violates foreign key constraint") {
			return fmt.Errorf("%w: item cannot be deleted as it's referenced by other records", ErrPricelistForeignKey)
		}
		return fmt.Errorf("failed to delete item: %w", err)
	}
	return nil
}
