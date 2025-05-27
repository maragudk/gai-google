package schema_test

import (
	"os"
	"testing"

	"google.golang.org/genai"
	"maragu.dev/gai"
	"maragu.dev/gai/tools"
	"maragu.dev/is"

	"maragu.dev/gai-google/internal/schema"
)

func TestConvertToolToFunction(t *testing.T) {
	t.Run("converts ReadFile tool", func(t *testing.T) {
		root, err := os.OpenRoot("../../testdata")
		is.NotError(t, err)

		tool := tools.NewReadFile(root)
		funcDecl, err := schema.ConvertToolToFunction(tool)
		is.NotError(t, err)

		is.Equal(t, "read_file", funcDecl.Name)
		is.Equal(t, "Read the contents of a given relative file path. Use this when you want to see what's inside a file. Do not use this with directory names.", funcDecl.Description)

		// Check parameters
		is.NotError(t, err)
		is.Equal(t, genai.TypeObject, funcDecl.Parameters.Type)
		is.Equal(t, 1, len(funcDecl.Parameters.Properties))

		pathProp, ok := funcDecl.Parameters.Properties["path"]
		is.True(t, ok, "expected path property")
		is.Equal(t, genai.TypeString, pathProp.Type)
		is.Equal(t, "The relative path of a file in the working directory.", pathProp.Description)
	})

	t.Run("converts ListDir tool", func(t *testing.T) {
		root, err := os.OpenRoot("../../testdata")
		is.NotError(t, err)

		tool := tools.NewListDir(root)
		funcDecl, err := schema.ConvertToolToFunction(tool)
		is.NotError(t, err)

		is.Equal(t, "list_dir", funcDecl.Name)
		is.Equal(t, "List files and directories at a given path recursively. If no path is provided, lists files and directories in the current directory.", funcDecl.Description)

		// ListDir has a path parameter
		is.Equal(t, genai.TypeObject, funcDecl.Parameters.Type)
		is.Equal(t, 1, len(funcDecl.Parameters.Properties))

		pathProp, ok := funcDecl.Parameters.Properties["path"]
		is.True(t, ok, "expected path property")
		is.Equal(t, genai.TypeString, pathProp.Type)
	})
}

func TestConvertTools(t *testing.T) {
	t.Run("converts multiple tools", func(t *testing.T) {
		root, err := os.OpenRoot("../../testdata")
		is.NotError(t, err)

		gaiTools := []gai.Tool{
			tools.NewReadFile(root),
			tools.NewListDir(root),
		}

		genaiTools, err := schema.ConvertTools(gaiTools)
		is.NotError(t, err)

		is.Equal(t, 1, len(genaiTools))
		is.Equal(t, 2, len(genaiTools[0].FunctionDeclarations))

		is.Equal(t, "read_file", genaiTools[0].FunctionDeclarations[0].Name)
		is.Equal(t, "list_dir", genaiTools[0].FunctionDeclarations[1].Name)
	})
}

func TestConvertToolSchema(t *testing.T) {
	t.Run("converts empty schema", func(t *testing.T) {
		testSchema := gai.ToolSchema{}

		genaiSchema, err := schema.ConvertToolSchema(testSchema)
		is.NotError(t, err)

		is.Equal(t, genai.TypeObject, genaiSchema.Type)
		is.Equal(t, 0, len(genaiSchema.Properties))
	})

	t.Run("converts simple properties", func(t *testing.T) {
		toolSchema := gai.ToolSchema{
			Properties: map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "The name",
				},
				"age": map[string]any{
					"type":        "integer",
					"description": "The age",
				},
			},
		}

		genaiSchema, err := schema.ConvertToolSchema(toolSchema)
		is.NotError(t, err)

		is.Equal(t, genai.TypeObject, genaiSchema.Type)
		is.Equal(t, 2, len(genaiSchema.Properties))

		nameProp := genaiSchema.Properties["name"]
		is.Equal(t, genai.TypeString, nameProp.Type)
		is.Equal(t, "The name", nameProp.Description)

		ageProp := genaiSchema.Properties["age"]
		is.Equal(t, genai.TypeInteger, ageProp.Type)
		is.Equal(t, "The age", ageProp.Description)
	})

	t.Run("converts JSON Schema format with properties wrapper", func(t *testing.T) {
		toolSchema := gai.ToolSchema{
			Properties: map[string]any{
				"properties": map[string]any{
					"file": map[string]any{
						"type":        "string",
						"description": "File path",
					},
				},
				"required": []any{"file"},
			},
		}

		genaiSchema, err := schema.ConvertToolSchema(toolSchema)
		is.NotError(t, err)

		is.Equal(t, genai.TypeObject, genaiSchema.Type)
		is.Equal(t, 1, len(genaiSchema.Properties))
		is.Equal(t, 1, len(genaiSchema.Required))
		is.Equal(t, "file", genaiSchema.Required[0])

		fileProp := genaiSchema.Properties["file"]
		is.Equal(t, genai.TypeString, fileProp.Type)
		is.Equal(t, "File path", fileProp.Description)
	})

	t.Run("converts array type", func(t *testing.T) {
		toolSchema := gai.ToolSchema{
			Properties: map[string]any{
				"tags": map[string]any{
					"type":        "array",
					"description": "List of tags",
					"items": map[string]any{
						"type": "string",
					},
				},
			},
		}

		genaiSchema, err := schema.ConvertToolSchema(toolSchema)
		is.NotError(t, err)

		tagsProp := genaiSchema.Properties["tags"]
		is.Equal(t, genai.TypeArray, tagsProp.Type)
		is.Equal(t, "List of tags", tagsProp.Description)
		is.Equal(t, genai.TypeString, tagsProp.Items.Type)
	})

	t.Run("converts nested object type", func(t *testing.T) {
		toolSchema := gai.ToolSchema{
			Properties: map[string]any{
				"person": map[string]any{
					"type":        "object",
					"description": "Person details",
					"properties": map[string]any{
						"name": map[string]any{
							"type": "string",
						},
						"age": map[string]any{
							"type": "integer",
						},
					},
				},
			},
		}

		genaiSchema, err := schema.ConvertToolSchema(toolSchema)
		is.NotError(t, err)

		personProp := genaiSchema.Properties["person"]
		is.Equal(t, genai.TypeObject, personProp.Type)
		is.Equal(t, "Person details", personProp.Description)
		is.Equal(t, 2, len(personProp.Properties))

		is.Equal(t, genai.TypeString, personProp.Properties["name"].Type)
		is.Equal(t, genai.TypeInteger, personProp.Properties["age"].Type)
	})

	t.Run("converts all basic types", func(t *testing.T) {
		toolSchema := gai.ToolSchema{
			Properties: map[string]any{
				"text":    map[string]any{"type": "string"},
				"number":  map[string]any{"type": "number"},
				"integer": map[string]any{"type": "integer"},
				"boolean": map[string]any{"type": "boolean"},
				"unknown": map[string]any{"type": "custom"}, // Should default to string
			},
		}

		genaiSchema, err := schema.ConvertToolSchema(toolSchema)
		is.NotError(t, err)

		is.Equal(t, genai.TypeString, genaiSchema.Properties["text"].Type)
		is.Equal(t, genai.TypeNumber, genaiSchema.Properties["number"].Type)
		is.Equal(t, genai.TypeInteger, genaiSchema.Properties["integer"].Type)
		is.Equal(t, genai.TypeBoolean, genaiSchema.Properties["boolean"].Type)
		is.Equal(t, genai.TypeString, genaiSchema.Properties["unknown"].Type)
	})
}
