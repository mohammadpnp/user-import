package repository

import (
	"context"
	"errors"
	"fmt"

	domain "github.com/mohammadpnp/user-import/internal/domain/user"
	"github.com/mohammadpnp/user-import/internal/infrastructure/db/models"
	"gorm.io/gorm"
)

type UserQueryRepository struct {
	db *gorm.DB
}

func NewUserQueryRepository(db *gorm.DB) *UserQueryRepository {
	return &UserQueryRepository{db: db}
}

func (r *UserQueryRepository) GetByID(ctx context.Context, userID string) (*domain.User, error) {
	var row models.User

	err := r.db.WithContext(ctx).
		Preload("Addresses").
		First(&row, "id = ?", userID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}

	addresses := make([]domain.Address, 0, len(row.Addresses))
	for _, address := range row.Addresses {
		addresses = append(addresses, domain.Address{
			Street:  address.Street,
			City:    address.City,
			State:   address.State,
			ZipCode: address.ZipCode,
			Country: address.Country,
		})
	}

	userAggregate := &domain.User{
		ID:          row.ID,
		Name:        row.Name,
		Email:       row.Email,
		PhoneNumber: row.PhoneNumber,
		Addresses:   addresses,
	}

	return userAggregate, nil
}
