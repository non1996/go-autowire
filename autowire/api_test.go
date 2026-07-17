package autowire

import (
	"strings"
	"testing"

	"github.com/non1996/go-autowire/autowire/internal"
)

type requiredDependency struct{}

type requiredHolder struct {
	dep *requiredDependency `autowire:"true"`
}

type optionalHolder struct {
	dep *requiredDependency `autowire:"true" required:"false"`
}

type conditionalComponent struct{}

func TestConditionExpressionParsed(t *testing.T) {
	factory := Component[conditionalComponent]().Condition("feature.enabled=true").Register()
	structFactory := factory.(*internal.StructFactory)

	if structFactory.Condition == nil {
		t.Fatalf("expected condition to be parsed")
	}

	if structFactory.Condition.Scope != "feature" || structFactory.Condition.Key != "enabled" || structFactory.Condition.Value != "true" {
		t.Fatalf("unexpected condition parsed result: %+v", structFactory.Condition)
	}
}

func TestGetComponentByNameOptionalReturnsZeroValue(t *testing.T) {
	defaultAppContext = internal.NewAppContext()

	got := GetComponentByName[*requiredDependency]("missing", false)
	if got != nil {
		t.Fatalf("expected nil component, got %#v", got)
	}
}

func TestAutowireFieldDefaultRequiredTrue(t *testing.T) {
	defaultAppContext = internal.NewAppContext()
	Register(Component[requiredHolder]().Register())

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatalf("expected panic when required dependency is missing")
		}

		msg := recovered.(error).Error()
		if !strings.Contains(msg, "requiredDependency") {
			t.Fatalf("unexpected panic: %v", recovered)
		}
	}()

	_ = GetComponent[*requiredHolder]()
}

func TestAutowireFieldRequiredFalseAllowsMissingDependency(t *testing.T) {
	defaultAppContext = internal.NewAppContext()
	Register(Component[optionalHolder]().Register())

	holder := GetComponent[*optionalHolder]()
	if holder.dep != nil {
		t.Fatalf("expected nil optional dependency, got %#v", holder.dep)
	}
}
