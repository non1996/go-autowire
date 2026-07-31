package autowire

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
)

type requiredDependency struct{}

type requiredHolder struct {
	dep *requiredDependency `autowire:"true"`
}

type optionalHolder struct {
	dep *requiredDependency `autowire:"true" required:"false"`
}

type conditionalComponent struct{}

func TestTypeOf(t *testing.T) {
	if TypeOf[*requiredDependency]() != reflect.TypeOf((*requiredDependency)(nil)) {
		t.Fatalf("unexpected type: %s", TypeOf[*requiredDependency]())
	}
}

func TestConditionExpressionParsed(t *testing.T) {
	factory := Component[conditionalComponent]().Condition("feature.enabled=true").Register()
	structFactory := factory.factoryHandle().factory

	if structFactory.GetCondition() == nil {
		t.Fatalf("expected condition to be parsed")
	}

	condition := structFactory.GetCondition()
	if condition.Scope != "feature" || condition.Key != "enabled" || condition.Value != "true" {
		t.Fatalf("unexpected condition parsed result: %+v", condition)
	}
}

func TestGetComponentByNameOptionalReturnsZeroValue(t *testing.T) {
	ctx := NewContext()

	got := GetComponentByNameFrom[*requiredDependency](ctx, "missing", false)
	if got != nil {
		t.Fatalf("expected nil component, got %#v", got)
	}
}

func TestAutowireFieldDefaultRequiredTrue(t *testing.T) {
	ctx := NewContext()
	ctx.Register(Component[requiredHolder]().Register())

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

	_ = GetComponentFrom[*requiredHolder](ctx)
}

func TestAutowireFieldRequiredFalseAllowsMissingDependency(t *testing.T) {
	ctx := NewContext()
	ctx.Register(Component[optionalHolder]().Register())

	holder := GetComponentFrom[*optionalHolder](ctx)
	if holder.dep != nil {
		t.Fatalf("expected nil optional dependency, got %#v", holder.dep)
	}
}

type exampleProperties struct {
	Enabled bool
	Port    int
	Names   []string
	Nested  struct {
		TimeoutMS int
	}
}

type exampleConfiguration struct {
	properties exampleProperties
}

type exampleBean interface {
	Value() string
}

type exampleBeanImpl struct {
	value string
}

func (b *exampleBeanImpl) Value() string {
	return b.value
}

type valueAndEnvComponent struct {
	enabled   bool        `value:"example/Enabled"`
	port      int         `value:"example/Port"`
	names     []string    `value:"example/Names"`
	timeoutMS int         `value:"example/Nested.TimeoutMS"`
	region    string      `env:"AUTOWIRE_TEST_REGION" default:"cn"`
	optional  string      `value:"example/NotFound" required:"false"`
	bean      exampleBean `autowire:"true"`
}

func TestConfigurationProvidesBeanPropertiesValueAndEnvironment(t *testing.T) {
	ctx := NewContext()
	ctx.Register(
		Component[exampleConfiguration]().
			Configuration().
			Properties(Property[exampleConfiguration]("example", func(configuration *exampleConfiguration) exampleProperties {
				return configuration.properties
			})).
			Beans(Bean[exampleConfiguration, exampleBean]("exampleBean", func(*exampleConfiguration) exampleBean {
				return &exampleBeanImpl{value: "from bean"}
			})).
			PostConstruct(func(configuration *exampleConfiguration) error {
				configuration.properties = exampleProperties{
					Enabled: true,
					Port:    8080,
					Names:   []string{"alice", "bob"},
				}
				configuration.properties.Nested.TimeoutMS = 1500
				return nil
			}).
			Register(),
		Component[valueAndEnvComponent]().Register(),
	)

	t.Setenv("AUTOWIRE_TEST_REGION", "us")
	component := GetComponentFrom[*valueAndEnvComponent](ctx)
	if !component.enabled || component.port != 8080 || !reflect.DeepEqual(component.names, []string{"alice", "bob"}) || component.timeoutMS != 1500 {
		t.Fatalf("unexpected property injection: %#v", component)
	}
	if component.region != "us" {
		t.Fatalf("unexpected environment injection: %q", component.region)
	}
	if component.optional != "" {
		t.Fatalf("expected empty optional property, got %q", component.optional)
	}
	if component.bean == nil || component.bean.Value() != "from bean" {
		t.Fatalf("unexpected bean injection: %#v", component.bean)
	}
}

type featureProperties struct {
	Enabled bool
}

