package schema

import (
	"encoding/json"
	"fmt"

	"google.golang.org/genai"
	"maragu.dev/gai"
)

// ConvertTools converts gai.Tool slice to genai.Tool slice.
func ConvertTools(tools []gai.Tool) ([]*genai.Tool, error) {
	var funcDecls []*genai.FunctionDeclaration
	for _, tool := range tools {
		funcDecl, err := ConvertToolToFunction(tool)
		if err != nil {
			return nil, fmt.Errorf("converting tool %s: %w", tool.Name, err)
		}
		funcDecls = append(funcDecls, funcDecl)
	}
	return []*genai.Tool{{FunctionDeclarations: funcDecls}}, nil
}

// ConvertToolToFunction converts a gai.Tool to genai.FunctionDeclaration.
func ConvertToolToFunction(tool gai.Tool) (*genai.FunctionDeclaration, error) {
	schema, err := ConvertToolSchema(tool.Schema)
	if err != nil {
		return nil, fmt.Errorf("converting schema: %w", err)
	}

	return &genai.FunctionDeclaration{
		Name:        tool.Name,
		Description: tool.Description,
		Parameters:  schema,
	}, nil
}

// ConvertToolSchema converts gai.ToolSchema to genai.Schema.
func ConvertToolSchema(schema gai.ToolSchema) (*genai.Schema, error) {
	if schema.Properties == nil {
		return &genai.Schema{
			Type:       genai.TypeObject,
			Properties: map[string]*genai.Schema{},
		}, nil
	}

	props, ok := schema.Properties.(map[string]any)
	if !ok {
		jsonData, err := json.Marshal(schema.Properties)
		if err != nil {
			return nil, fmt.Errorf("marshaling properties: %w", err)
		}
		if err := json.Unmarshal(jsonData, &props); err != nil {
			return nil, fmt.Errorf("unmarshaling properties: %w", err)
		}
	}

	genaiProps := make(map[string]*genai.Schema)
	var required []string

	// Check if properties are wrapped in a "properties" key (JSON Schema format)
	if propsMap, ok := props["properties"].(map[string]any); ok {
		// Standard JSON Schema format
		for name, prop := range propsMap {
			propSchema, err := ConvertProperty(prop)
			if err != nil {
				return nil, fmt.Errorf("converting property %s: %w", name, err)
			}
			genaiProps[name] = propSchema
		}
		
		if reqList, ok := props["required"].([]any); ok {
			for _, req := range reqList {
				if reqStr, ok := req.(string); ok {
					required = append(required, reqStr)
				}
			}
		}
	} else {
		// Direct properties format (as used by gai tools)
		for name, prop := range props {
			propSchema, err := ConvertProperty(prop)
			if err != nil {
				return nil, fmt.Errorf("converting property %s: %w", name, err)
			}
			genaiProps[name] = propSchema
		}
	}

	return &genai.Schema{
		Type:       genai.TypeObject,
		Properties: genaiProps,
		Required:   required,
	}, nil
}

// ConvertProperty converts a property definition to genai.Schema.
func ConvertProperty(prop any) (*genai.Schema, error) {
	propMap, ok := prop.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("property is not a map")
	}

	schema := &genai.Schema{}

	if typeStr, ok := propMap["type"].(string); ok {
		switch typeStr {
		case "string":
			schema.Type = genai.TypeString
		case "number":
			schema.Type = genai.TypeNumber
		case "integer":
			schema.Type = genai.TypeInteger
		case "boolean":
			schema.Type = genai.TypeBoolean
		case "array":
			schema.Type = genai.TypeArray
			if items, ok := propMap["items"].(map[string]any); ok {
				itemSchema, err := ConvertProperty(items)
				if err != nil {
					return nil, fmt.Errorf("converting array items: %w", err)
				}
				schema.Items = itemSchema
			}
		case "object":
			schema.Type = genai.TypeObject
			if props, ok := propMap["properties"].(map[string]any); ok {
				schema.Properties = make(map[string]*genai.Schema)
				for name, subProp := range props {
					subSchema, err := ConvertProperty(subProp)
					if err != nil {
						return nil, fmt.Errorf("converting object property %s: %w", name, err)
					}
					schema.Properties[name] = subSchema
				}
			}
		default:
			schema.Type = genai.TypeString
		}
	}

	if desc, ok := propMap["description"].(string); ok {
		schema.Description = desc
	}

	return schema, nil
}