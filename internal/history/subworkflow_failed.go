package history

import "github.com/cschleiden/go-workflows/internal/workflowerrors"

type SubWorkflowFailedAttributes struct {
	Error *workflowerrors.Error `json:"error,omitempty"`
}
