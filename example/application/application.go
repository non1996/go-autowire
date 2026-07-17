package application

import (
	"github.com/non1996/go-autowire/example/service"
)

type Application struct {
	xyzService service.IXyzService `autowire:"true"`
}

func (a *Application) Run() {
	a.xyzService.Run()
}
