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

	var (
		primaryMatches []*ContainerNode
		otherMatches   []*ContainerNode
	)

	for _, comp := range comps {
		if !ctx.match(comp.factory.GetCondition()) {
			continue
		}

		if comp.factory.IsPrimary() {
			primaryMatches = append(primaryMatches, comp)
		} else {
			otherMatches = append(otherMatches, comp)
		}
	}

	if len(primaryMatches) == 1 {
		return ctx.getInstance(primaryMatches[0])
	}

	if len(primaryMatches) > 1 {
		panic(errMultiPrimaryMatch(typeName))
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
	if comp == nil {
		return nil
	}

	return ctx.getInstance(comp)
}

func (ctx *AppContext) match(cond *Condition) bool {
	if cond == nil {
		return true
	}

	v, exist := ctx.properties.get(cond.Scope, cond.Key)
	if !exist {
		return false
	}

	s := fmt.Sprintf("%+v", v)
	return exist && cond.Value == s
}

func (ctx *AppContext) getInstance(component *ContainerNode) any {
	if component == nil {
		return nil
	}

	if component.instance == nil {
		if component.building {
			panic(errCircularDependency(component.factory.GetAlias()))
		}

		component.building = true
		component.instance = component.factory.build(ctx)
		component.building = false
	}

	return component.instance
}
