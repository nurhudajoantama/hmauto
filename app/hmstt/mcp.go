package hmstt

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type listStatesByTypeInput struct {
	Type string `json:"type" jsonschema:"State type, e.g. switch"`
}

type getStateInput struct {
	Type string `json:"type" jsonschema:"State type, e.g. switch"`
	Key  string `json:"key"  jsonschema:"State key, e.g. modem"`
}

type createStateInput struct {
	Type        string `json:"type"        jsonschema:"State type, e.g. switch"`
	Key         string `json:"key"         jsonschema:"State key, e.g. modem"`
	Value       string `json:"value"       jsonschema:"State value, e.g. on or off"`
	Description string `json:"description" jsonschema:"Human-readable description of what this state controls, e.g. Controls the modem power switch"`
}

type setStateInput struct {
	Type        string  `json:"type"        jsonschema:"State type, e.g. switch"`
	Key         string  `json:"key"         jsonschema:"State key, e.g. modem"`
	Value       string  `json:"value"       jsonschema:"State value, e.g. on or off"`
	Description *string `json:"description" jsonschema:"Optional: update the description of this state"`
}

type patchStateInput struct {
	Type        string  `json:"type"        jsonschema:"State type, e.g. switch"`
	Key         string  `json:"key"         jsonschema:"State key, e.g. modem"`
	Value       *string `json:"value"       jsonschema:"Optional: new state value, e.g. on or off"`
	Description *string `json:"description" jsonschema:"Optional: new description for this state"`
}

func textResult(v any) *mcp.CallToolResult {
	b, _ := json.Marshal(v)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(b)}},
	}
}

func errResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}
}

// RegisterMCPTools registers all hmstt tools on the given MCP server.
func RegisterMCPTools(s *mcp.Server, svc *HmsttService) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_all_states",
		Description: "List all IoT states across every type. Each state includes a description explaining its purpose.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		entries, err := svc.GetAllStates(ctx)
		if err != nil {
			return errResult(err.Error()), nil, nil
		}
		data := make([]StateResponse, 0, len(entries))
		for _, e := range entries {
			data = append(data, entryToResponse(e))
		}
		return textResult(data), nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_states_by_type",
		Description: "List all IoT states for a given type (e.g. switch). Each state includes a description explaining its purpose.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input listStatesByTypeInput) (*mcp.CallToolResult, any, error) {
		entries, err := svc.GetAllByType(ctx, input.Type)
		if err != nil {
			return errResult(err.Error()), nil, nil
		}
		data := make([]StateResponse, 0, len(entries))
		for _, e := range entries {
			data = append(data, entryToResponse(e))
		}
		return textResult(data), nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_state",
		Description: "Get a single IoT state by type and key, including its description.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input getStateInput) (*mcp.CallToolResult, any, error) {
		entry, err := svc.GetState(ctx, input.Type, input.Key)
		if err != nil {
			return errResult(err.Error()), nil, nil
		}
		return textResult(entryToResponse(entry)), nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_state",
		Description: "Create a new IoT state entry with a description. Returns error if the key already exists.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input createStateInput) (*mcp.CallToolResult, any, error) {
		if err := svc.CreateState(ctx, input.Type, input.Key, input.Value, input.Description); err != nil {
			return errResult(err.Error()), nil, nil
		}
		entry, err := svc.GetState(ctx, input.Type, input.Key)
		if err != nil {
			return errResult(err.Error()), nil, nil
		}
		return textResult(entryToResponse(entry)), nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "set_state",
		Description: "Update the value of an existing IoT state. Optionally update the description. MQTT event is fired only if the value changes.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input setStateInput) (*mcp.CallToolResult, any, error) {
		if err := svc.SetState(ctx, input.Type, input.Key, input.Value, input.Description); err != nil {
			return errResult(err.Error()), nil, nil
		}
		entry, err := svc.GetState(ctx, input.Type, input.Key)
		if err != nil {
			return errResult(err.Error()), nil, nil
		}
		return textResult(entryToResponse(entry)), nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "patch_state",
		Description: "Partially update an IoT state. Provide value, description, or both — fields not provided are left unchanged. MQTT event is fired only if the value changes.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input patchStateInput) (*mcp.CallToolResult, any, error) {
		if err := svc.PatchState(ctx, input.Type, input.Key, input.Value, input.Description); err != nil {
			return errResult(err.Error()), nil, nil
		}
		entry, err := svc.GetState(ctx, input.Type, input.Key)
		if err != nil {
			return errResult(err.Error()), nil, nil
		}
		return textResult(entryToResponse(entry)), nil, nil
	})
}
