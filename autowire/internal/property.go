package internal

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/bytedance/gg/collection/tuple"
)

type propertyProvider struct {
	scope    string
	instance any
	provide  func() any
	once     sync.Once
}

type properties struct {
	m      map[string]any // scope + "/" + key -> value
	scopes map[string]*propertyProvider
	mu     sync.RWMutex
}

func newProperties() properties {
	return properties{
		m:      map[string]any{},
		scopes: map[string]*propertyProvider{},
	}
}

func (p *properties) add(provider *propertyProvider) {
	if provider == nil {
		panic(fmt.Errorf("property provider is nil"))
	}
	if provider.scope == "" {
		panic(fmt.Errorf("property scope is empty"))
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.scopes[provider.scope]; exists {
		panic(errPropertyDuplicate(provider.scope))
	}

	p.scopes[provider.scope] = provider
}

func (p *properties) setLocked(scope, key string, value any) {
	p.m[fmt.Sprintf("%s/%s", scope, key)] = value
}

func (p *properties) get(scope, key string) (any, bool) {
	p.mu.RLock()
	provider, exists := p.scopes[scope]
	p.mu.RUnlock()
	if !exists {
		return nil, false
	}

	provider.once.Do(func() {
		instance := provider.provide()
		kvs := objToKvPairs(instance)

		p.mu.Lock()
		defer p.mu.Unlock()

		provider.instance = instance
		for _, kv := range kvs {
			p.setLocked(scope, kv.First, kv.Second)
		}
	})

	p.mu.RLock()
	defer p.mu.RUnlock()
	value, exists := p.m[fmt.Sprintf("%s/%s", scope, key)]
	return value, exists
}

func objToKvPairs(obj any) (res []tuple.T2[string, any]) {
	v := deRefValue(reflect.ValueOf(obj))
	if v.Kind() != reflect.Struct {
		return nil
	}

	return objToKvImpl("", v)
}

func objToKvImpl(prefix string, v reflect.Value) (res []tuple.T2[string, any]) {
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
				res = append(res, tuple.Make2(joinKey(prefix, fieldValue.Type().Name()), fieldValue.Interface()))
			} else {
				res = append(res, tuple.Make2(joinKey(prefix, fieldType.Name), fieldValue.Interface()))
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
