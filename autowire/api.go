package autowire

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unsafe"

	"github.com/non1996/go-autowire/autowire/internal"
	"github.com/non1996/go-autowire/autowire/internal/util"
)

func TypeOf[T any]() reflect.Type {
	return util.TypeOf[T]()
}

// Factory 是由 Component 创建的组件注册定义。
type Factory interface {
	factoryHandle() factoryHandle
}

type factoryHandle struct {
	factory internal.IComponentFactory
}

func (h factoryHandle) factoryHandle() factoryHandle {
	return h
}

// Container 是一个隔离的依赖注入容器。
type Container struct {
	app *internal.AppContext
}

// NewContext 创建一个隔离容器，适用于测试或应用需要多个独立配置容器的场景。
func NewContext() *Container {
	return &Container{app: internal.NewAppContext()}
}

// Register 将组件工厂注册到默认容器。
func Register(factories ...Factory) {
	defaultContext.Register(factories...)
}

// Register 将组件工厂注册到当前容器。
func (c *Container) Register(factories ...Factory) {
	for _, factory := range factories {
		if factory == nil {
			panic(fmt.Errorf("[autowire] cannot register a nil factory"))
		}
		c.app.Register(factory.factoryHandle().factory)
	}
}

// Context 返回默认容器。
func Context() *Container {
	return defaultContext
}

// GetComponent 从默认容器中返回类型为 T 的唯一生效组件；找不到必填组件时会 panic。
func GetComponent[T any](require ...bool) T {
	return GetComponentFrom[T](defaultContext, require...)
}

// GetComponentFrom 从 c 中返回类型为 T 的唯一生效组件。
func GetComponentFrom[T any](c *Container, require ...bool) T {
	component := contextOrDefault(c).app.GetComponent(util.TypeOf[T](), require...)
	return util.Cast[T](component)
}

// GetComponentByName 从默认容器中返回名称为 name 的生效组件。
func GetComponentByName[T any](name string, require ...bool) T {
	return GetComponentByNameFrom[T](defaultContext, name, require...)
}

// GetComponentByNameFrom 从 c 中返回名称为 name 的生效组件。
func GetComponentByNameFrom[T any](c *Container, name string, require ...bool) T {
	component := contextOrDefault(c).app.GetComponentByName(name, require...)
	return util.Cast[T](component)
}

// Inject 为已有的结构体指针执行字段注入。
func (c *Container) Inject(app any) {
	if app == nil {
		panic(fmt.Errorf("[autowire] cannot inject into nil"))
	}

	value := reflect.ValueOf(app)
	if value.Kind() != reflect.Ptr || value.IsNil() || value.Elem().Kind() != reflect.Struct {
		panic(fmt.Errorf("[autowire] inject target must be a non-nil pointer to a struct, got %T", app))
	}

	contextOrDefault(c).app.Inject(internal.ApplicationFactory{
		App:       app,
		Injectors: fieldInjectors(value.Elem().Type()),
	})
}

func contextOrDefault(c *Container) *Container {
	if c == nil {
		return defaultContext
	}
	return c
}

type ComponentFactoryBuilder[C any] struct {
	alias         string                 // 组件别名
	primary       bool                   // 是否“主”组件
	configuration bool                   // 是否配置类组件
	typ           reflect.Type           // 组件的反射类型
	implement     []reflect.Type         // 实现接口的反射类型
	condition     *internal.Condition    // 组件初始化条件
	postConstruct internal.ConstructFunc // 构造器
	beans         []BeanDefinition
	properties    []PropertyDefinition
}

func Component[C any]() *ComponentFactoryBuilder[C] {
	return &ComponentFactoryBuilder[C]{
		typ:     internal.TypeOf[C](),
		primary: true,
	}
}

func (f *ComponentFactoryBuilder[C]) Alias(alias string) *ComponentFactoryBuilder[C] {
	f.alias = alias
	return f
}

func (f *ComponentFactoryBuilder[C]) Primary(primary bool) *ComponentFactoryBuilder[C] {
	f.primary = primary
	return f
}

func (f *ComponentFactoryBuilder[C]) Implement(types ...reflect.Type) *ComponentFactoryBuilder[C] {
	f.implement = types
	return f
}

