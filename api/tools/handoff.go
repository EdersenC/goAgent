package tools

import (
	"github.com/EdersenC/gigaAi/api"
)



func NewHandOff(agent *api.Agent, parameters api.ToolParameters) *api.Tool {
    return &api.Tool{
        Type: "function",
        Function: api.ToolFunction{
            Name: "Handoff",
            Description: ``,
            Parameters: parameters,
        },
    }
}
