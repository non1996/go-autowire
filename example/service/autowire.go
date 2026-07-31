package service

import (
	_ "github.com/non1996/go-autowire/example/dal"

	"github.com/non1996/go-autowire/autowire"
)

func init() {
	autowire.Register(
		autowire.Component[xyzServiceImpl]().Implement(autowire.TypeOf[IXyzService]()).Register(),
	)
}
