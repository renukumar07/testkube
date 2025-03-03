package triggers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTriggerStatus(t *testing.T) {
	t.Parallel()

	status := newTriggerStatus()

	status.testSuiteExecutionIDs = []string{"test-suite-execution-1"}
	status.start()

	assert.True(t, status.hasActiveTests())
	assert.NotNil(t, status.lastExecutionStarted)
	assert.Nil(t, status.lastExecutionFinished)

	status.done()

	assert.False(t, status.hasActiveTests())
	assert.NotNil(t, status.lastExecutionStarted)
	assert.NotNil(t, status.lastExecutionFinished)
	assert.True(t, status.lastExecutionFinished.After(*status.lastExecutionStarted))
}
