package user

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	domain "github.com/mohammadpnp/user-import/internal/domain/user"
)

var userIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

type GetUserByIDInput struct {
	ID string
}

type GetUserAddressOutput struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	State   string `json:"state"`
	ZipCode string `json:"zip_code"`
	Country string `json:"country"`
}

type GetUserByIDOutput struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Email       string                 `json:"email"`
	PhoneNumber string                 `json:"phone_number"`
	Addresses   []GetUserAddressOutput `json:"addresses"`
}

type GetUserByID interface {
	Execute(ctx context.Context, in GetUserByIDInput) (GetUserByIDOutput, error)
}

type getUserByID struct {
	repo domain.UserQueryRepository
}

func NewGetUserByID(repo domain.UserQueryRepository) GetUserByID {
	return &getUserByID{repo: repo}
}

func (uc *getUserByID) Execute(ctx context.Context, in GetUserByIDInput) (GetUserByIDOutput, error) {
	if !userIDPattern.MatchString(in.ID) {
		return GetUserByIDOutput{}, ErrInvalidUserID
	}

	userAggregate, err := uc.repo.GetByID(ctx, in.ID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return GetUserByIDOutput{}, ErrUserNotFound
		}
		return GetUserByIDOutput{}, fmt.Errorf("%w: %v", ErrGetUserByID, err)
	}

	addresses := make([]GetUserAddressOutput, 0, len(userAggregate.Addresses))
	for _, address := range userAggregate.Addresses {
		addresses = append(addresses, GetUserAddressOutput{
			Street:  address.Street,
			City:    address.City,
			State:   address.State,
			ZipCode: address.ZipCode,
			Country: address.Country,
		})
	}

	return GetUserByIDOutput{
		ID:          userAggregate.ID,
		Name:        userAggregate.Name,
		Email:       userAggregate.Email,
		PhoneNumber: userAggregate.PhoneNumber,
		Addresses:   addresses,
	}, nil
}
