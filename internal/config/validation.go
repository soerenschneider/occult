package config

import "github.com/go-playground/validator/v10"

var validation = validator.New()

func Validate(s any) error {
	return validation.Struct(s)
}
