package explain

import (
	"github.com/irenedo/kubectl-inspect/pkg/kubectl"
)

// Fetcher runs kubectl explain for individual field paths.
type Fetcher struct {
	executor kubectl.Executor
	resource string
	flags    kubectl.Flags
}

// NewFetcher creates a new Fetcher.
func NewFetcher(executor kubectl.Executor, resource string, flags kubectl.Flags) *Fetcher {
	return &Fetcher{
		executor: executor,
		resource: resource,
		flags:    flags,
	}
}

// FetchDetail fetches the explain output for a specific field path.
// No caching — always fetches fresh from the API server.
func (f *Fetcher) FetchDetail(fieldPath string) DetailResult {
	var fullPath string
	if fieldPath == "" {
		fullPath = f.resource
	} else {
		fullPath = f.resource + "." + fieldPath
	}

	output, err := f.executor.ExplainField(fullPath, f.flags)
	return DetailResult{
		RawOutput: output,
		Err:       err,
	}
}
