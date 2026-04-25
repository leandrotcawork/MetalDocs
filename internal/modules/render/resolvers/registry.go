package resolvers

type Registry struct {
	items map[string]ComputedResolver
}

func NewRegistry() *Registry {
	return &Registry{
		items: make(map[string]ComputedResolver),
	}
}

func (r *Registry) Register(cr ComputedResolver) {
	r.items[cr.Key()] = cr
}

func (r *Registry) Get(key string) (ComputedResolver, bool) {
	cr, ok := r.items[key]
	return cr, ok
}

func (r *Registry) Known() map[string]int {
	out := make(map[string]int, len(r.items))
	for key, cr := range r.items {
		out[key] = cr.Version()
	}
	return out
}
