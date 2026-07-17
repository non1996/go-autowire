package main

import (
	"github.com/non1996/go-autowire/autowire"
	"github.com/non1996/go-autowire/example/application"
)

func main() {
	app := autowire.GetComponent[*application.Application]()
	app.Run()
}
