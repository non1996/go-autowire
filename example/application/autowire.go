package application

import (
	"github.com/non1996/go-autowire/example/config"
	_ "github.com/non1996/go-autowire/example/service"

	"github.com/non1996/go-autowire/autowire"
)

type featureGreeter struct{}

func (featureGreeter) Greet() string {
	return "[Application] conditional component is active"
}

func init() {
	autowire.Register(
		autowire.Component[featureGreeter]().
			Condition("example/Feature.Enabled=true").
			Implement(autowire.TypeOf[config.Greeter]()).
			Register(),
		autowire.Component[Application]().Register(),
	)
}
