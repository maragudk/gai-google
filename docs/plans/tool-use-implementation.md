# Tool Use Implementation Plan for gai-google

## Overview
This document outlines the plan for implementing tool use (function calling) in the gai-google library. The Google Gemini SDK calls this feature "function calling" while the gai interface calls it "tool use".

## Current State
- The `ChatCompleter` in `chat_complete.go` currently only handles text messages
- Tests for tool use are already written in `chat_complete_test.go` but failing
- The gai interface expects tools to be part of `ChatCompleteRequest` and tool calls to be returned as `MessagePartTypeToolCall`

## Implementation Requirements

### 1. Tool/Function Declaration Conversion
Convert `gai.Tool` to `genai.FunctionDeclaration`:
- Extract tool name, description, and parameter schema from `gai.Tool`
- Build `genai.Schema` from the tool's parameter definition
- Create `genai.FunctionDeclaration` with the converted information
- Group function declarations into `genai.Tool` objects

### 2. Message Handling Updates
Handle different message types in the request:
- `gai.MessageRoleUser` with tool results (`MessagePartTypeToolResult`)
- `gai.MessageRoleModel` with tool calls (`MessagePartTypeToolCall`)

### 3. Response Processing
Process model responses that contain function calls:
- Check for `FunctionCall` parts in the response
- Convert function calls to `gai.MessagePartTypeToolCall`
- Include tool call ID, name, and arguments

### 4. Configuration Updates
Update `GenerateContentConfig` to include:
- Tools configuration
- Tool calling mode settings

## Detailed Implementation Steps

### Step 1: Add Tool Configuration to ChatComplete
```go
// In ChatComplete method
if len(req.Tools) > 0 {
    // Convert gai.Tools to genai.Tools
    var tools []*genai.Tool
    // ... conversion logic
    config.Tools = tools
    config.ToolConfig = &genai.ToolConfig{
        FunctionCallingConfig: &genai.FunctionCallingConfig{
            Mode: genai.FunctionCallingConfigModeAny,
        },
    }
}
```

### Step 2: Update Message Part Handling
Add cases for tool-related message parts:
```go
case gai.MessagePartTypeToolCall:
    // Convert to genai.FunctionCall
case gai.MessagePartTypeToolResult:
    // Convert to genai.FunctionResponse
```

### Step 3: Update Response Stream Processing
Handle function calls in the response:
```go
// In the response processing loop
for _, part := range chunk.Candidates[0].Content.Parts {
    switch p := part.(type) {
    case *genai.Part:
        // Existing text handling
    case *genai.FunctionCall:
        // New function call handling
    }
}
```

## Key Considerations

### Tool Schema Conversion
The gai interface uses a generic parameter structure that needs to be converted to genai's Schema format:
- Map JSON schema types to genai.Type constants
- Handle required fields
- Preserve parameter descriptions

### Tool Call ID Management
- Google SDK uses function call names for identification
- Need to generate or manage IDs for tool calls to match gai interface expectations

### Error Handling
- Handle cases where tool schema is invalid
- Gracefully handle function calling errors
- Ensure proper error propagation through the streaming interface

## Testing Approach
1. Verify existing tests pass after implementation
2. Test edge cases:
   - Tools with no parameters
   - Multiple tool calls in one response
   - Tool call errors
   - Invalid tool schemas

## References
- [Google Gemini Function Calling Example](https://github.com/google-gemini/api-examples/blob/main/go/function_calling.go)
- gai interface documentation
- Existing test cases in `chat_complete_test.go`