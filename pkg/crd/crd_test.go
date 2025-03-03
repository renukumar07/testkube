package crd

import (
	"testing"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"

	"github.com/stretchr/testify/assert"
)

func TestGenerateYAML(t *testing.T) {

	t.Run("generate single CRD yaml", func(t *testing.T) {
		// given
		expected := "apiVersion: executor.testkube.io/v1\nkind: Webhook\nmetadata:\n  name: name1\n  namespace: namespace1\n  labels:\n    key1: value1\nspec:\n  events:\n  - start-test\n  uri: http://localhost\n  selector: app=backend\n"
		webhooks := []testkube.Webhook{
			{
				Name:      "name1",
				Namespace: "namespace1",
				Uri:       "http://localhost",
				Events:    []testkube.EventType{*testkube.EventStartTest},
				Selector:  "app=backend",
				Labels:    map[string]string{"key1": "value1"},
			},
		}

		// when
		result, err := GenerateYAML[testkube.Webhook](TemplateWebhook, webhooks)

		// then
		assert.NoError(t, err)
		assert.Equal(t, expected, result)
	})

	t.Run("generate multiple CRDs yaml", func(t *testing.T) {
		// given
		expected := "apiVersion: executor.testkube.io/v1\nkind: Webhook\nmetadata:\n  name: name1\n  namespace: namespace1\n  labels:\n    key1: value1\nspec:\n  events:\n  - start-test\n  uri: http://localhost\n  selector: app=backend\n\n---\napiVersion: executor.testkube.io/v1\nkind: Webhook\nmetadata:\n  name: name2\n  namespace: namespace2\n  labels:\n    key2: value2\nspec:\n  events:\n  - end-test-success\n  uri: http://localhost\n  selector: app=backend\n"
		webhooks := []testkube.Webhook{
			{
				Name:      "name1",
				Namespace: "namespace1",
				Uri:       "http://localhost",
				Events:    []testkube.EventType{*testkube.EventStartTest},
				Selector:  "app=backend",
				Labels:    map[string]string{"key1": "value1"},
			},
			{
				Name:      "name2",
				Namespace: "namespace2",
				Uri:       "http://localhost",
				Events:    []testkube.EventType{*testkube.EventEndTestSuccess},
				Selector:  "app=backend",
				Labels:    map[string]string{"key2": "value2"},
			},
		}

		// when
		result, err := GenerateYAML[testkube.Webhook](TemplateWebhook, webhooks)

		// then
		assert.NoError(t, err)
		assert.Equal(t, expected, result)
	})
	t.Run("generate executor CRD yaml", func(t *testing.T) {
		// given
		expected := "apiVersion: executor.testkube.io/v1\nkind: Executor\nmetadata:\n  name: name1\n  namespace: namespace1\n  labels:\n    key1: value1\nspec:\n  types:\n  - custom-curl-container/test\n  executor_type: container\n  image: docker.io/curlimages/curl:latest\n  args:\n  - -v\n  - test\n  command:\n  - curl\n  imagePullSecrets:\n  - name: secret-name\n"
		executors := []testkube.ExecutorCreateRequest{
			{
				Namespace:    "namespace1",
				Name:         "name1",
				ExecutorType: "container",
				Image:        "docker.io/curlimages/curl:latest",
				ImagePullSecrets: []testkube.LocalObjectReference{{
					Name: "secret-name",
				}},
				Command: []string{"curl"},
				Args:    []string{"-v", "test"},
				Types:   []string{"custom-curl-container/test"},
				Labels: map[string]string{
					"key1": "value1",
				},
				Features: []string{},
			},
		}

		// when
		result, err := GenerateYAML[testkube.ExecutorCreateRequest](TemplateExecutor, executors)

		// then
		assert.NoError(t, err)
		assert.Equal(t, expected, result)
	})
	t.Run("generate test CRD yaml", func(t *testing.T) {
		// given
		expected := "apiVersion: tests.testkube.io/v3\nkind: Test\nmetadata:\n  name: name1\n  namespace: namespace1\n  labels:\n    key1: value1\nspec:\n  executionRequest:\n    name: execution-name\n    args:\n      - -v\n      - test\n    image: docker.io/curlimages/curl:latest\n    command:\n    - curl\n    imagePullSecrets:\n    - name: secret-name\n"
		tests := []testkube.TestUpsertRequest{
			{
				Name:      "name1",
				Namespace: "namespace1",
				Labels: map[string]string{
					"key1": "value1",
				},
				ExecutionRequest: &testkube.ExecutionRequest{
					Name:  "execution-name",
					Image: "docker.io/curlimages/curl:latest",
					ImagePullSecrets: []testkube.LocalObjectReference{{
						Name: "secret-name",
					}},
					Command: []string{"curl"},
					Args:    []string{"-v", "test"},
				},
			},
		}

		// when
		result, err := GenerateYAML[testkube.TestUpsertRequest](TemplateTest, tests)

		// then
		assert.NoError(t, err)
		assert.Equal(t, expected, result)
	})

}
