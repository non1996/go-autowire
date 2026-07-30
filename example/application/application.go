package application

import (
	"fmt"

	"github.com/non1996/go-autowire/example/config"
	"github.com/non1996/go-autowire/example/service"
)

type Application struct {
	xyzService service.IXyzService    `autowire:"true"`
	provider   config.MessageProvider `autowire:"true" qualifier:"messageProvider"`
	greeter    config.Greeter         `autowire:"true" required:"false"`
	message    string                 `value:"example/Message"`
	region     string                 `env:"AUTOWIRE_EXAMPLE_REGION" default:"cn"`
}

func (a *Application) Run() {
	fmt.Printf("[Application] message=%s provider=%s region=%s\n", a.message, a.provider.Message(), a.region)
	if a.greeter != nil {
		fmt.Println(a.greeter.Greet())
	}
	a.xyzService.Run()
}
