package autowire

import (
	"fmt"
	"reflect"
	"strconv"
	"unsafe"

	"github.com/non1996/go-autowire/autowire/internal"
	"github.com/non1996/go-autowire/autowire/internal/util"
)

func TypeOf[T any]() reflect.Type {
	return util.TypeOf[T]()
}

func Register(factories ...internal.IComponentFactory) any {
	for _, factory := range factories {
		defaultAppContext.Register(factory)
		// TODO 处理Bean
	}
	return nil
}

func Context() *internal.AppContext {
	return defaultAppContext
}

func GetComponent[T any](require ...bool) T {
	component := defaultAppContext.GetComponent(util.TypeOf[T](), require...)
	return util.Cast[T](component)
}

func GetComponentByName[T any](name string, require ...bool) T {
	component := defaultAppContext.GetComponentByName(name, require...)
	return util.Cast[T](component)
}

type ComponentFactoryBuilder[C any] struct {
	alias          string                 // 组件别名
	primary        bool                   // 是否“主”组件
	configuration  bool                   // 是否配置类组件
	typ            reflect.Type           // 组件的反射类型
	implement      []reflect.Type         // 实现接口的反射类型
	condition      *internal.Condition    // 组件初始化条件
	fieldInjectors []internal.Injector    // 依赖注入器
	postConstruct  internal.ConstructFunc // 构造器
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
	f.condition = &internal.Condition{
		Expression: expression,
	}
	return f
}

func (f *ComponentFactoryBuilder[C]) Configuration() *ComponentFactoryBuilder[C] {
	f.configuration = true
	return f
}

func (f *ComponentFactoryBuilder[C]) PostConstruct(constructFunc func(*C) error) *ComponentFactoryBuilder[C] {
	f.postConstruct = func(any any) error {
		return constructFunc(any.(*C))
	}

	return f
}

func (f *ComponentFactoryBuilder[C]) Register() internal.IComponentFactory {
	if f.typ.Kind() != reflect.Struct {
		panic(fmt.Errorf("[autowire] only support struct type component"))
	}

	// 未指定别名的情况下，使用struct名称作为别名
	if f.alias == "" {
		f.alias = f.typ.Name()
	}

	for i := 0; i < f.typ.NumField(); i++ {
		field := f.typ.Field(i)

		tag := parseFieldTag(field)
		if !tag.autowire {
			continue
		}

		f.fieldInjectors = append(f.fieldInjectors, internal.ComponentInjector{
			Qualifier: tag.qualifier,
			Required:  tag.required,
			DepType:   field.Type,
			InjectFn: func(component, dependency any) {
				SetPtrUnExportField(component, field.Name, dependency)
			},
		})
	}

	factory := &internal.StructFactory{
		Alias:          f.alias,
		Primary:        f.primary,
		Configuration:  f.configuration,
		Type:           f.typ,
		Implement:      f.implement,
		Condition:      f.condition,
		FieldInjectors: f.fieldInjectors,
		PostConstruct:  f.postConstruct,
	}

	return factory
}

type autowireTag struct {
	autowire  bool
	qualifier string
	required  bool
}

// 先只支持结构体注入
func parseFieldTag(field reflect.StructField) (tag autowireTag) {
	rawTag := field.Tag

	_autowire, exist := rawTag.Lookup("autowire")
	if !exist || _autowire != "true" {
		return autowireTag{}
	}

	tag.autowire = true

	qualifier, exist := rawTag.Lookup("qualifier")
	if exist && qualifier != "" {
		tag.qualifier = qualifier
	}

	required, exist := rawTag.Lookup("required")
	if exist && (required == "true" || required == "false") {
		tag.required, _ = strconv.ParseBool(required)
	}

	return tag
}

func GetPtrUnExportField(s any, fieldName string) reflect.Value {
	v := reflect.ValueOf(s).Elem().FieldByName(fieldName)
	return reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem()
}

func SetPtrUnExportField(s any, fieldName string, val any) {
	v := GetPtrUnExportField(s, fieldName)
	rv := reflect.ValueOf(val)

	v.Set(rv)
}
