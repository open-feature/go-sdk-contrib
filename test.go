package main

import (
	"log"

	"github.com/open-feature/golang-sdk-contrib/hooks/validator/pkg/regex"
	"github.com/open-feature/golang-sdk-contrib/hooks/validator/pkg/validator"
)

func main() {
	hexValidator, err := regex.Hex()
	if err != nil {
		log.Fatal(err)
	}
	v := validator.Hook{Validator: hexValidator}
}
