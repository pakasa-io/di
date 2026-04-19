package builtin

import (
	"context"
	"reflect"

	diModel "github.com/pakasa-io/di/internal/model"
)

type Kind int

const (
	None Kind = iota
	Context
	Container
)

func ForDependency(dep diModel.Dependency, contextType, containerType reflect.Type) Kind {
	dep = diModel.NormalizeDependency(dep)
	return ForType(dep.Type, contextType, containerType)
}

func ForType(typ, contextType, containerType reflect.Type) Kind {
	switch typ {
	case contextType:
		return Context
	case containerType:
		return Container
	default:
		return None
	}
}

func Resolve(kind Kind, ctx context.Context, container any) reflect.Value {
	switch kind {
	case Context:
		if ctx == nil {
			ctx = context.Background()
		}
		return reflect.ValueOf(ctx)
	case Container:
		return reflect.ValueOf(container)
	default:
		return reflect.Value{}
	}
}
