// Package runtimesetting binds the configuration store to the IAM process.
package runtimesetting

import (
	"sync"
	"sync/atomic"

	"github.com/morehao/golib/gsetting"
	"gorm.io/gorm"
)

type Store struct {
	Setting *gsetting.Setting
	Admin   *gsetting.AdminAPI
	pool    gorm.ConnPool
}

var (
	constructorMu sync.Mutex
	processStore  atomic.Pointer[Store]
)

// New captures the admin handle alongside its store. gsetting publishes that
// handle through a package global, so construction and capture must be serialized.
func New(db *gorm.DB, opts ...gsetting.Option) (*Store, error) {
	constructorMu.Lock()
	defer constructorMu.Unlock()
	opts = append(opts, gsetting.WithoutAutoMigrate())
	setting, err := gsetting.New(db, opts...)
	if err != nil {
		return nil, err
	}
	return &Store{Setting: setting, Admin: gsetting.GetAdmin(), pool: db.Statement.ConnPool}, nil
}

func SetProcessStore(store *Store) {
	processStore.Store(store)
}

func ProcessStore() *Store {
	return processStore.Load()
}

// ForDB reuses the process cache for its connection pool. A transaction or
// a separate database gets an admin bound to that connection.
func ForDB(db *gorm.DB) (*Store, error) {
	store := processStore.Load()
	if db != nil && store != nil && store.pool == db.Statement.ConnPool {
		return store, nil
	}
	return New(db)
}