func (f *ComponentFactoryBuilder[C]) Condition(expression string) *ComponentFactoryBuilder[C] {
	f.condition = parseCondition(expression)
	return f
}

func (f *ComponentFactoryBuilder[C]) Configuration() *ComponentFactoryBuilder[C] {
	f.configuration = true
	return f
}

// Beans 为当前配置组件关联其产生的组件。
func (f *ComponentFactoryBuilder[C]) Beans(definitions ...BeanDefinition) *ComponentFactoryBuilder[C] {
	f.beans = append(f.beans, definitions...)
	return f
}

// Properties 为当前配置组件关联其产生的属性对象。
func (f *ComponentFactoryBuilder[C]) Properties(definitions ...PropertyDefinition) *ComponentFactoryBuilder[C] {
	f.properties = append(f.properties, definitions...)
	return f
}

func (f *ComponentFactoryBuilder[C]) PostConstruct(constructFunc func(*C) error) *ComponentFactoryBuilder[C] {
	if constructFunc == nil {
		panic(fmt.Errorf("[autowire] post construct function is nil"))
	}
	f.postConstruct = func(any any) error {
		return constructFunc(any.(*C))
	}

	return f
}

func (f *ComponentFactoryBuilder[C]) Register() Factory {
	if f.typ.Kind() != reflect.Struct {
		panic(fmt.Errorf("[autowire] only support struct type components, got %s", f.typ))
	}

	// 未指定别名时，使用结构体名称作为别名。
	f.alias = strings.TrimSpace(f.alias)
	if f.alias == "" {
		f.alias = f.typ.Name()
	}
	if f.alias == "" {
		panic(fmt.Errorf("[autowire] component alias is empty"))
	}

	for _, implementedType := range f.implement {
		validateImplement(f.typ, implementedType)
	}

	beans := make([]internal.BeanFactory, 0, len(f.beans))
	for _, definition := range f.beans {
		if definition == nil {
			panic(fmt.Errorf("[autowire] component [%s] contains a nil bean definition", f.alias))
		}
		beans = append(beans, definition.beanDefinitionHandle().build(f.alias))
	}

	properties := make([]internal.PropertyFactory, 0, len(f.properties))
	for _, definition := range f.properties {
		if definition == nil {
			panic(fmt.Errorf("[autowire] component [%s] contains a nil property definition", f.alias))
		}
		properties = append(properties, definition.propertyDefinitionHandle().build(f.alias))
	}

	factory := &internal.StructFactory{
		Alias:          f.alias,
		Primary:        f.primary,
		Configuration:  f.configuration,
		Type:           f.typ,
		Implement:      f.implement,
		Condition:      f.condition,
		FieldInjectors: fieldInjectors(f.typ),
		PostConstruct:  f.postConstruct,
		Beans:          beans,
		Properties:     properties,
	}

	return factoryHandle{factory: factory}
}

// BeanDefinition 描述由配置组件创建的组件。
type BeanDefinition interface {
	beanDefinitionHandle() beanDefinitionHandle
}

type beanDefinitionHandle struct {
	build func(componentAlias string) internal.BeanFactory
}

type beanDefinition[C, B any] struct {
	alias     string
	build     func(*C) B
	primary   bool
	implement []reflect.Type
	condition *internal.Condition
}

// Bean 定义由配置组件构建的组件。B 可以是指针或接口类型；未指定别名时使用 B 的类型名。
func Bean[C, B any](alias string, build func(*C) B, options ...BeanOption) BeanDefinition {
	if build == nil {
		panic(fmt.Errorf("[autowire] bean build function is nil"))
	}

	definition := &beanDefinition[C, B]{
		alias:   strings.TrimSpace(alias),
		build:   build,
		primary: true,
	}
	beanOptions := &beanOptions{}
	for _, option := range options {
		if option == nil {
			panic(fmt.Errorf("[autowire] bean option is nil"))
		}
		option(beanOptions)
	}
	if beanOptions.primary != nil {
		definition.primary = *beanOptions.primary
	}
	definition.implement = beanOptions.implement
	definition.condition = beanOptions.condition

	return definition
}

func (d *beanDefinition[C, B]) beanDefinitionHandle() beanDefinitionHandle {
	return beanDefinitionHandle{build: d.beanFactory}
}

