package service

import (
	"fmt"

	"github.com/non1996/go-autowire/example/dal"
)

type xyzServiceImpl struct {
	aDao dal.IADao `autowire:"true"`
	bDao dal.IBDao `autowire:"true"`
}

func (xyz *xyzServiceImpl) Run() {
	xyz.aDao.GetA()
	xyz.bDao.GetB()

	fmt.Println("[xyz] service is running")
}
