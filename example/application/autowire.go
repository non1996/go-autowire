package application

import (
	_ "github.com/non1996/go-autowire/example/service"

	"github.com/non1996/go-autowire/autowire"
)

func init() {
	autowire.Register(
		autowire.Component[Application]().Register(),
	)
}
