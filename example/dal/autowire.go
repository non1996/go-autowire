package dal

import (
	_ "github.com/non1996/go-autowire/example/infra"

	"github.com/non1996/go-autowire/autowire"
)

func init() {
	autowire.Register(
		autowire.Component[aDaoImpl]().Implement(autowire.TypeOf[IADao]()).Register(),
		autowire.Component[bDaoImpl]().Implement(autowire.TypeOf[IBDao]()).Register(),
	)
}
