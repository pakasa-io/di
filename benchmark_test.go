package di

import "testing"

func BenchmarkResolveSingleton(b *testing.B) {
	resetContainersForTest()
	b.Cleanup(resetContainersForTest)
	container := newTestContainer()

	MustProvideTo[*testService](container, func() *testService {
		return &testService{ID: "singleton"}
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ResolveFrom[*testService](container); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkResolveTransient(b *testing.B) {
	resetContainersForTest()
	b.Cleanup(resetContainersForTest)
	container := newTestContainer()

	MustProvideTo[*testService](container, func() *testService {
		return &testService{ID: "transient"}
	}, WithLifetime(LifetimeTransient))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ResolveFrom[*testService](container); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkResolveScoped(b *testing.B) {
	resetContainersForTest()
	b.Cleanup(resetContainersForTest)
	container := newTestContainer()

	MustProvideTo[*testService](container, func() *testService {
		return &testService{ID: "scoped"}
	}, WithLifetime(LifetimeScoped))

	scope := container.MustNewScope()
	defer scope.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ResolveInScope[*testService](scope); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkInvoke(b *testing.B) {
	resetContainersForTest()
	b.Cleanup(resetContainersForTest)
	container := newTestContainer()

	MustProvideTo[*testService](container, func() *testService {
		return &testService{ID: "invoke"}
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := InvokeOn(container, func(*testService) error { return nil }); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFillStruct(b *testing.B) {
	resetContainersForTest()
	b.Cleanup(resetContainersForTest)
	container := newTestContainer()

	MustProvideTo[*testService](container, func() *testService {
		return &testService{ID: "fill"}
	})
	MustProvideTo[*testService](container, func() *testService {
		return &testService{ID: "named"}
	}, WithName("named"))

	injector := container.Injector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var target fillTarget
		if err := injector.FillStruct(&target); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkResolveAll(b *testing.B) {
	resetContainersForTest()
	b.Cleanup(resetContainersForTest)
	container := newTestContainer()

	MustProvideTo[*testGroupValue](container, func() *testGroupValue {
		return &testGroupValue{ID: 1}
	}, WithGroup("bench"), WithName("one"), WithLifetime(LifetimeTransient))
	MustProvideTo[*testGroupValue](container, func() *testGroupValue {
		return &testGroupValue{ID: 2}
	}, WithGroup("bench"), WithName("two"), WithLifetime(LifetimeTransient))
	MustProvideTo[*testGroupValue](container, func() *testGroupValue {
		return &testGroupValue{ID: 3}
	}, WithGroup("bench"), WithName("three"), WithLifetime(LifetimeTransient))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		values, err := ResolveGroupFrom[*testGroupValue](container, "bench")
		if err != nil {
			b.Fatal(err)
		}
		if len(values) != 3 {
			b.Fatalf("unexpected aggregate size: %d", len(values))
		}
	}
}
