# Testing With testutil

`testutil` gives you isolated named containers, scoped fixtures, temporary overrides, and cleanup hooks that attach to `testing.T`.

```go
package mypkg_test

import (
	"testing"

	di "github.com/pakasa-io/di"
	diglobal "github.com/pakasa-io/di/global"
	"github.com/pakasa-io/di/testutil"
)

type DB struct{}
type Repo struct {
	DB *DB
}

func TestRepo(t *testing.T) {
	name, container := testutil.Container(t)

	testutil.MustProvide[*DB](t, name, func() *DB {
		return &DB{}
	})
	testutil.MustProvide[*Repo](t, name, func(db *DB) *Repo {
		return &Repo{DB: db}
	})

	testutil.Override[*DB](t, container, func() (*DB, error) {
		return &DB{}, nil
	})

	repo, err := diglobal.ResolveIn[*Repo](name)
	if err != nil {
		t.Fatal(err)
	}
	if repo.DB == nil {
		t.Fatal("expected injected DB")
	}
}
```

Scoped test fixtures:

```go
fixture := testutil.ScopedFixture(t)

testutil.MustProvide[*RequestService](t, fixture.Name, func() *RequestService {
	return &RequestService{}
}, di.WithLifetime(di.LifetimeScoped))

service := di.MustResolveInScope[*RequestService](fixture.Scope)
_ = service
```

Use `testutil.Reset(t)` when a test needs a clean global registry and cleared reflection caches.
