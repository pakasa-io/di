package di

import (
	"context"
	"sync/atomic"
	"testing"
)

type testService struct {
	ID string
}

type testDependent struct {
	Service *testService
}

type testAutoWireDeps struct {
	Service *testService
}

type testAutoWireConsumer struct {
	Deps testAutoWireDeps
}

type testOptionalDependent struct {
	Service Optional[*testService]
}

type testOptionalBrokenConsumer struct {
	Service Optional[*testService]
}

type optionalFillTarget struct {
	Default Optional[*testService]
	Named   Optional[*testService] `di:"name=named"`
	Missing Optional[*testDependent]
}

type optionalBuiltinFillTarget struct {
	Ctx       Optional[context.Context]
	Container Optional[*Container]
}

type testAlias interface {
	AliasID() string
}

type testAliasImpl struct{}

func (testAliasImpl) AliasID() string {
	return "alias"
}

type testAliasImpl2 struct{}

func (testAliasImpl2) AliasID() string {
	return "alias-2"
}

type testLifecycle struct {
	preCount  *atomic.Int32
	postCount *atomic.Int32
}

func (t *testLifecycle) PreConstruct() {
	t.preCount.Add(1)
}

func (t *testLifecycle) PostConstruct() {
	t.postCount.Add(1)
}

type testClosable struct {
	closed *atomic.Int32
}

func (t *testClosable) Close() error {
	t.closed.Add(1)
	return nil
}

type fillTarget struct {
	Default   *testService
	Named     *testService `di:"name=named"`
	Container *Container
}

type contextFillTarget struct {
	Ctx     context.Context
	Service *testService
}

type testGroupValue struct {
	ID int
}

type testScopedValue struct {
	Seq int32
}

type testAppSingleton struct {
	Scoped *testScopedValue
}

type testCloseA struct{}

type testCloseZ struct{}

type testContextValue struct {
	Value string
}

type testContextKey struct{}

func resetContainersForTest() {
	ClearRuntimeCaches()
}

func prepareTest(t *testing.T) {
	t.Helper()
	resetContainersForTest()
	t.Cleanup(resetContainersForTest)
}

func newTestContainer() *Container {
	return NewContainer()
}

func bindInContainer[T any](c *Container) *BindingBuilder[T] {
	return &BindingBuilder[T]{
		container: c,
		binding:   c.bindType(getType[T]()),
	}
}

func resolveFromContainer[T any](c *Container) (T, error) {
	var zero T

	value, err := c.resolveByType(getType[T](), c.scope, newResolutionState())
	if err != nil {
		return zero, err
	}

	return value.Interface().(T), nil
}
