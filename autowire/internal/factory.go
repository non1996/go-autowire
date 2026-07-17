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
	build(ctx *AppContext) any    // 构造组件、依赖注入、后置初始化
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

func (f StructFactory) build(appCtx *AppContext) any {
	component := reflect.New(f.Type)

	// 依赖注入
	for _, fieldInjector := range f.FieldInjectors {
		fieldInjector.inject(appCtx, component.Interface())
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
}

func (f BeanFactory) GetAlias() string {
	return f.Alias
}

func (f BeanFactory) GetType() reflect.Type {
	return f.Type
}

func (f BeanFactory) GetImplement() []reflect.Type {
	return nil
}

func (f BeanFactory) IsPrimary() bool {
	return false
}

func (f BeanFactory) isConfiguration() bool {
	return false
}

func (f BeanFactory) GetCondition() *Condition {
	return nil
}

func (f BeanFactory) onRegister(_ *AppContext) {
}

func (f BeanFactory) build(appCtx *AppContext) any {
	comp := appCtx.GetComponentByName(f.ComponentAlias)
	return f.BuildFunc(comp)
}

// PropertyFactory 参数工厂
type PropertyFactory struct {
	Alias          string
	ComponentAlias string
	Type           reflect.Type
	BuildFunc      func(any) any
}

func (f PropertyFactory) GetAlias() string {
	return fmt.Sprintf("_config/%s", f.Alias)
}

func (f PropertyFactory) GetType() reflect.Type {
	return f.Type
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

func (f PropertyFactory) onRegister(_ *AppContext) {
}

func (f PropertyFactory) build(appCtx *AppContext) any {
	comp := appCtx.GetComponentByName(f.ComponentAlias)
	return f.BuildFunc(comp)
}

// ApplicationFactory 应用工厂
type ApplicationFactory struct {
	App       any
	Injectors []Injector
}

func (a ApplicationFactory) build(appCtx *AppContext) any {
	for _, fieldInjector := range a.Injectors {
		fieldInjector.inject(appCtx, a.App)
	}

	return a.App
}
