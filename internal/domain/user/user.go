package user

import (
	"net/mail"
	"strings"
)

type Address struct {
	Street  string
	City    string
	State   string
	ZipCode string
	Country string
}

func (a Address) validate() error {
	if strings.TrimSpace(a.Street) == "" ||
		strings.TrimSpace(a.City) == "" ||
		strings.TrimSpace(a.State) == "" ||
		strings.TrimSpace(a.ZipCode) == "" ||
		strings.TrimSpace(a.Country) == "" {
		return ErrInvalidAddress
	}
	return nil
}

type User struct {
	ID          string
	Name        string
	Email       string
	PhoneNumber string
	Addresses   []Address
}

func NewUser(id, name, email, phoneNumber string, addresses []Address) (User, error) {
	if _, err := mail.ParseAddress(email); err != nil {
		return User{}, ErrInvalidEmail
	}

	for _, address := range addresses {
		if err := address.validate(); err != nil {
			return User{}, err
		}
	}

	return User{
		ID:          id,
		Name:        name,
		Email:       email,
		PhoneNumber: phoneNumber,
		Addresses:   addresses,
	}, nil
}
