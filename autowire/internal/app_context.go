package internal

import (
	"fmt"
)

type AppContext struct {
	components           ComponentContainer
	properties           properties
	environmentVariables environmentVariables
}

func NewAppContext() *AppContext {
	return &AppContext{
		components:           NewContainer(),
		properties:           newProperties(),
		environmentVariables: newEnvironmentVariables(),
	}
}

func (ctx *AppContext) Register(factory IComponentFactory) any {
	ctx.components.Register(factory)
	return struct{}{}
}

func (ctx *AppContext) Inject(appFactory IComponentFactory) any {
	return appFactory.build(ctx)
}

func (ctx *AppContext) GetComponent(typ Type, require ...bool) any {
	typeName := getTypeNameT(typ)

	comps := ctx.components.ListByTypeName(typeName)
	if len(comps) == 0 && required(require) {
		panic(errComponentNotFound(typeName))
	}
	if len(comps) == 1 {
		return ctx.getInstance(comps[0])
	}

	var (
		primary      *ContainerNode
		otherMatches []*ContainerNode
	)

	for _, comp := range comps {
		if comp.factory.IsPrimary() {
			primary = comp
		} else if ctx.match(comp.factory.GetCondition()) {
			otherMatches = append(otherMatches, comp)
		}
	}

	if primary != nil {
		return ctx.getInstance(primary)
	}

	if len(otherMatches) == 1 {
		return ctx.getInstance(otherMatches[0])
	}

	if len(otherMatches) > 1 {
		panic(errMultiMatch)
	}

	if len(otherMatches) == 0 && required(require) {
		panic(errComponentNotFound(typeName))
	}

	return nil
}

func (ctx *AppContext) GetComponentByName(name string, require ...bool) any {
	comp := ctx.components.GetByAlias(name)
	if comp == nil && required(require) {
		panic(errComponentNotFound(name))
	}

	return ctx.getInstance(comp)
}

func (ctx *AppContext) match(cond *Condition) bool {
	if cond == nil {
		return false
	}

	v, exist := ctx.properties.get(cond.Scope, cond.Key)
	if !exist {
		return false
	}

	s := fmt.Sprintf("%+v", v)
	return exist && cond.Value == s
}

func (ctx *AppContext) getInstance(component *ContainerNode) any {
	if component.instance == nil {
		component.instance = component.factory.build(ctx)
	}

	return component.instance
}
