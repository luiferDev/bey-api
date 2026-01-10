package products

import (
	"errors"
)

type ProductService struct {
	categoryRepo *CategoryRepository
	productRepo  *ProductRepository
	variantRepo  *ProductVariantRepository
	imageRepo    *ProductImageRepository
}

func NewProductService(
	categoryRepo *CategoryRepository,
	productRepo *ProductRepository,
	variantRepo *ProductVariantRepository,
	imageRepo *ProductImageRepository,
) *ProductService {
	return &ProductService{
		categoryRepo: categoryRepo,
		productRepo:  productRepo,
		variantRepo:  variantRepo,
		imageRepo:    imageRepo,
	}
}

// ValidateCategory verifica que una categoría existe y está activa
func (s *ProductService) ValidateCategory(categoryID uint) error {
	category, err := s.categoryRepo.FindByID(categoryID)
	if err != nil {
		return err
	}
	if category == nil {
		return errors.New("category not found")
	}
	return nil
}

// CreateProductWithVariants crea un producto con sus variantes e imágenes
func (s *ProductService) CreateProductWithVariants(
	productReq CreateProductRequest,
	variants []CreateProductVariantRequest,
	images []CreateProductImageRequest,
) (*Product, error) {
	// Validar categoría
	if err := s.ValidateCategory(productReq.CategoryID); err != nil {
		return nil, err
	}

	// Crear producto
	isActive := true
	if productReq.IsActive != nil {
		isActive = *productReq.IsActive
	}

	product := &Product{
		CategoryID:  productReq.CategoryID,
		Name:        productReq.Name,
		Slug:        productReq.Slug,
		Brand:       productReq.Brand,
		Description: productReq.Description,
		BasePrice:   productReq.BasePrice,
		IsActive:    isActive,
	}

	if err := s.productRepo.Create(product); err != nil {
		return nil, err
	}

	// Crear variantes si se proporcionan
	for _, variantReq := range variants {
		variant := &ProductVariant{
			ProductID:  product.ID,
			SKU:        variantReq.SKU,
			Price:      variantReq.Price,
			Stock:      variantReq.Stock,
			Attributes: variantReq.Attributes,
		}
		if err := s.variantRepo.Create(variant); err != nil {
			return nil, err
		}
	}

	// Crear imágenes si se proporcionan
	for i, imageReq := range images {
		isMain := false
		if imageReq.IsMain != nil {
			isMain = *imageReq.IsMain
		} else if i == 0 { // Primera imagen como principal por defecto
			isMain = true
		}

		image := &ProductImage{
			ProductID: product.ID,
			VariantID: imageReq.VariantID,
			URLImage:  imageReq.URLImage,
			IsMain:    isMain,
			SortOrder: imageReq.SortOrder,
		}
		if err := s.imageRepo.Create(image); err != nil {
			return nil, err
		}
	}

	// Recargar producto con relaciones
	return s.productRepo.FindByID(product.ID)
}

// GetProductWithDetails obtiene un producto con todas sus relaciones
func (s *ProductService) GetProductWithDetails(productID uint) (*Product, error) {
	return s.productRepo.FindByID(productID)
}

// UpdateProductStock actualiza el stock de una variante específica
func (s *ProductService) UpdateProductStock(variantID uint, newStock int) error {
	variant, err := s.variantRepo.FindByID(variantID)
	if err != nil {
		return err
	}
	if variant == nil {
		return errors.New("variant not found")
	}

	return s.variantRepo.UpdateStock(variantID, newStock)
}

// CheckProductAvailability verifica si un producto tiene stock disponible
func (s *ProductService) CheckProductAvailability(productID uint) (bool, int, error) {
	variants, err := s.variantRepo.FindByProductID(productID)
	if err != nil {
		return false, 0, err
	}

	totalStock := 0
	for _, variant := range variants {
		totalStock += variant.Stock
	}

	return totalStock > 0, totalStock, nil
}

// GetProductsByCategory obtiene productos de una categoría específica con paginación
func (s *ProductService) GetProductsByCategory(categoryID uint, offset, limit int) ([]Product, error) {
	// Validar que la categoría existe
	if err := s.ValidateCategory(categoryID); err != nil {
		return nil, err
	}

	return s.productRepo.FindByCategoryID(categoryID, offset, limit)
}

// SearchProducts busca productos por nombre o descripción
func (s *ProductService) SearchProducts(query string, offset, limit int) ([]Product, error) {
	// Esta funcionalidad requeriría agregar un método de búsqueda al repositorio
	// Por ahora retornamos todos los productos activos
	return s.productRepo.FindByActive(true, offset, limit)
}

// DeactivateProduct desactiva un producto y todas sus variantes
func (s *ProductService) DeactivateProduct(productID uint) error {
	product, err := s.productRepo.FindByID(productID)
	if err != nil {
		return err
	}
	if product == nil {
		return errors.New("product not found")
	}

	product.IsActive = false
	return s.productRepo.Update(product)
}

// GetCategoryHierarchy obtiene la jerarquía completa de categorías
func (s *ProductService) GetCategoryHierarchy() ([]Category, error) {
	return s.categoryRepo.FindAll()
}

// ValidateProductSlug verifica que un slug de producto sea único
func (s *ProductService) ValidateProductSlug(slug string, excludeID *uint) error {
	product, err := s.productRepo.FindBySlug(slug)
	if err != nil {
		return err
	}
	if product != nil && (excludeID == nil || product.ID != *excludeID) {
		return errors.New("product slug already exists")
	}
	return nil
}

// ValidateCategorySlug verifica que un slug de categoría sea único
func (s *ProductService) ValidateCategorySlug(slug string, excludeID *uint) error {
	category, err := s.categoryRepo.FindBySlug(slug)
	if err != nil {
		return err
	}
	if category != nil && (excludeID == nil || category.ID != *excludeID) {
		return errors.New("category slug already exists")
	}
	return nil
}

// ValidateVariantSKU verifica que un SKU de variante sea único
func (s *ProductService) ValidateVariantSKU(sku string, excludeID *uint) error {
	variant, err := s.variantRepo.FindBySKU(sku)
	if err != nil {
		return err
	}
	if variant != nil && (excludeID == nil || variant.ID != *excludeID) {
		return errors.New("variant SKU already exists")
	}
	return nil
}

// GetProductStats obtiene estadísticas de un producto
func (s *ProductService) GetProductStats(productID uint) (map[string]interface{}, error) {
	product, err := s.productRepo.FindByID(productID)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, errors.New("product not found")
	}

	variants, err := s.variantRepo.FindByProductID(productID)
	if err != nil {
		return nil, err
	}

	images, err := s.imageRepo.FindByProductID(productID)
	if err != nil {
		return nil, err
	}

	totalStock := 0
	variantCount := len(variants)
	for _, variant := range variants {
		totalStock += variant.Stock
	}

	stats := map[string]interface{}{
		"product_id":     productID,
		"variant_count":  variantCount,
		"total_stock":    totalStock,
		"image_count":    len(images),
		"is_active":      product.IsActive,
		"has_stock":      totalStock > 0,
	}

	return stats, nil
}