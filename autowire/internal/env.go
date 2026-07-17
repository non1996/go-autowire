package internal

import (
	"os"

	"github.com/non1996/go-jsonobj/function"
)

type environmentVariables struct {
	m map[string]string
}

func newEnvironmentVariables() environmentVariables {
	return environmentVariables{m: map[string]string{}}
}

func (e *environmentVariables) get(name string, defaultValue string, require ...bool) string {
	v, exist := e.m[name]
	if exist {
		return v
	}

	ev, exist := os.LookupEnv(name)
	if !exist && required(require) {
		panic(errEnvNotFound(name))
	}

	if exist {
		e.m[name] = ev
	}

	return function.Ternary(exist, ev, defaultValue)
}
