package internal

import (
	"os"
	"sync"

	"github.com/bytedance/gg/gcond"
)

type environmentVariables struct {
	m  map[string]string
	mu sync.Mutex
}

func newEnvironmentVariables() environmentVariables {
	return environmentVariables{m: map[string]string{}}
}

func (e *environmentVariables) get(name string, defaultValue string, require ...bool) (string, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	v, exist := e.m[name]
	if exist {
		return v, true
	}

	ev, exist := os.LookupEnv(name)
	if !exist && required(require) {
		panic(errEnvNotFound(name))
	}

	if exist {
		e.m[name] = ev
	}

	return gcond.If(exist, ev, defaultValue), exist
}
