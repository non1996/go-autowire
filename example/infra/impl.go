package infra

import (
	"fmt"
)

type mysqlImpl struct {
}

func (impl *mysqlImpl) Transaction() {
	fmt.Println("Transaction")
}