type enabledByFeature struct{}

func TestConditionalComponentIsFilteredByTypeAndName(t *testing.T) {
	ctx := NewContext()
	ctx.Register(
		Component[exampleConfiguration]().
			Configuration().
			Properties(Property[exampleConfiguration]("feature", func(*exampleConfiguration) featureProperties {
				return featureProperties{Enabled: false}
			})).
			Register(),
		Component[enabledByFeature]().Alias("featureComponent").Condition("feature/Enabled=true").Register(),
	)

	if got := GetComponentFrom[*enabledByFeature](ctx, false); got != nil {
		t.Fatalf("expected inactive component by type, got %#v", got)
	}
	if got := GetComponentByNameFrom[*enabledByFeature](ctx, "featureComponent", false); got != nil {
		t.Fatalf("expected inactive component by name, got %#v", got)
	}
}

type injectedApplication struct {
	bean exampleBean `autowire:"true"`
	port int         `value:"example/Port"`
}

func TestInjectExistingApplication(t *testing.T) {
	ctx := NewContext()
	ctx.Register(
		Component[exampleConfiguration]().
			Configuration().
			Properties(Property[exampleConfiguration]("example", func(*exampleConfiguration) exampleProperties {
				return exampleProperties{Port: 9000}
			})).
			Beans(Bean[exampleConfiguration, exampleBean]("", func(*exampleConfiguration) exampleBean {
				return &exampleBeanImpl{value: "injected"}
			})).
			Register(),
	)

	app := &injectedApplication{}
	ctx.Inject(app)
	if app.port != 9000 || app.bean == nil || app.bean.Value() != "injected" {
		t.Fatalf("unexpected injected application: %#v", app)
	}
}

type concurrentComponent struct{}

var concurrentBuilds int

func TestConcurrentGetBuildsSingletonOnce(t *testing.T) {
	ctx := NewContext()
	concurrentBuilds = 0
	ctx.Register(Component[concurrentComponent]().PostConstruct(func(*concurrentComponent) error {
		concurrentBuilds++
		return nil
	}).Register())

	var waitGroup sync.WaitGroup
	for i := 0; i < 20; i++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			if GetComponentFrom[*concurrentComponent](ctx) == nil {
				t.Error("expected component")
			}
		}()
	}
	waitGroup.Wait()

	if concurrentBuilds != 1 {
		t.Fatalf("expected one singleton build, got %d", concurrentBuilds)
	}
}

type failedConstruction struct{}

func TestFailedConstructionReturnsStableError(t *testing.T) {
	ctx := NewContext()
	ctx.Register(Component[failedConstruction]().PostConstruct(func(*failedConstruction) error {
		return fmt.Errorf("initialization failed")
	}).Register())

	for i := 0; i < 2; i++ {
		func() {
			defer func() {
				recovered := recover()
				if recovered == nil || !strings.Contains(fmt.Sprint(recovered), "failedConstruction") || strings.Contains(fmt.Sprint(recovered), "circular dependency") {
					t.Fatalf("unexpected build failure: %v", recovered)
				}
			}()
			_ = GetComponentFrom[*failedConstruction](ctx)
		}()
	}
}

func TestMissingRequiredEnvironmentPanics(t *testing.T) {
	const name = "AUTOWIRE_TEST_REQUIRED_ENV"
	_ = os.Unsetenv(name)
	type component struct {
		value string `env:"AUTOWIRE_TEST_REQUIRED_ENV"`
	}
	ctx := NewContext()
	ctx.Register(Component[component]().Register())

	defer func() {
		recovered := recover()
		if recovered == nil || !strings.Contains(fmt.Sprint(recovered), name) {
			t.Fatalf("unexpected missing environment error: %v", recovered)
		}
	}()
	_ = GetComponentFrom[*component](ctx)
}

func TestEnvironmentDefaultAndOptionalValues(t *testing.T) {
	const (
		defaultName  = "AUTOWIRE_TEST_DEFAULT_ENV"
		optionalName = "AUTOWIRE_TEST_OPTIONAL_ENV"
	)
	t.Setenv(defaultName, "")
	t.Setenv(optionalName, "")
	_ = os.Unsetenv(defaultName)
	_ = os.Unsetenv(optionalName)

	type component struct {
		defaultValue  int    `env:"AUTOWIRE_TEST_DEFAULT_ENV" default:"42"`
		optionalValue string `env:"AUTOWIRE_TEST_OPTIONAL_ENV" required:"false"`
	}

	ctx := NewContext()
	ctx.Register(Component[component]().Register())

	got := GetComponentFrom[*component](ctx)
	if got.defaultValue != 42 || got.optionalValue != "" {
		t.Fatalf("unexpected environment injection: %#v", got)
	}
}

