package diagnostics_test

import (
	"fmt"

	di "github.com/pakasa-io/di"
	didiag "github.com/pakasa-io/di/diagnostics"
)

type exampleDB struct{}
type exampleRepo struct{ DB *exampleDB }

func ExampleExplain() {
	container := di.NewContainer()

	di.MustProvideTo[*exampleDB](container, func() *exampleDB { return &exampleDB{} })
	di.MustProvideTo[*exampleRepo](container, func(db *exampleDB) *exampleRepo {
		return &exampleRepo{DB: db}
	})

	explanation, err := didiag.Explain[*exampleRepo](container)
	if err != nil {
		panic(err)
	}

	fmt.Println(len(explanation.Dependencies))
	// Output:
	// 1
}
