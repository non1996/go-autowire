package internal

import (
	"reflect"
	"strings"
	"testing"
)

type featureToggle struct {
	Enabled string
}

type namedComponent struct {
	Name string
}

type loopComponent struct{}

func TestPropertiesProviderCachesInstance(t *testing.T) {
	props := newProperties()
	count := 0
	props.add(propertyProvider{
		scope: "feature",
		provide: func() any {
			count++
			return featureToggle{Enabled: "true"}
		},
	})

	v1, exist := props.get("feature", "Enabled")
	if !exist || v1 != "true" {
		t.Fatalf("unexpected first property lookup result: exist=%v value=%v", exist, v1)
	}

	v2, exist := props.get("feature", "Enabled")
	if !exist || v2 != "true" {
		t.Fatalf("unexpected second property lookup result: exist=%v value=%v", exist, v2)
	}

	if count != 1 {
		t.Fatalf("expected provider to be called once, got %d", count)
	}
}

func TestGetComponentRespectsCondition(t *testing.T) {
	ctx := NewAppContext()
	ctx.properties.add(propertyProvider{
		scope: "feature",
		provide: func() any {
			return featureToggle{Enabled: "true"}
		},
	})

	ctx.Register(&StructFactory{
		Alias:   "enabled",
		Type:    reflect.TypeOf(namedComponent{}),
		Primary: false,
		Condition: &Condition{
			Scope: "feature",
			Key:   "Enabled",
			Value: "true",
		},
		PostConstruct: func(v any) error {
			v.(*namedComponent).Name = "enabled"
			return nil
		},
	})

	ctx.Register(&StructFactory{
		Alias:   "disabled",
		Type:    reflect.TypeOf(namedComponent{}),
		Primary: false,
		Condition: &Condition{
			Scope: "feature",
			Key:   "Enabled",
			Value: "false",
		},
		PostConstruct: func(v any) error {
			v.(*namedComponent).Name = "disabled"
			return nil
		},
	})

	component := ctx.GetComponent(reflect.TypeOf((*namedComponent)(nil))).(*namedComponent)
	if component.Name != "enabled" {
		t.Fatalf("expected enabled component, got %#v", component)
	}
}

func TestGetComponentDetectsCircularDependency(t *testing.T) {
	ctx := NewAppContext()
	ctx.Register(&StructFactory{
		Alias:   "loop",
		Type:    reflect.TypeOf(loopComponent{}),
		Primary: true,
		FieldInjectors: []Injector{
			ComponentInjector{
				Required: true,
				DepType:  reflect.TypeOf(loopComponent{}),
				InjectFn: func(_, _ any) {},
			},
		},
	})

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatalf("expected circular dependency panic")
		}

		msg := recovered.(error).Error()
		if !strings.Contains(msg, "circular dependency") {
			t.Fatalf("unexpected panic: %v", recovered)
		}
	}()

	_ = ctx.GetComponent(reflect.TypeOf((*loopComponent)(nil)))
}

func TestEnvironmentVariableErrorContainsVariableName(t *testing.T) {
	envs := newEnvironmentVariables()

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatalf("expected panic for missing env")
		}

		msg := recovered.(error).Error()
		if !strings.Contains(msg, "AUTOWIRE_TEST_MISSING_ENV") {
			t.Fatalf("unexpected panic: %v", recovered)
		}
	}()

	_ = envs.get("AUTOWIRE_TEST_MISSING_ENV", "", true)
}
