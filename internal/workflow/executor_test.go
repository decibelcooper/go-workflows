package workflow

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/cschleiden/go-dt/internal/command"
	"github.com/cschleiden/go-dt/pkg/converter"
	"github.com/cschleiden/go-dt/pkg/core"
	"github.com/cschleiden/go-dt/pkg/core/task"
	"github.com/cschleiden/go-dt/pkg/history"
	"github.com/stretchr/testify/require"
)

func Test_ExecuteWorkflow(t *testing.T) {
	r := NewRegistry()

	var workflowHits int

	Workflow1 := func(ctx Context) error {
		workflowHits++

		return nil
	}

	r.RegisterWorkflow("w1", Workflow1)

	task := &task.Workflow{
		WorkflowInstance: core.NewWorkflowInstance("instanceID", "executionID"),
		History: []history.HistoryEvent{
			history.NewHistoryEvent(
				history.HistoryEventType_WorkflowExecutionStarted,
				-1,
				history.ExecutionStartedAttributes{
					Name:    "w1",
					Version: "",
					Inputs:  [][]byte{},
				},
			),
		},
	}

	e := &executor{
		registry: r,
		task:     task,
		workflow: NewWorkflow(reflect.ValueOf(Workflow1)),
	}

	e.ExecuteWorkflowTask(context.Background())

	require.Equal(t, 1, workflowHits)
	require.True(t, e.workflow.Completed())
	require.Len(t, e.workflow.context.commands, 1)
}

func Test_ReplayWorkflowWithActivityResult(t *testing.T) {
	r := NewRegistry()

	var workflowHit int

	Workflow1 := func(ctx Context) error {
		workflowHit++

		f1, err := ctx.ExecuteActivity("a1", 42)
		if err != nil {
			panic("error executing activity 1")
		}

		var r int
		err = f1.Get(ctx, &r)
		if err != nil {
			panic("error getting activity 1 result")
		}

		workflowHit++

		return nil
	}
	Activity1 := func(ctx Context, r int) (int, error) {
		fmt.Println("Entering Activity1")

		return r, nil
	}

	r.RegisterWorkflow("w1", Workflow1)
	r.RegisterActivity("a1", Activity1)

	inputs, _ := converter.DefaultConverter.To(42)
	result, _ := converter.DefaultConverter.To(42)

	task := &task.Workflow{
		WorkflowInstance: core.NewWorkflowInstance("instanceID", "executionID"),
		History: []history.HistoryEvent{
			history.NewHistoryEvent(
				history.HistoryEventType_WorkflowExecutionStarted,
				-1,
				history.ExecutionStartedAttributes{
					Name:    "w1",
					Version: "",
					Inputs:  [][]byte{inputs},
				},
			),
			history.NewHistoryEvent(
				history.HistoryEventType_ActivityScheduled,
				0,
				history.ActivityScheduledAttributes{
					Name:    "a1",
					Version: "",
					Inputs:  [][]byte{inputs},
				},
			),
			history.NewHistoryEvent(
				history.HistoryEventType_ActivityCompleted,
				0,
				history.ActivityCompletedAttributes{
					Result: result,
				},
			),
		},
	}

	e := &executor{
		registry: r,
		task:     task,
		workflow: NewWorkflow(reflect.ValueOf(Workflow1)),
	}

	e.ExecuteWorkflowTask(context.Background())

	require.Equal(t, 2, workflowHit)
	require.True(t, e.workflow.Completed())
	require.Len(t, e.workflow.context.commands, 1)
}

func Test_ExecuteWorkflowWithActivityCommand(t *testing.T) {
	r := NewRegistry()

	var workflowHits int

	Workflow1 := func(ctx context.Context) error {
		workflowHits++

		f1, err := ctx.ExecuteActivity("a1", 42)
		if err != nil {
			panic("error executing activity 1")
		}

		var r int
		err = f1.Get(ctx, &r)
		if err != nil {
			panic("error getting activity 1 result")
		}

		workflowHits++

		return nil
	}
	Activity1 := func(ctx Context, r int) (int, error) {
		fmt.Println("Entering Activity1")

		return r, nil
	}

	r.RegisterWorkflow("w1", Workflow1)
	r.RegisterActivity("a1", Activity1)

	task := &task.Workflow{
		WorkflowInstance: core.NewWorkflowInstance("instanceID", "executionID"),
		History: []history.HistoryEvent{
			history.NewHistoryEvent(
				history.HistoryEventType_WorkflowExecutionStarted,
				-1,
				history.ExecutionStartedAttributes{
					Name:    "w1",
					Version: "",
					Inputs:  [][]byte{},
				},
			),
		},
	}

	e := &executor{
		registry: r,
		task:     task,
		workflow: NewWorkflow(reflect.ValueOf(Workflow1)),
	}

	e.ExecuteWorkflowTask(context.Background())

	require.Equal(t, 1, workflowHits)

	require.Len(t, e.workflow.context.commands, 1)

	inputs, _ := converter.DefaultConverter.To(42)
	require.Equal(t, command.Command{
		ID:   0,
		Type: command.CommandType_ScheduleActivityTask,
		Attr: command.ScheduleActivityTaskCommandAttr{
			Name:    "a1",
			Version: "",
			Inputs:  [][]byte{inputs},
		},
	}, e.workflow.context.commands[0])
}