func (d *beanDefinition[C, B]) beanFactory(componentAlias string) internal.BeanFactory {
	typ := TypeOf[B]()
	if typ == nil {
		panic(fmt.Errorf("[autowire] bean [%s] has an invalid nil type", d.alias))
	}
	alias := d.alias
	if alias == "" {
		alias = typeAlias(typ)
	}
	for _, implementedType := range d.implement {
		validateBeanImplement(typ, implementedType, alias)
	}

	return internal.BeanFactory{
		Alias:          alias,
		ComponentAlias: componentAlias,
		Type:           typ,
		BuildFunc: func(component any) any {
			configuration, ok := component.(*C)
			if !ok {
				panic(fmt.Errorf("[autowire] bean [%s] configuration type mismatch: got %T", alias, component))
			}
			return d.build(configuration)
		},
		Primary:   d.primary,
		Implement: d.implement,
		Condition: d.condition,
	}
}

// BeanOption 用于自定义 Bean 定义。
type BeanOption func(*beanOptions)

type beanOptions struct {
	primary   *bool
	implement []reflect.Type
	condition *internal.Condition
}

// PrimaryBean 控制 Bean 是否在同类型候选中优先被解析。
func PrimaryBean(primary bool) BeanOption {
	return func(options *beanOptions) {
		options.primary = &primary
	}
}

// ImplementBean 声明 Bean 实现的接口。
func ImplementBean(types ...reflect.Type) BeanOption {
	return func(options *beanOptions) {
		options.implement = append(options.implement, types...)
	}
}

// ConditionBean 仅在表达式匹配属性值时激活 Bean。
func ConditionBean(expression string) BeanOption {
	return func(options *beanOptions) {
		options.condition = parseCondition(expression)
	}
}

// PropertyDefinition 描述由配置组件构建并在指定 scope 下暴露的属性对象。
type PropertyDefinition interface {
	propertyDefinitionHandle() propertyDefinitionHandle
}

type propertyDefinitionHandle struct {
	build func(componentAlias string) internal.PropertyFactory
}

type propertyDefinition[C, P any] struct {
	scope string
	build func(*C) P
}

// Property 在指定 scope 下暴露配置对象。其导出结构体字段会使用点分隔的嵌套字段名展开。
func Property[C, P any](scope string, build func(*C) P) PropertyDefinition {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		panic(fmt.Errorf("[autowire] property scope is empty"))
	}
	if build == nil {
		panic(fmt.Errorf("[autowire] property [%s] build function is nil", scope))
	}
	return propertyDefinition[C, P]{scope: scope, build: build}
}

func (d propertyDefinition[C, P]) propertyDefinitionHandle() propertyDefinitionHandle {
	return propertyDefinitionHandle{build: d.propertyFactory}
}

func (d propertyDefinition[C, P]) propertyFactory(componentAlias string) internal.PropertyFactory {
	typ := TypeOf[P]()
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		panic(fmt.Errorf("[autowire] property [%s] must return a struct or pointer to a struct, got %s", d.scope, TypeOf[P]()))
	}

	return internal.PropertyFactory{
		Scope:          d.scope,
		ComponentAlias: componentAlias,
		BuildFunc: func(component any) any {
			configuration, ok := component.(*C)
			if !ok {
				panic(fmt.Errorf("[autowire] property [%s] configuration type mismatch: got %T", d.scope, component))
			}
			return d.build(configuration)
		},
	}
}

