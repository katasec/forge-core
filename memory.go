package forge

import (
	memorypkg "github.com/katasec/forge-core/memory"
	"github.com/katasec/forge-core/memory/inmem"
)

type MemoryStore = memorypkg.Store
type InMemoryStore = inmem.Store

func NewInMemoryStore() *InMemoryStore {
	return inmem.New()
}
