package main

// ModelInfo represents information about an LLM model
type ModelInfo struct {
	ID          string // Model identifier used in API calls
	Name        string // Human-readable name
	Description string // Brief description of the model
	MaxTokens   int    // Default max tokens for this model
	IsDefault   bool   // Whether this is the default model for the provider
}

// ProviderModels maps each provider to its available models
var ProviderModels = map[LLMProvider][]ModelInfo{
	ProviderAnthropic: {
		{
			ID:          "claude-3-5-haiku-latest",
			Name:        "Claude 3.5 Haiku",
			Description: "Fast and cost-effective model for simple tasks",
			MaxTokens:   4000,
			IsDefault:   true,
		},
		{
			ID:          "claude-3-5-sonnet-latest",
			Name:        "Claude 3.5 Sonnet",
			Description: "Balanced performance for complex reasoning",
			MaxTokens:   16000,
			IsDefault:   false,
		},
		{
			ID:          "claude-3-7-sonnet-latest",
			Name:        "Claude 3.7 Sonnet",
			Description: "Advanced reasoning with detailed outputs",
			MaxTokens:   16000,
			IsDefault:   false,
		},
		{
			ID:          "claude-3-opus-latest",
			Name:        "Claude 3 Opus",
			Description: "Most capable model for complex analysis",
			MaxTokens:   32000,
			IsDefault:   false,
		},
	},
	ProviderOpenAI: {
		{
			ID:          "gpt-4o",
			Name:        "GPT-4o",
			Description: "Latest GPT-4 model with optimal performance",
			MaxTokens:   4000,
			IsDefault:   true,
		},
		{
			ID:          "gpt-4-turbo",
			Name:        "GPT-4 Turbo",
			Description: "Improved GPT-4 with better performance",
			MaxTokens:   4000,
			IsDefault:   false,
		},
		{
			ID:          "gpt-3.5-turbo",
			Name:        "GPT-3.5 Turbo",
			Description: "Fast and cost-effective model",
			MaxTokens:   4000, 
			IsDefault:   false,
		},
	},
	ProviderGemini: {
		{
			ID:          "gemini-2.5-pro-preview-03-25",
			Name:        "Gemini 2.5 Pro Preview",
			Description: "Enhanced thinking and reasoning, multimodal understanding, advanced coding",
			MaxTokens:   32000,
			IsDefault:   true,
		},
		{
			ID:          "gemini-2.5-flash-preview-04-17",
			Name:        "Gemini 2.5 Flash Preview",
			Description: "Adaptive thinking, cost efficiency for multimodal tasks",
			MaxTokens:   16000,
			IsDefault:   false,
		},
		{
			ID:          "gemini-2.0-flash",
			Name:        "Gemini 2.0 Flash",
			Description: "Speed, thinking, realtime streaming, and multimodal generation",
			MaxTokens:   8000,
			IsDefault:   false,
		},
	},
}

// GetDefaultModel returns the default model for a provider
func GetDefaultModel(provider LLMProvider) string {
	models, exists := ProviderModels[provider]
	if !exists {
		return ""
	}
	
	for _, model := range models {
		if model.IsDefault {
			return model.ID
		}
	}
	
	// Fallback to first model if no default is marked
	if len(models) > 0 {
		return models[0].ID
	}
	
	return ""
}

// GetModelInfo returns information about a specific model
func GetModelInfo(provider LLMProvider, modelID string) (ModelInfo, bool) {
	models, exists := ProviderModels[provider]
	if !exists {
		return ModelInfo{}, false
	}
	
	for _, model := range models {
		if model.ID == modelID {
			return model, true
		}
	}
	
	return ModelInfo{}, false
}

// GetAvailableModels returns a list of all available models for a provider
func GetAvailableModels(provider LLMProvider) []ModelInfo {
	return ProviderModels[provider]
}