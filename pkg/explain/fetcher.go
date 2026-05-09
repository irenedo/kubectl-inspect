package explain

import (
	"context"
	"sync"

	"github.com/irenedo/kubectl-inspect/pkg/kubectl"
)

// Fetcher runs kubectl explain for individual field paths.
// It caches results by field path so revisiting a node is instant.
type Fetcher struct {
	mu       sync.RWMutex
	executor kubectl.Executor
	resource string
	flags    kubectl.Flags
	cache    map[string]DetailResult
}

// NewFetcher creates a new Fetcher.
func NewFetcher(executor kubectl.Executor, resource string, flags kubectl.Flags) *Fetcher {
	return &Fetcher{
		executor: executor,
		resource: resource,
		flags:    flags,
		cache:    make(map[string]DetailResult),
	}
}

// FetchDetail fetches the explain output for a specific field path.
// Results are cached for the lifetime of the Fetcher.
func (f *Fetcher) FetchDetail(ctx context.Context, fieldPath string) DetailResult {
	f.mu.RLock()
	if cached, ok := f.cache[fieldPath]; ok {
		f.mu.RUnlock()
		return cached
	}
	f.mu.RUnlock()

	var fullPath string
	if fieldPath == "" {
		fullPath = f.resource
	} else {
		fullPath = f.resource + "." + fieldPath
	}

	output, err := f.executor.ExplainField(ctx, fullPath, f.flags)
	result := DetailResult{
		RawOutput: output,
		Err:       err,
	}
	f.mu.Lock()
	f.cache[fieldPath] = result
	f.mu.Unlock()
	return result
}