func fieldInjectors(typ reflect.Type) []internal.Injector {
	injectors := make([]internal.Injector, 0)
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		autowire, hasAutowire := field.Tag.Lookup("autowire")
		valuePath, hasValue := field.Tag.Lookup("value")
		envName, hasEnv := field.Tag.Lookup("env")
		if !hasAutowire && !hasValue && !hasEnv {
			continue
		}
		if countTrue(hasAutowire, hasValue, hasEnv) != 1 {
			panic(fmt.Errorf("[autowire] field [%s.%s] must declare exactly one of autowire, value, env", typ, field.Name))
		}
		required := parseRequiredTag(field)
		fieldName := field.Name

		switch {
		case hasAutowire:
			if autowire != "true" {
				panic(fmt.Errorf("[autowire] field [%s.%s] autowire must be true", typ, fieldName))
			}
			if field.Type.Kind() != reflect.Interface && field.Type.Kind() != reflect.Ptr {
				panic(fmt.Errorf("[autowire] field [%s.%s] autowire target must be an interface or pointer, got %s", typ, fieldName, field.Type))
			}
			injectors = append(injectors, internal.ComponentInjector{
				Qualifier: strings.TrimSpace(field.Tag.Get("qualifier")),
				Required:  required,
				DepType:   field.Type,
				InjectFn: func(component, dependency any) {
					mustSetField(component, fieldName, dependency, "component")
				},
			})
		case hasValue:
			scope, key := parsePropertyPath(valuePath, fmt.Sprintf("%s.%s", typ, fieldName))
			injectors = append(injectors, internal.ValueInjector{
				Scope: scope, Key: key, Required: required,
				InjectFn: func(component any, value any) {
					mustSetField(component, fieldName, value, fmt.Sprintf("property [%s/%s]", scope, key))
				},
			})
		case hasEnv:
			envName = strings.TrimSpace(envName)
			if envName == "" {
				panic(fmt.Errorf("[autowire] field [%s.%s] env name is empty", typ, fieldName))
			}
			defaultValue, hasDefault := field.Tag.Lookup("default")
			injectors = append(injectors, internal.EnvInjector{
				Key: envName, Required: required, DefaultValue: defaultValue, HasDefault: hasDefault,
				InjectFn: func(component any, value string) {
					mustSetField(component, fieldName, value, fmt.Sprintf("env [%s]", envName))
				},
			})
		}
	}
	return injectors
}

func parseCondition(expression string) *internal.Condition {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		panic(fmt.Errorf("[autowire] condition expression is empty"))
	}

	idx := strings.Index(expression, "=")
	if idx <= 0 || idx == len(expression)-1 {
		panic(fmt.Errorf("[autowire] invalid condition expression: %s", expression))
	}

	path := strings.TrimSpace(expression[:idx])
	value := strings.TrimSpace(expression[idx+1:])
	if path == "" || value == "" {
		panic(fmt.Errorf("[autowire] invalid condition expression: %s", expression))
	}

	var scope, key string
	if slash := strings.Index(path, "/"); slash > 0 && slash < len(path)-1 {
		scope = strings.TrimSpace(path[:slash])
		key = strings.TrimSpace(path[slash+1:])
	} else if dot := strings.Index(path, "."); dot > 0 && dot < len(path)-1 {
		scope = strings.TrimSpace(path[:dot])
		key = strings.TrimSpace(path[dot+1:])
	}

	if scope == "" || key == "" {
		panic(fmt.Errorf("[autowire] invalid condition expression: %s", expression))
	}

	return &internal.Condition{
		Expression: expression,
		Scope:      scope,
		Key:        key,
		Value:      value,
	}
}

func parseRequiredTag(field reflect.StructField) bool {
	raw, exists := field.Tag.Lookup("required")
	if !exists {
		return true
	}
	required, err := strconv.ParseBool(raw)
	if err != nil {
		panic(fmt.Errorf("[autowire] field [%s.%s] has invalid required value %q", field.Type, field.Name, raw))
	}
	return required
}

func parsePropertyPath(path string, field string) (scope, key string) {
	path = strings.TrimSpace(path)
	if slash := strings.Index(path, "/"); slash > 0 && slash < len(path)-1 {
		scope = strings.TrimSpace(path[:slash])
		key = strings.TrimSpace(path[slash+1:])
	} else if dot := strings.Index(path, "."); dot > 0 && dot < len(path)-1 {
		scope = strings.TrimSpace(path[:dot])
		key = strings.TrimSpace(path[dot+1:])
	}
	if scope == "" || key == "" {
		panic(fmt.Errorf("[autowire] field [%s] has invalid property path %q; expected scope/key", field, path))
	}
	return scope, key
}

func countTrue(values ...bool) int {
	count := 0
	for _, value := range values {
		if value {
			count++
		}
	}
	return count
}

func validateImplement(typ, implementedType reflect.Type) {
	if implementedType == nil || implementedType.Kind() != reflect.Interface {
		panic(fmt.Errorf("[autowire] component [%s] implement target must be an interface, got %v", typ, implementedType))
	}
	if !reflect.PointerTo(typ).Implements(implementedType) {
		panic(fmt.Errorf("[autowire] component [%s] does not implement [%s]", typ, implementedType))
	}
}

