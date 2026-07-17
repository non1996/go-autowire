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
	f.condition = parseCondition(expression)
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
	tag.required = true

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

func GetPtrUnExportField(s any, fieldName string) reflect.Value {
	v := reflect.ValueOf(s).Elem().FieldByName(fieldName)
	return reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem()
}

func SetPtrUnExportField(s any, fieldName string, val any) {
	v := GetPtrUnExportField(s, fieldName)
	if val == nil {
		v.Set(reflect.Zero(v.Type()))
		return
	}

	rv := reflect.ValueOf(val)
	if rv.Type().AssignableTo(v.Type()) {
		v.Set(rv)
		return
	}

	if rv.Type().ConvertibleTo(v.Type()) {
		v.Set(rv.Convert(v.Type()))
		return
	}

	panic(fmt.Errorf("[autowire] field [%s] cannot assign value type [%s] to [%s]", fieldName, rv.Type(), v.Type()))
}