func TestDuplicatePropertyScopePanics(t *testing.T) {
	ctx := NewContext()
	ctx.Register(
		Component[exampleConfiguration]().
			Alias("firstConfiguration").
			Configuration().
			Properties(Property[exampleConfiguration]("duplicate", func(*exampleConfiguration) exampleProperties {
				return exampleProperties{}
			})).
			Register(),
	)

	defer func() {
		recovered := recover()
		if recovered == nil || !strings.Contains(fmt.Sprint(recovered), "duplicate") {
			t.Fatalf("unexpected duplicate property error: %v", recovered)
		}
	}()

	ctx.Register(
		Component[exampleConfiguration]().
			Alias("secondConfiguration").
			Configuration().
			Properties(Property[exampleConfiguration]("duplicate", func(*exampleConfiguration) exampleProperties {
				return exampleProperties{}
			})).
			Register(),
	)
}

type selectionService interface {
	Name() string
}

type primarySelectionService struct{}

func (*primarySelectionService) Name() string {
	return "primary"
}

type secondarySelectionService struct{}

func (*secondarySelectionService) Name() string {
	return "secondary"
}

func TestAliasPrimaryAndImplement(t *testing.T) {
	ctx := NewContext()
	ctx.Register(
		Component[secondarySelectionService]().
			Alias("secondary").
			Primary(false).
			Implement(TypeOf[selectionService]()).
			Register(),
		Component[primarySelectionService]().
			Alias("primary").
			Implement(TypeOf[selectionService]()).
			Register(),
	)

	if got := GetComponentFrom[selectionService](ctx); got == nil || got.Name() != "primary" {
		t.Fatalf("unexpected primary component: %#v", got)
	}
	if got := GetComponentByNameFrom[selectionService](ctx, "secondary"); got == nil || got.Name() != "secondary" {
		t.Fatalf("unexpected named component: %#v", got)
	}
}

type beanOptionService interface {
	Name() string
}

type beanOptionServiceImpl struct {
	name string
}

func (s *beanOptionServiceImpl) Name() string {
	return s.name
}

func TestBeanOptions(t *testing.T) {
	ctx := NewContext()
	secondary := Bean[exampleConfiguration, beanOptionService](
		"secondaryBean",
		func(*exampleConfiguration) beanOptionService { return &beanOptionServiceImpl{name: "secondary"} },
		PrimaryBean(false),
		ImplementBean(TypeOf[beanOptionService]()),
	)
	if secondary.beanDefinitionHandle().build("configuration").IsPrimary() {
		t.Fatal("expected PrimaryBean(false) to disable primary selection")
	}
	ctx.Register(
		Component[exampleConfiguration]().
			Configuration().
			Properties(Property[exampleConfiguration]("beanFeature", func(*exampleConfiguration) featureProperties {
				return featureProperties{Enabled: true}
			})).
			Beans(
				secondary,
				Bean[exampleConfiguration, beanOptionService](
					"primaryBean",
					func(*exampleConfiguration) beanOptionService { return &beanOptionServiceImpl{name: "primary"} },
					ConditionBean("beanFeature/Enabled=true"),
				),
			).
			Register(),
	)

	if got := GetComponentFrom[beanOptionService](ctx); got == nil || got.Name() != "primary" {
		t.Fatalf("unexpected primary bean: %#v", got)
	}
	if got := GetComponentByNameFrom[beanOptionService](ctx, "secondaryBean"); got == nil || got.Name() != "secondary" {
		t.Fatalf("unexpected named bean: %#v", got)
	}
}

func TestSetPtrUnExportFieldReturnsDiagnosticErrors(t *testing.T) {
	target := &requiredHolder{}
	if err := SetPtrUnExportField(target, "dep", &requiredDependency{}); err != nil {
		t.Fatalf("set unexported field: %v", err)
	}
	if target.dep == nil {
		t.Fatal("expected dependency to be assigned")
	}
	if err := SetPtrUnExportField(target, "missing", nil); err == nil {
		t.Fatal("expected missing field error")
	}
	if err := SetPtrUnExportField(nil, "dep", nil); err == nil {
		t.Fatal("expected nil target error")
	}
}
