package autowire

import (
	"github.com/non1996/go-autowire/autowire/internal"
)

var defaultContext = &Container{app: internal.NewAppContext()}
