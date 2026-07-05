package provider

import (
	"fmt"
	"strings"
	"sync"
)

// Registry manages the set of registered ResourceProviders.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]ResourceProvider
	aliases   map[string]string // Maps short names to the canonical provider name
}

// NewRegistry creates a new empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]ResourceProvider),
		aliases:   make(map[string]string),
	}
}

// Register registers a ResourceProvider. It returns an error if the provider name or
// aliases conflict with existing providers.
func (r *Registry) Register(p ResourceProvider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := strings.ToLower(p.GetResourceType())
	if _, exists := r.providers[name]; exists {
		return fmt.Errorf("provider already registered for resource type: %s", p.GetResourceType())
	}

	r.providers[name] = p

	// Register short names
	for _, alias := range p.GetShortNames() {
		aliasLower := strings.ToLower(alias)
		if existing, exists := r.aliases[aliasLower]; exists {
			return fmt.Errorf("alias '%s' for '%s' conflicts with existing provider '%s'", alias, name, existing)
		}
		r.aliases[aliasLower] = name
	}

	return nil
}

// Get finds a provider by its resource type name or any of its short-name aliases.
func (r *Registry) Get(query string) (ResourceProvider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	queryLower := strings.ToLower(strings.TrimSpace(query))

	// Try lookup by direct name
	if p, exists := r.providers[queryLower]; exists {
		return p, true
	}

	// Try lookup by alias
	if canonicalName, exists := r.aliases[queryLower]; exists {
		if p, exists := r.providers[canonicalName]; exists {
			return p, true
		}
	}

	return nil, false
}

// ListProviders returns all registered resource providers.
func (r *Registry) ListProviders() []ResourceProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]ResourceProvider, 0, len(r.providers))
	for _, p := range r.providers {
		list = append(list, p)
	}
	return list
}
