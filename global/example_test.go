package global_test

import (
	"fmt"

	diglobal "github.com/pakasa-io/di/global"
)

type exampleDB struct{}
type exampleRepo struct{ DB *exampleDB }

func ExampleProvide() {
	_ = diglobal.Reset()
	defer diglobal.Reset()

	diglobal.MustProvide[*exampleDB](func() *exampleDB {
		return &exampleDB{}
	})
	diglobal.MustProvide[*exampleRepo](func(db *exampleDB) *exampleRepo {
		return &exampleRepo{DB: db}
	})

	repo := diglobal.MustResolve[*exampleRepo]()
	fmt.Println(repo.DB != nil)
	// Output:
	// true
}
