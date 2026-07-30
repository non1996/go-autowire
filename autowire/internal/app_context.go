package internal

import (
	"fmt"
	"sync"
)

type buildState struct {
	building map[*ContainerNode]struct{}
}

func (s *buildState) enter(component *ContainerNode) {
	if s.building == nil {
		s.building = make(map[*ContainerNode]struct{})
	}
	if _, exists := s.building[component]; exists {
		panic(errCircularDependency(component.factory.GetAlias()))
	}
	s.building[component] = struct{}{}
}

func (s *buildState) leave(component *ContainerNode) {
	delete(s.building, component)
}

type AppContext struct {
	components           ComponentContainer
	properties           properties
	environmentVariables environmentVariables
	stateMu              sync.RWMutex
	buildMu              sync.Mutex
}

func NewAppContext() *AppContext {
	return &AppContext{
		components:           NewContainer(),
		properties:           newProperties(),
		environmentVariables: newEnvironmentVariables(),
	}
}

func (ctx *AppContext) Register(factory IComponentFactory) any {
	ctx.stateMu.Lock()
	defer ctx.stateMu.Unlock()

	ctx.register(factory)
	return struct{}{}
}

func (ctx *AppContext) Inject(appFactory IComponentFactory) any {
	ctx.stateMu.RLock()
	defer ctx.stateMu.RUnlock()

	ctx.buildMu.Lock()
	defer ctx.buildMu.Unlock()

	return appFactory.build(ctx, &buildState{})
}

func (ctx *AppContext) GetComponent(typ Type, require ...bool) any {
	ctx.stateMu.RLock()
	defer ctx.stateMu.RUnlock()

	ctx.buildMu.Lock()
	defer ctx.buildMu.Unlock()

	return ctx.getComponent(typ, required(require), &buildState{})
}

func (ctx *AppContext) getComponent(typ Type, require bool, state *buildState) any {
	typeName := getTypeNameT(typ)

	comps := ctx.components.ListByTypeName(typeName)
	if len(comps) == 0 && require {
		panic(errComponentNotFound(typeName))
	}

	var (
		primaryMatches []*ContainerNode
		otherMatches   []*ContainerNode
	)

	for _, comp := range comps {
		if !ctx.active(comp.factory) {
			continue
		}

		if comp.factory.IsPrimary() {
			primaryMatches = append(primaryMatches, comp)
		} else {
			otherMatches = append(otherMatches, comp)
		}
	}

	if len(primaryMatches) == 1 {
		return ctx.getInstance(primaryMatches[0], state)
	}

	if len(primaryMatches) > 1 {
		aliases := make([]string, 0, len(primaryMatches))
		for _, match := range primaryMatches {
			aliases = append(aliases, match.factory.GetAlias())
		}
		panic(errMultiPrimaryMatch(typeName, aliases))
	}

	if len(otherMatches) == 1 {
		return ctx.getInstance(otherMatches[0], state)
	}

	if len(otherMatches) > 1 {
		panic(errMultiMatch)
	}

	if len(otherMatches) == 0 && require {
		panic(errComponentNotFound(typeName))
	}

	return nil
}

func (ctx *AppContext) GetComponentByName(name string, require ...bool) any {
	ctx.stateMu.RLock()
	defer ctx.stateMu.RUnlock()

	ctx.buildMu.Lock()
	defer ctx.buildMu.Unlock()

	return ctx.getComponentByName(name, required(require), &buildState{})
}

func (ctx *AppContext) getComponentByName(name string, require bool, state *buildState) any {
	comp := ctx.components.GetByAlias(name)
	if comp == nil || !ctx.active(comp.factory) {
		if require {
			panic(errComponentNotFound(name))
		}
		return nil
	}

	return ctx.getInstance(comp, state)
}

func (ctx *AppContext) active(factory IComponentFactory) bool {
	return ctx.match(factory.GetCondition())
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

func (ctx *AppContext) getInstance(component *ContainerNode, state *buildState) (instance any) {
	if component == nil {
		return nil
	}

	if component.err != nil {
		panic(component.err)
	}

	if component.instance != nil {
		return component.instance
	}

	state.enter(component)
	defer func() {
		state.leave(component)
		if recovered := recover(); recovered != nil {
			component.err = fmt.Errorf("failed building component [%s]: %v", component.factory.GetAlias(), recovered)
			panic(component.err)
		}
	}()

	component.instance = component.factory.build(ctx, state)
	return component.instance
}

func (ctx *AppContext) register(factory IComponentFactory) {
	switch factory.(type) {
	case PropertyFactory, *PropertyFactory:
		factory.onRegister(ctx)
	default:
		ctx.components.Register(factory)
		factory.onRegister(ctx)
	}
}
