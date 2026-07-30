package internal

import (
	"reflect"
)

// Injector 注入器
// 注意：不管组件是以值类型注册还是以指针类型注册，泛型参数 C 都是组件的值类型
type Injector interface {
	inject(*AppContext, *buildState, any)
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

func (f ComponentInjector) inject(ctx *AppContext, state *buildState, comp any) {
	var dep any
	if f.Qualifier != "" {
		dep = ctx.getComponentByName(f.Qualifier, f.Required, state)
	} else {
		dep = ctx.getComponent(f.DepType, f.Required, state)
	}

	f.InjectFn(comp, dep)
}

// ValueInjector 值注入器
type ValueInjector struct {
	Scope    string
	Key      string
	Required bool
	InjectFn func(any, any)
}

func (i ValueInjector) inject(ctx *AppContext, _ *buildState, comp any) {
	value, exist := ctx.properties.get(i.Scope, i.Key)
	if !exist && i.Required {
		panic(errValueNotFound(i.Scope, i.Key))
	}

	i.InjectFn(comp, value)
}

// EnvInjector 环境变量注入器
type EnvInjector struct {
	Key          string
	Required     bool
	DefaultValue string
	HasDefault   bool
	InjectFn     func(any, string)
}

func (i EnvInjector) inject(ctx *AppContext, _ *buildState, comp any) {
	ev, exists := ctx.environmentVariables.get(i.Key, i.DefaultValue, i.Required && !i.HasDefault)
	if !exists && !i.HasDefault {
		return
	}
	i.InjectFn(comp, ev)
}
