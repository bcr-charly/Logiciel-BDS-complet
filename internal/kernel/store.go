package kernel

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Store gère les fichiers JSON plats de data/ : un fichier par collection,
// réécriture atomique (fichier temporaire + renommage).
type Store struct {
	Dir string
}

func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Store{Dir: dir}, nil
}

func (s *Store) path(name string) string {
	return filepath.Join(s.Dir, name+".json")
}

// Load lit la collection name dans v (pointeur vers un slice). Un fichier
// absent laisse v intact (collection vide).
func (s *Store) Load(name string, v any) error {
	b, err := os.ReadFile(s.path(name))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// Save réécrit la collection de façon atomique.
func (s *Store) Save(name string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	p := s.path(name)
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// Collection est une collection JSON persistée, manipulée en mémoire et
// sauvegardée à chaque mutation.
type Collection[T any] struct {
	mu    sync.Mutex
	store *Store
	name  string
	Items []T
}

func NewCollection[T any](store *Store, name string) (*Collection[T], error) {
	c := &Collection[T]{store: store, name: name}
	if err := store.Load(name, &c.Items); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Collection[T]) All() []T {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]T, len(c.Items))
	copy(out, c.Items)
	return out
}

func (c *Collection[T]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.Items)
}

func (c *Collection[T]) Find(pred func(T) bool) (T, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, it := range c.Items {
		if pred(it) {
			return it, true
		}
	}
	var zero T
	return zero, false
}

// UpdateWhere applique fn sur le premier élément satisfaisant pred et
// sauvegarde. Retourne l'élément modifié.
func (c *Collection[T]) UpdateWhere(pred func(T) bool, fn func(*T)) (T, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := range c.Items {
		if pred(c.Items[i]) {
			fn(&c.Items[i])
			if err := c.store.Save(c.name, c.Items); err != nil {
				return c.Items[i], true, err
			}
			return c.Items[i], true, nil
		}
	}
	var zero T
	return zero, false, nil
}

func (c *Collection[T]) Append(item T) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Items = append(c.Items, item)
	return c.store.Save(c.name, c.Items)
}

func (c *Collection[T]) DeleteWhere(pred func(T) bool) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := range c.Items {
		if pred(c.Items[i]) {
			c.Items = append(c.Items[:i], c.Items[i+1:]...)
			return true, c.store.Save(c.name, c.Items)
		}
	}
	return false, nil
}

// Save force la sauvegarde de l'état courant.
func (c *Collection[T]) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.store.Save(c.name, c.Items)
}
