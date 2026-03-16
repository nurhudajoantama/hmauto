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
	Type  string `json:"type"  jsonschema:"State type, e.g. switch"`
	Key   string `json:"key"   jsonschema:"State key, e.g. modem"`
	Value string `json:"value" jsonschema:"State value, e.g. on or off"`
}

type setStateInput struct {
	Type  string `json:"type"  jsonschema:"State type, e.g. switch"`
	Key   string `json:"key"   jsonschema:"State key, e.g. modem"`
	Value string `json:"value" jsonschema:"State value, e.g. on or off"`
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
		Description: "List all IoT states across every type",
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
		Description: "List all IoT states for a given type (e.g. switch)",
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
		Description: "Get a single IoT state by type and key",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input getStateInput) (*mcp.CallToolResult, any, error) {
		entry, err := svc.GetState(ctx, input.Type, input.Key)
		if err != nil {
			return errResult(err.Error()), nil, nil
		}
		return textResult(entryToResponse(entry)), nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_state",
		Description: "Create a new IoT state entry. Returns 409 if the key already exists.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input createStateInput) (*mcp.CallToolResult, any, error) {
		if err := svc.CreateState(ctx, input.Type, input.Key, input.Value); err != nil {
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
		Description: "Update the value of an existing IoT state by type and key",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input setStateInput) (*mcp.CallToolResult, any, error) {
		if err := svc.SetState(ctx, input.Type, input.Key, input.Value); err != nil {
			return errResult(err.Error()), nil, nil
		}
		entry, err := svc.GetState(ctx, input.Type, input.Key)
		if err != nil {
			return errResult(err.Error()), nil, nil
		}
		return textResult(entryToResponse(entry)), nil, nil
	})
}
