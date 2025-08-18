package services

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestOpenAIClient_CleanJSONResponse tests the JSON cleaning functionality
func TestOpenAIClient_CleanJSONResponse(t *testing.T) {
	client := &OpenAIClient{}
	
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Clean JSON",
			input:    `{"activities": []}`,
			expected: `{"activities": []}`,
		},
		{
			name:     "JSON with markdown code blocks",
			input:    "```json\n{\"activities\": []}\n```",
			expected: `{"activities": []}`,
		},
		{
			name:     "JSON with just backticks",
			input:    "```\n{\"activities\": []}\n```",
			expected: `{"activities": []}`,
		},
		{
			name:     "JSON with extra whitespace",
			input:    "  \n  {\"activities\": []}  \n  ",
			expected: `{"activities": []}`,
		},
		{
			name:     "Plain text response (problematic case)",
			input:    "I'm unable to extract structured data from the provided content.",
			expected: "I'm unable to extract structured data from the provided content.",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := client.cleanJSONResponse(tc.input)
			if result != tc.expected {
				t.Errorf("Expected: %q, got: %q", tc.expected, result)
			}
		})
	}
}

// TestOpenAIClient_JSONParsingResilience tests how the client handles various response formats
func TestOpenAIClient_JSONParsingResilience(t *testing.T) {
	testCases := []struct {
		name            string
		response        string
		expectError     bool
		errorContains   string
	}{
		{
			name:        "Valid empty JSON",
			response:    `{"activities": []}`,
			expectError: false,
		},
		{
			name:        "Valid JSON with activities",
			response:    `{"activities": [{"title": "Test", "type": "class"}]}`,
			expectError: false,
		},
		{
			name:          "Plain text response",
			response:      "I'm unable to extract structured data.",
			expectError:   true,
			errorContains: "invalid character",
		},
		{
			name:          "Invalid JSON",
			response:      `{"activities": [}`,
			expectError:   true,
			errorContains: "invalid character",
		},
		{
			name:          "Empty response",
			response:      "",
			expectError:   true,
			errorContains: "unexpected end of JSON",
		},
		{
			name:        "JSON with markdown blocks",
			response:    "```json\n{\"activities\": []}\n```",
			expectError: false,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := &OpenAIClient{}
			cleanedContent := client.cleanJSONResponse(tc.response)
			
			// Try to parse the cleaned JSON
			var activitiesData struct {
				Activities []interface{} `json:"activities"`
			}
			err := json.Unmarshal([]byte(cleanedContent), &activitiesData)
			
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tc.errorContains != "" && !strings.Contains(err.Error(), tc.errorContains) {
					t.Errorf("Expected error to contain %q, but got: %v", tc.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestOpenAIClient_PromptConstruction tests that prompts are built correctly
func TestOpenAIClient_BuildUserPrompt(t *testing.T) {
	client := &OpenAIClient{}
	
	content := "Test content"
	sourceURL := "https://example.com"
	
	prompt := client.buildUserPrompt(content, sourceURL)
	
	// Check that the prompt contains our JSON enforcement
	if !strings.Contains(prompt, "CRITICAL: You MUST respond with valid JSON only") {
		t.Error("Prompt should contain JSON enforcement language")
	}
	
	if !strings.Contains(prompt, `{"activities": []}`) {
		t.Error("Prompt should contain fallback JSON format")
	}
	
	if !strings.Contains(prompt, "valid JSON that can be parsed programmatically") {
		t.Error("Prompt should emphasize programmatic parsing")
	}
	
	if !strings.Contains(prompt, content) {
		t.Error("Prompt should contain the provided content")
	}
	
	if !strings.Contains(prompt, sourceURL) {
		t.Error("Prompt should contain the source URL")
	}
}

// TestOpenAIClient_ValidateJSONResponse tests the JSON response validation
func TestOpenAIClient_ValidateJSONResponse(t *testing.T) {
	client := &OpenAIClient{}
	
	testCases := []struct {
		name        string
		response    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid JSON response",
			response:    `{"activities": []}`,
			expectError: false,
		},
		{
			name:        "Valid JSON with activities",
			response:    `{"activities": [{"title": "Test"}]}`,
			expectError: false,
		},
		{
			name:        "Plain text response (the bug)",
			response:    "I'm unable to extract structured data from the provided content.",
			expectError: true,
			errorMsg:    "plain text instead of JSON",
		},
		{
			name:        "Another plain text pattern",
			response:    "Sorry, I cannot process this content.",
			expectError: true,
			errorMsg:    "plain text instead of JSON",
		},
		{
			name:        "Empty response",
			response:    "",
			expectError: true,
			errorMsg:    "empty response",
		},
		{
			name:        "Missing opening brace",
			response:    `"activities": []}`,
			expectError: true,
			errorMsg:    "does not start with '{'",
		},
		{
			name:        "Missing closing brace",
			response:    `{"activities": []`,
			expectError: true,
			errorMsg:    "does not end with '}'",
		},
		{
			name:        "Missing activities key",
			response:    `{"events": []}`,
			expectError: true,
			errorMsg:    "does not contain required 'activities' key",
		},
		{
			name:        "Response with whitespace",
			response:    `  {"activities": []}  `,
			expectError: false,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := client.validateJSONResponse(tc.response)
			
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tc.errorMsg != "" && !strings.Contains(err.Error(), tc.errorMsg) {
					t.Errorf("Expected error to contain %q, but got: %v", tc.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestOpenAIClient_ConfigurationUnit tests client configuration
func TestOpenAIClient_ConfigurationUnit(t *testing.T) {
	// Test default configuration
	t.Setenv("OPENAI_API_KEY", "test-key")
	
	client := NewOpenAIClient()
	
	if client.model != "gpt-4o-mini" {
		t.Errorf("Expected model gpt-4o-mini, got %s", client.model)
	}
	
	if client.temperature != 0.1 {
		t.Errorf("Expected temperature 0.1, got %f", client.temperature)
	}
	
	if client.maxTokens != 8000 {
		t.Errorf("Expected maxTokens 8000, got %d", client.maxTokens)
	}
	
	// Test custom configuration
	customClient := NewOpenAIClientWithConfig("gpt-4", 0.2, 3000)
	
	if customClient.model != "gpt-4" {
		t.Errorf("Expected custom model gpt-4, got %s", customClient.model)
	}
	
	if customClient.temperature != 0.2 {
		t.Errorf("Expected custom temperature 0.2, got %f", customClient.temperature)
	}
	
	if customClient.maxTokens != 3000 {
		t.Errorf("Expected custom maxTokens 3000, got %d", customClient.maxTokens)
	}
}