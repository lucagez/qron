package executor

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHttpExecutor(t *testing.T) {
	t.Run("Should perform http requests", func(t *testing.T) {
		exe := NewHttpExecutor(5)

		job := Job{
			Status: PENDING,
			Kind:   TASK,
			Config: `{"url":"https://jsonplaceholder.typicode.com/todos/12", "method":"GET"}`,
			State:  "",
		}

		err := exe.Run(&job)

		assert.Nil(t, err)
	})
}
