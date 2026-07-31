package internal

import (
	"fmt"
	"reflect"
)

// IComponentFactory 工厂
type IComponentFactory interface {
	GetAlias() string             // 组件名称
	GetType() reflect.Type        // 组件类型
	GetImplement() []reflect.Type // 组件实现类型
	IsPrimary() bool              // 同类型中是否是主要类型
	GetCondition() *Condition     // 组件构造条件
	onRegister(*AppContext)       // 注册完成后的派生产物处理
	build(*AppContext, *buildState) any
}

// ConstructFunc 构造函数
type ConstructFunc func(any any) error

type Condition struct {
	Expression string
	Scope      string
	Key        string
	Value      string
}

// StructFactory 组件工厂
type StructFactory struct {
	Alias          string         // 别名
	Ptr            bool           // 实例是否以指针存储（默认）
	Primary        bool           // 同一类型下，存在多实例时，是否是主实例
	Configuration  bool           // 是否为配置组件
	Type           reflect.Type   // 实例类型，要求只能是结构体类型
	Implement      []reflect.Type // 实例实现的接口类型
	Condition      *Condition     // 初始化实例的条件
	FieldInjectors []Injector     // 依赖注入器
	PostConstruct  ConstructFunc  // 构造函数
	Beans          []BeanFactory  // 仅配置组件生效，产生bean
	Properties     []PropertyFactory
}

func (f StructFactory) GetAlias() string {
	return f.Alias
}

func (f StructFactory) GetType() reflect.Type {
	return f.Type
}

func (f StructFactory) GetImplement() []reflect.Type {
	return f.Implement
}

func (f StructFactory) IsPrimary() bool {
	return f.Primary
}

func (f StructFactory) GetCondition() *Condition {
	return f.Condition
}

func (f StructFactory) onRegister(ctx *AppContext) {
	if len(f.Beans) == 0 && len(f.Properties) == 0 {
		return
	}
	if !f.Configuration {
		panic(fmt.Errorf("component [%s] declares beans or properties but is not a configuration", f.Alias))
	}
	if !ctx.match(f.Condition) {
		return
	}

	for index := range f.Properties {
		ctx.register(&f.Properties[index])
	}
	for index := range f.Beans {
		ctx.register(&f.Beans[index])
	}
}

func (f StructFactory) build(appCtx *AppContext, state *buildState) any {
	component := reflect.New(f.Type)

	// 依赖注入
	for _, fieldInjector := range f.FieldInjectors {
		fieldInjector.inject(appCtx, state, component.Interface())
	}

	// 执行后置操作
	if f.PostConstruct != nil {
		err := f.PostConstruct(component.Interface())
		if err != nil {
			panic(err)
		}
	}

	// 返回实例
	instance := component.Interface()
	return instance
}

// BeanFactory Bean 工厂
type BeanFactory struct {
	Alias          string        // 别名
	ComponentAlias string        // 依赖的配置组件的名称
	Type           reflect.Type  // BuildFunc返回的实例类型，要求只能是结构体类型
	BuildFunc      func(any) any // 构造bean组件
	Primary        bool
	Implement      []reflect.Type
	Condition      *Condition
}

func (f BeanFactory) GetAlias() string {
	return f.Alias
}

func (f BeanFactory) GetType() reflect.Type {
	return f.Type
}

func (f BeanFactory) GetImplement() []reflect.Type {
	return f.Implement
}

func (f BeanFactory) IsPrimary() bool {
	return f.Primary
}

func (f BeanFactory) GetCondition() *Condition {
	return f.Condition
}

func (f BeanFactory) onRegister(_ *AppContext) {
}

func (f BeanFactory) build(appCtx *AppContext, state *buildState) any {
	comp := appCtx.getComponentByName(f.ComponentAlias, true, state)
	return f.BuildFunc(comp)
}

// PropertyFactory 参数工厂
type PropertyFactory struct {
	Scope          string
	ComponentAlias string
	BuildFunc      func(any) any
}

func (f PropertyFactory) GetAlias() string {
	return fmt.Sprintf("_config/%s", f.Scope)
}

func (f PropertyFactory) GetType() reflect.Type {
	return nil
}

func (f PropertyFactory) GetImplement() []reflect.Type {
	return nil
}

func (f PropertyFactory) IsPrimary() bool {
	return false
}

func (f PropertyFactory) GetCondition() *Condition {
	return nil
}

func (f PropertyFactory) onRegister(ctx *AppContext) {
	ctx.properties.add(&propertyProvider{
		scope: f.Scope,
		provide: func() any {
			return f.build(ctx, &buildState{})
		},
	})
}

func (f PropertyFactory) build(appCtx *AppContext, state *buildState) any {
	comp := appCtx.getComponentByName(f.ComponentAlias, true, state)
	return f.BuildFunc(comp)
}

// ApplicationFactory 应用工厂
type ApplicationFactory struct {
	App       any
	Injectors []Injector
}

func (a ApplicationFactory) GetAlias() string {
	return "_application"
}

func (a ApplicationFactory) GetType() reflect.Type {
	return reflect.TypeOf(a.App)
}

func (a ApplicationFactory) GetImplement() []reflect.Type {
	return nil
}

func (a ApplicationFactory) IsPrimary() bool {
	return false
}

func (a ApplicationFactory) GetCondition() *Condition {
	return nil
}

func (a ApplicationFactory) onRegister(_ *AppContext) {
}

func (a ApplicationFactory) build(appCtx *AppContext, state *buildState) any {
	for _, fieldInjector := range a.Injectors {
		fieldInjector.inject(appCtx, state, a.App)
	}

	return a.App
}
