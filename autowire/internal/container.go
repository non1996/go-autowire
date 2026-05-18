package internal

import (
	"github.com/non1996/go-jsonobj/stream"

	"github.com/non1996/go-autowire/autowire/internal/util"
)

// ContainerNode 组件容器节点
// factory: 组件工厂
// instance: 组件实例
type ContainerNode struct {
	factory  IComponentFactory
	instance any
}

// ComponentContainer 组件容器
type ComponentContainer struct {
	list       []*ContainerNode // 容器节点列表
	aliasIndex map[string]int   // 别名 -> 节点下标
	typeIndex  map[string][]int // 类型名 -> 节点下标列表
}

func NewContainer() ComponentContainer {
	return ComponentContainer{
		aliasIndex: map[string]int{},
		typeIndex:  map[string][]int{},
	}
}

// Register 注册工厂
func (c *ComponentContainer) Register(f IComponentFactory) {
	if util.MapContainsKey(c.aliasIndex, f.GetAlias()) {
		panic(errComponentDuplicate(f.GetAlias()))
	}

	idx := len(c.list)
	c.list = append(c.list, &ContainerNode{factory: f})

	// 别名索引
	c.aliasIndex[f.GetAlias()] = idx

	// 类型索引
	typeName := getTypeNameT(f.GetType())
	c.typeIndex[typeName] = append(c.typeIndex[typeName], idx)

	// 实现接口索引
	impls := f.GetImplement()
	for _, impl := range impls {
		implName := getTypeNameT(impl)
		c.typeIndex[implName] = append(c.typeIndex[implName], idx)
	}
}

// GetByAlias 根据别名获取容器节点
func (c *ComponentContainer) GetByAlias(name string) *ContainerNode {
	idx, exist := c.aliasIndex[name]
	if exist {
		return c.list[idx]
	}

	return nil
}

// ListByTypeName 根据类型名获取该类型下的所有容器
func (c *ComponentContainer) ListByTypeName(typeName string) []*ContainerNode {
	idxes := c.typeIndex[typeName]

	return stream.Map(idxes, func(idx int) *ContainerNode { return c.list[idx] })
}
