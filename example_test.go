package di

import (
	"fmt"
	"strings"
	"sync/atomic"
)

type exampleDB struct {
	DSN string
}

type exampleRepo struct {
	DB *exampleDB
}

type exampleNamedTarget struct {
	DB *exampleDB
}

type exampleScoped struct {
	ID int32
}

func prepareExample() func() {
	resetContainersForTest()
	return resetContainersForTest
}

func ExampleProvideTo() {
	defer prepareExample()()
	container := NewContainer()

	MustProvideTo[*exampleDB](container, func() *exampleDB {
		return &exampleDB{DSN: "postgres://main"}
	})
	MustProvideTo[*exampleRepo](container, func(db *exampleDB) *exampleRepo {
		return &exampleRepo{DB: db}
	})

	repo, err := ResolveFrom[*exampleRepo](container)
	if err != nil {
		panic(err)
	}

	fmt.Println(repo.DB.DSN)
	// Output:
	// postgres://main
}

func ExampleProvideNamedTo() {
	defer prepareExample()()
	container := NewContainer()

	MustProvideNamedTo[*exampleDB](container, "primary", func() *exampleDB {
		return &exampleDB{DSN: "primary"}
	})
	MustProvideTo[*exampleNamedTarget](container, func(db *exampleDB) *exampleNamedTarget {
		return &exampleNamedTarget{DB: db}
	}, WithDeps(Named[*exampleDB]("primary")))

	target, err := ResolveFrom[*exampleNamedTarget](container)
	if err != nil {
		panic(err)
	}

	fmt.Println(target.DB.DSN)
	// Output:
	// primary
}

func ExampleContainer_NewScope() {
	defer prepareExample()()
	container := NewContainer()

	var seq atomic.Int32
	MustProvideTo[*exampleScoped](container, func() *exampleScoped {
		return &exampleScoped{ID: seq.Add(1)}
	}, WithLifetime(LifetimeScoped))

	scope1 := container.MustNewScope()
	scope2 := container.MustNewScope()
	defer scope1.Close()
	defer scope2.Close()

	first, _ := ResolveInScope[*exampleScoped](scope1)
	second, _ := ResolveInScope[*exampleScoped](scope1)
	third, _ := ResolveInScope[*exampleScoped](scope2)

	fmt.Println(first == second)
	fmt.Println(first != third)
	// Output:
	// true
	// true
}

func ExampleContainer_DumpGraph() {
	defer prepareExample()()
	container := NewContainer()

	MustProvideTo[*exampleDB](container, func() *exampleDB {
		return &exampleDB{DSN: "graph"}
	})
	MustProvideTo[*exampleRepo](container, func(db *exampleDB) *exampleRepo {
		return &exampleRepo{DB: db}
	})

	dump, err := container.DumpGraph()
	if err != nil {
		panic(err)
	}
	dot, err := container.DumpGraphDOT()
	if err != nil {
		panic(err)
	}

	fmt.Println(strings.Contains(dump, "exampleRepo"))
	fmt.Println(strings.Contains(dot, "digraph di"))
	// Output:
	// true
	// true
}
