/*
 * Testkube API
 *
 * Testkube provides a Kubernetes-native framework for test definition, execution and results
 *
 * API version: 1.0.0
 * Contact: testkube@kubeshop.io
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */
package testkube

// test suite execution summary
type TestSuiteStepExecutionSummary struct {
	Id string `json:"id"`
	// execution name
	Name string `json:"name"`
	// test name
	TestName string             `json:"testName,omitempty"`
	Status   *ExecutionStatus   `json:"status"`
	Type_    *TestSuiteStepType `json:"type,omitempty"`
}
