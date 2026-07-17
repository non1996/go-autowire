package dal

import (
	"fmt"

	"github.com/non1996/go-autowire/example/infra"
)

type aDaoImpl struct {
	db infra.IMySQL `autowire:"true"`
}

func (a *aDaoImpl) GetA() {
	a.db.Transaction()

	fmt.Println("[ADao] GetA")
}
