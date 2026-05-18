package internal

import (
	"fmt"
	"reflect"

	"github.com/modern-go/reflect2"
	util "github.com/non1996/go-jsonobj/utils"
)

type propertyProvider struct {
	scope    string
	instance any
	provide  func() any
}

type properties struct {
	m      map[string]any // scope + "/" + key -> value
	scopes map[string]propertyProvider
}

func newProperties() properties {
	return properties{
		m:      map[string]any{},
		scopes: map[string]propertyProvider{},
	}
}

func (p *properties) add(provider propertyProvider) {
	p.scopes[provider.scope] = provider
}

func (p *properties) set(scope, key string, value any) {
	p.m[fmt.Sprintf("%s/%s", scope, key)] = value
}

func (p *properties) get(scope, key string) (any, bool) {
	provider, exist := p.scopes[scope]
	if !exist {
		return nil, false
	}

	if reflect2.IsNil(provider.instance) {
		provider.instance = provider.provide()
		kvs := objToKvPairs(provider.instance)

		for _, kv := range kvs {
			p.set(scope, kv.First, kv.Second)
		}
	}

	value, exist := p.m[fmt.Sprintf("%s/%s", scope, key)]
	return value, exist
}

func objToKvPairs(obj any) (res []util.Pair[string, any]) {
	v := deRefValue(reflect.ValueOf(obj))
	if v.Kind() != reflect.Struct {
		return nil
	}

	return objToKvImpl("", v)
}

func objToKvImpl(prefix string, v reflect.Value) (res []util.Pair[string, any]) {
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		fieldValue := v.Field(i)
		fieldType := t.Field(i)

		if !fieldType.IsExported() {
			continue
		}

		if fieldValue.Type().Kind() == reflect.Pointer {
			if fieldValue.IsNil() {
				continue
			}

			fieldValue = deRefValue(fieldValue)
		}

		if fieldValue.Type().Kind() == reflect.Struct {
			if fieldType.Anonymous {
				res = append(res, objToKvImpl(joinKey(prefix, fieldValue.Type().Name()), fieldValue)...)
			} else {
				res = append(res, objToKvImpl(joinKey(prefix, fieldType.Name), fieldValue)...)
			}
		} else if validConfigFieldKind(fieldValue.Type().Kind()) {
			if fieldType.Anonymous {
				res = append(res, util.NewPair(joinKey(prefix, fieldValue.Type().Name()), fieldValue.Interface()))
			} else {
				res = append(res, util.NewPair(joinKey(prefix, fieldType.Name), fieldValue.Interface()))
			}
		}
	}

	return res
}

func joinKey(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}

func deRefValue(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	return v
}

func validConfigFieldKind(kind reflect.Kind) bool {
	if kind > reflect.Invalid && kind < reflect.Complex64 {
		return true
	}
	return kind == reflect.String || kind == reflect.Slice || kind == reflect.Array
}