func validateBeanImplement(typ, implementedType reflect.Type, alias string) {
	if implementedType == nil || implementedType.Kind() != reflect.Interface {
		panic(fmt.Errorf("[autowire] bean [%s] implement target must be an interface, got %v", alias, implementedType))
	}
	if !typ.Implements(implementedType) {
		panic(fmt.Errorf("[autowire] bean [%s] type [%s] does not implement [%s]", alias, typ, implementedType))
	}
}

func typeAlias(typ reflect.Type) string {
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	return typ.Name()
}

func mustSetField(target any, fieldName string, value any, source string) {
	if err := SetPtrUnExportField(target, fieldName, value); err != nil {
		panic(fmt.Errorf("[autowire] inject %s into field [%s]: %w", source, fieldName, err))
	}
}

// SetPtrUnExportField 为非空指针所指结构体的导出或非导出字段赋值；
// 赋值目标或类型不合法时返回可诊断错误。
func SetPtrUnExportField(s any, fieldName string, val any) error {
	v, err := ptrField(s, fieldName)
	if err != nil {
		return err
	}
	if val == nil {
		v.Set(reflect.Zero(v.Type()))
		return nil
	}

	if err := assignValue(v, reflect.ValueOf(val)); err != nil {
		return fmt.Errorf("cannot assign value type [%T] to [%s]: %w", val, v.Type(), err)
	}
	return nil
}

func ptrField(s any, fieldName string) (reflect.Value, error) {
	value := reflect.ValueOf(s)
	if value.Kind() != reflect.Ptr || value.IsNil() {
		return reflect.Value{}, fmt.Errorf("target must be a non-nil pointer to a struct, got %T", s)
	}
	value = value.Elem()
	if value.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("target must point to a struct, got %T", s)
	}
	field := value.FieldByName(fieldName)
	if !field.IsValid() {
		return reflect.Value{}, fmt.Errorf("field [%s] does not exist", fieldName)
	}
	if !field.CanSet() {
		field = reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
	}
	return field, nil
}

func assignValue(destination, source reflect.Value) error {
	if source.Type().AssignableTo(destination.Type()) {
		destination.Set(source)
		return nil
	}
	if source.Kind() == reflect.String {
		return assignString(destination, source.String())
	}
	if source.Type().ConvertibleTo(destination.Type()) && numericKind(source.Kind()) && numericKind(destination.Kind()) {
		destination.Set(source.Convert(destination.Type()))
		return nil
	}
	if destination.Kind() == reflect.Slice && source.Kind() == reflect.Slice {
		result := reflect.MakeSlice(destination.Type(), source.Len(), source.Len())
		for i := 0; i < source.Len(); i++ {
			if err := assignValue(result.Index(i), source.Index(i)); err != nil {
				return fmt.Errorf("slice item %d: %w", i, err)
			}
		}
		destination.Set(result)
		return nil
	}
	return fmt.Errorf("incompatible types")
}

func assignString(destination reflect.Value, value string) error {
	switch destination.Kind() {
	case reflect.String:
		destination.SetString(value)
		return nil
	case reflect.Bool:
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		destination.SetBool(parsed)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parsed, err := strconv.ParseInt(value, 10, destination.Type().Bits())
		if err != nil {
			return err
		}
		destination.SetInt(parsed)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		parsed, err := strconv.ParseUint(value, 10, destination.Type().Bits())
		if err != nil {
			return err
		}
		destination.SetUint(parsed)
		return nil
	case reflect.Float32, reflect.Float64:
		parsed, err := strconv.ParseFloat(value, destination.Type().Bits())
		if err != nil {
			return err
		}
		destination.SetFloat(parsed)
		return nil
	case reflect.Slice:
		values := strings.Split(value, ",")
		result := reflect.MakeSlice(destination.Type(), len(values), len(values))
		for i, item := range values {
			if err := assignString(result.Index(i), strings.TrimSpace(item)); err != nil {
				return fmt.Errorf("slice item %d: %w", i, err)
			}
		}
		destination.Set(result)
		return nil
	default:
		return fmt.Errorf("cannot convert string")
	}
}

func numericKind(kind reflect.Kind) bool {
	return kind >= reflect.Int && kind <= reflect.Float64
}
