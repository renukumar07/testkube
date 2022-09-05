package client

import (
	"io"
	"net/http"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/output"
)

// ResultEvent event passed when watching execution changes
type ResultEvent struct {
	Result testkube.ExecutionResult
	Error  error
}

// Executor abstraction to implement new executors
type Executor interface {
	// Execute starts new external test execution, reads data and returns ID
	// execution is started asynchronously client can check later for results
	Execute(execution *testkube.Execution, options ExecuteOptions) (result testkube.ExecutionResult, err error)

	// Execute starts new external test execution, reads data and returns ID
	// execution is started synchronously client is blocked
	ExecuteSync(execution *testkube.Execution, options ExecuteOptions) (result testkube.ExecutionResult, err error)

	// Abort aborts pending execution, do nothing when there is no pending execution
	Abort(id string) (execution *testkube.ExecutionResult)

	Logs(execution *testkube.Execution) (logs chan output.Output, err error)
}

// HTTPClient interface for getting REST based requests
type HTTPClient interface {
	Post(url, contentType string, body io.Reader) (resp *http.Response, err error)
	Get(url string) (resp *http.Response, err error)
}
