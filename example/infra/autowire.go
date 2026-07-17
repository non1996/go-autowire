package infra

import "github.com/non1996/go-autowire/autowire"

func init() {
	autowire.Register(
		autowire.Component[mysqlImpl]().Implement(autowire.TypeOf[IMySQL]()).Register(),
	)
}
