package main

import (
	"github.com/non1996/go-autowire/autowire-cli/autowire"
)

func main() {
	autowire.GenerateAll(
		autowire.Config{
			Module:           "github.com/non1996/example",
			Root:             "/Users/bytedance/goproj/any_test",
			AutowireFileName: "autowire.go",
			GenFileName:      "autowire_gen.go",
		},
		"/Users/bytedance/goproj/any_test/autowire-example")
}
