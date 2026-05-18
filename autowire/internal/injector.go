package internal

import (
	"reflect"
)

// Injector 注入器
// 注意：不管组件是以值类型注册还是以指针类型注册，泛型参数 C 都是组件的值类型
type Injector interface {
	inject(*AppContext, any)
}

// ComponentInjector 组件注入器
// 注意：
// * 不管组件是以值类型注册还是以指针类型注册，泛型参数 C 都是组件的值类型
// * D 是被依赖组件的类型，可能是指针
type ComponentInjector struct {
	Qualifier string
	Required  bool
	InjectFn  func(any, any)
	DepType   reflect.Type
}

func (f ComponentInjector) inject(ctx *AppContext, comp any) {
	var dep any
	if f.Qualifier != "" {
		dep = ctx.GetComponentByName(f.Qualifier, f.Required)
	} else {
		dep = ctx.GetComponent(f.DepType, f.Required)
	}

	f.InjectFn(comp, dep)
}

// ValueInjector 值注入器
type ValueInjector[C any] struct {
	Scope    string
	Key      string
	Required bool
	InjectFn func(*C, any)
}

func (i ValueInjector[C]) inject(ctx *AppContext, comp *C) {
	value, exist := ctx.properties.get(i.Scope, i.Key)
	if !exist && i.Required {
		panic(errValueNotFound(i.Scope, i.Key))
	}

	i.InjectFn(comp, value)
}

// EnvInjector 环境变量注入器
type EnvInjector[C any] struct {
	Key          string
	Required     bool
	DefaultValue string
	InjectFn     func(*C, string)
}

func (i EnvInjector[C]) inject(ctx *AppContext, comp *C) {
	ev := ctx.environmentVariables.get(i.Key, i.DefaultValue, i.Required)
	i.InjectFn(comp, ev)
}
