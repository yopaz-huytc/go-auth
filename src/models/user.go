package models

type User struct {
	ID      string `json:"id" validate:"required"`
	Name    string `json:"name" validate:"required"`
	Surname string `json:"surname" validate:"required"`
	Company string `json:"company" validate:"required"`
	Age     string `json:"age" validate:"required"`
}
