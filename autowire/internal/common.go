package internal

import (
	"fmt"
	"reflect"
	"unsafe"

	"github.com/bytedance/gg/gvalue"
	"github.com/modern-go/reflect2"
)

type Type = reflect.Type

// TypeOf 获取反射类型
func TypeOf[T any]() Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}

func getTypeName[T any]() string {
	return getTypeNameT(TypeOf[T]())
}

// 如果是指针类型，持续解引用，直到得到底层值类型
func getTypeNameT(typ Type) string {
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	if typ.PkgPath() == "" {
		return typ.Name()
	}

	return fmt.Sprintf("%s.%s", typ.PkgPath(), typ.Name())
}

func getPtrUnExportField(s any, fieldName string) reflect.Value {
	v := reflect.ValueOf(s).Elem().FieldByName(fieldName)
	return reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem()
}

func setValueByFieldName(o any, fieldName string, val any) {
	v := getPtrUnExportField(o, fieldName)
	rv := reflect.ValueOf(val)

	if v.Kind() != rv.Kind() {
		panic("val != val")
	}

	v.Set(rv)
}

func required(r []bool) bool {
	return len(r) == 0 || r[0]
}

func cast[T any](v any) T {
	if reflect2.IsNil(v) {
		return gvalue.Zero[T]()
	}

	v2, ok := v.(T)
	if !ok {
		panic(fmt.Errorf("type cast failed, source type [%T] destination type [%s] are not compatible",
			v, getTypeName[T]()))
	}

	return v2
}
