package util

import (
	"fmt"
	"reflect"

	"github.com/modern-go/reflect2"
	"github.com/non1996/go-jsonobj/function"
)

type Type = reflect.Type

// TypeOf 获取反射类型
func TypeOf[T any]() Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}

// GetTypeName 获取类型名称
func GetTypeName[T any]() string {
	return GetTypeNameT(TypeOf[T]())
}

// GetTypeNameT 如果是指针类型，持续解引用，直到得到底层值类型
func GetTypeNameT(typ Type) string {
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	if typ.PkgPath() == "" {
		return typ.Name()
	}

	return fmt.Sprintf("%s.%s", typ.PkgPath(), typ.Name())
}

func MapContainsKey[K comparable, V any](m map[K]V, k K) bool {
	_, exist := m[k]
	return exist
}

// Ternary 三元表达式
func Ternary[T any](cond bool, v1, v2 T) T {
	if cond {
		return v1
	}
	return v2
}

func Cast[T any](v any) T {
	if reflect2.IsNil(v) {
		return function.Zero[T]()
	}

	v2, ok := v.(T)
	if !ok {
		panic(fmt.Errorf("type cast failed, source type [%T] destination type [%s] are not compatible",
			v, GetTypeName[T]()))
	}

	return v2
}
