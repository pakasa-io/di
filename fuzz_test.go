package di

import (
	"reflect"
	"strconv"
	"testing"
)

func FuzzParseStructTags(f *testing.F) {
	f.Add("")
	f.Add("name=primary")
	f.Add("name=primary,group=cache")
	f.Add("-")

	f.Fuzz(func(t *testing.T, tag string) {
		_ = parseStructTags(tag)
	})
}

func FuzzWithInterface(f *testing.F) {
	f.Add(uint8(0))
	f.Add(uint8(1))
	f.Add(uint8(2))
	f.Add(uint8(3))

	f.Fuzz(func(t *testing.T, selector uint8) {
		var (
			value     any
			expectErr bool
		)

		switch selector % 4 {
		case 0:
			value = (*testAlias)(nil)
		case 1:
			value = testAliasImpl{}
			expectErr = true
		case 2:
			value = (*testAliasImpl)(nil)
			expectErr = true
		default:
			value = nil
			expectErr = true
		}

		opt, err := WithInterface(value)
		if expectErr {
			if err == nil {
				t.Fatalf("expected error for %#v", value)
			}
			return
		}

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cfg, err := newConfig(opt)
		if err != nil {
			t.Fatalf("unexpected config error: %v", err)
		}
		if cfg.interfaceFor == nil || cfg.interfaceFor.Kind() != reflect.Interface {
			t.Fatalf("expected interface type to be recorded, got %#v", cfg.interfaceFor)
		}
	})
}

func FuzzValidateFactoryForBinding(f *testing.F) {
	f.Add(uint8(0))
	f.Add(uint8(1))
	f.Add(uint8(2))
	f.Add(uint8(3))
	f.Add(uint8(4))

	f.Fuzz(func(t *testing.T, selector uint8) {
		b := &binding{typ: getType[*testService]()}

		var (
			factory   any
			expectErr bool
		)

		switch selector % 5 {
		case 0:
			factory = func() *testService { return &testService{} }
		case 1:
			factory = func() string { return "nope" }
			expectErr = true
		case 2:
			factory = func() (*testService, int) { return &testService{}, 1 }
			expectErr = true
		case 3:
			b.dependencies = []Dependency{Dep[*testService]()}
			factory = func(*testService) (*testService, error) { return &testService{}, nil }
		default:
			b.dependencies = []Dependency{Dep[*testService]()}
			factory = func() *testService { return &testService{} }
			expectErr = true
		}

		err := validateFactoryForBinding(b, factory)
		if expectErr && err == nil {
			t.Fatal("expected validation error")
		}
		if !expectErr && err != nil {
			t.Fatalf("unexpected validation error: %v", err)
		}
	})
}

func FuzzFillStruct(f *testing.F) {
	f.Add("")
	f.Add("name=named")
	f.Add("-")
	f.Add("name==broken")

	f.Fuzz(func(t *testing.T, tag string) {
		resetContainersForTest()
		defer resetContainersForTest()
		container := newTestContainer()

		MustProvideTo[*testService](container, func() *testService {
			return &testService{ID: "default"}
		})
		MustProvideTo[*testService](container, func() *testService {
			return &testService{ID: "named"}
		}, WithName("named"))

		structType := reflect.StructOf([]reflect.StructField{
			{
				Name: "Service",
				Type: getType[*testService](),
				Tag:  reflect.StructTag(`di:` + strconv.Quote(tag)),
			},
			{
				Name: "Container",
				Type: containerPtrType,
			},
		})

		target := reflect.New(structType)
		_ = container.Injector().FillStruct(target.Interface())
	})
}
