package internal

import (
	"fmt"
)

var (
	errMultiMatch = fmt.Errorf("multiple ComponentContainer meet filter GetCondition and no instance was designated as primary")
)

func errMultiPrimaryMatch(typeName string) error {
	return fmt.Errorf("multiple primary instances [%s] found", typeName)
}

func errComponentNotFound(typeName string) error {
	return fmt.Errorf("instance [%s] not found", typeName)
}

func errComponentDuplicate(name string) error {
	return fmt.Errorf("instance [%s] is duplicate", name)
}

func errValueNotFound(scope string, key string) error {
	return fmt.Errorf("property [%s/%s] not found", scope, key)
}

func errEnvNotFound(name string) error {
	return fmt.Errorf("env [%s] not found", name)
}

func errCircularDependency(name string) error {
	return fmt.Errorf("circular dependency detected while building [%s]", name)
}
