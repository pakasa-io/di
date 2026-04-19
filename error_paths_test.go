package di

import (
	"errors"
	"strings"
	"testing"
)

func assertErrorCode(t *testing.T, err error, want ErrorCode) *Error {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}

	var diErr *Error
	if !errors.As(err, &diErr) {
		t.Fatalf("expected *Error, got %T: %v", err, err)
	}
	if diErr.Code != want {
		t.Fatalf("expected error code %s, got %s: %v", want, diErr.Code, err)
	}
	return diErr
}

func assertPanicsWithErrorCode(t *testing.T, want ErrorCode, fn func()) {
	t.Helper()

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected panic")
		}

		err, ok := recovered.(error)
		if !ok {
			t.Fatalf("expected panic value to be an error, got %T", recovered)
		}
		assertErrorCode(t, err, want)
	}()

	fn()
}

type missingFillTarget struct {
	Service *testService
}

func TestBindWithoutFactoryReportsNoFactoryAcrossAPIs(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	builder, err := BindTo[*testService](container)
	if err != nil {
		t.Fatalf("Bind failed: %v", err)
	}
	if builder == nil {
		t.Fatal("expected BindTo to return a builder")
	}

	if !HasInContainer[*testService](container) {
		t.Fatal("expected explicit binding without factory to still be discoverable")
	}

	info, err := DescribeInContainer[*testService](container)
	if err != nil {
		t.Fatalf("Describe failed: %v", err)
	}
	if info.HasFactory {
		t.Fatalf("expected binding info to report a missing factory, got %#v", info)
	}

	explanation, err := ExplainFrom[*testService](container)
	if err != nil {
		t.Fatalf("Explain failed: %v", err)
	}
	if explanation.HasFactory {
		t.Fatalf("expected explanation to report a missing factory, got %#v", explanation)
	}

	_, err = ResolveFrom[*testService](container)
	diErr := assertErrorCode(t, err, ErrorCodeNoFactory)
	if !strings.Contains(diErr.Error(), cacheKey(getType[*testService](), "")) {
		t.Fatalf("expected resolve error trace to mention the incomplete binding, got %v", diErr)
	}

	err = container.Validate()
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	if len(validationErr.Issues) != 1 {
		t.Fatalf("expected a single validation issue, got %#v", validationErr.Issues)
	}
	assertErrorCode(t, validationErr.Issues[0], ErrorCodeNoFactory)
}

func TestInjectorErrorPathsRejectInvalidInputsAndWrapMissingDependencies(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()
	injector := NewInjector(container)

	if injector == nil {
		t.Fatal("expected NewInjector to return an injector")
	}

	_, err := injector.CallContext(nil, "not-a-function")
	assertErrorCode(t, err, ErrorCodeNotAFunction)
	assertErrorCode(t, injector.FillStruct(42), ErrorCodeInvalidStruct)

	var nilTarget *missingFillTarget
	assertErrorCode(t, injector.FillStruct(nilTarget), ErrorCodeInvalidStruct)

	var (
		number int
		target missingFillTarget
	)
	assertErrorCode(t, injector.FillStruct(&number), ErrorCodeInvalidStruct)

	_, err = injector.Call(func(s *testService) {})
	diErr := assertErrorCode(t, err, ErrorCodeDependencyInjectionFailed)
	if !strings.Contains(diErr.Error(), "failed to inject function arguments") {
		t.Fatalf("expected injector call error to describe the failed phase, got %v", diErr)
	}
	if !errors.Is(err, ErrDependencyResolution) || !errors.Is(err, ErrBindingNotFound) {
		t.Fatalf("expected injector call to preserve dependency-resolution causes, got %v", err)
	}

	err = injector.FillStruct(&target)
	diErr = assertErrorCode(t, err, ErrorCodeDependencyInjectionFailed)
	if !strings.Contains(diErr.Error(), "failed to inject field Service") {
		t.Fatalf("expected FillStruct error to name the failing field, got %v", diErr)
	}
	if !errors.Is(err, ErrBindingNotFound) {
		t.Fatalf("expected FillStruct error to preserve the missing binding cause, got %v", err)
	}
}

func TestInvokeAndScopeErrorPathsDifferentiateInvalidFunctionAndNilScope(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()
	scope := container.MustNewScope()
	defer scope.Close()

	assertErrorCode(t, InvokeOn(container, 123), ErrorCodeInvalidFunction)
	assertErrorCode(t, scope.Invoke(123), ErrorCodeInvalidFunction)

	var nilScope *Scope
	assertErrorCode(t, nilScope.Invoke(func() {}), ErrorCodeNilScope)
	assertErrorCode(t, nilScope.ValidateBindings(), ErrorCodeNilScope)
	_, err := ResolveInScope[*testService](nilScope)
	assertErrorCode(t, err, ErrorCodeNilScope)
}

func TestMustWrappersPanicWithUnderlyingErrors(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	assertPanicsWithErrorCode(t, ErrorCodeBindingNotFound, func() {
		_ = MustResolveFrom[*testService](container)
	})
	assertPanicsWithErrorCode(t, ErrorCodeNoBindingsFound, func() {
		_ = MustResolveGroupFrom[*testService](container, "missing")
	})
	assertPanicsWithErrorCode(t, ErrorCodeInvalidFactory, func() {
		MustProvideTo[*testService](container, func() string { return "wrong" })
	})

	var nilScope *Scope
	assertPanicsWithErrorCode(t, ErrorCodeNilScope, func() {
		_ = MustResolveInScope[*testService](nilScope)
	})
	assertPanicsWithErrorCode(t, ErrorCodeInvalidOption, func() {
		_ = MustNewOverlayContainer(nil)
	})
}
