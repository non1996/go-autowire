package dal

import (
	"fmt"

	"github.com/non1996/go-autowire/example/infra"
)

type bDaoImpl struct {
	db infra.IMySQL `autowire:"true"`
}

func (b *bDaoImpl) GetB() {
	b.db.Transaction()

	fmt.Println("[BDao] GetB")
}
