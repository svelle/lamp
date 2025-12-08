package main

// ModelInfo represents information about an LLM model
type ModelInfo struct {
	ID          string // Model identifier used in API calls
	Name        string // Human-readable name
	Description string // Brief description of the model
	IsDefault   bool   // Whether this is the default model for the provider
}

// ProviderModels maps each provider to its available models
var ProviderModels = map[LLMProvider][]ModelInfo{
	ProviderAnthropic: {
		{
			ID:          "claude-sonnet-4-5-20250929",
			Name:        "Claude Sonnet 4.5",
			Description: "Latest Sonnet with improved coding and reasoning",
			IsDefault:   true,
		},
		{
			ID:          "claude-opus-4-5-20251101",
			Name:        "Claude Opus 4.5",
			Description: "Most intelligent model for coding, agents, and complex tasks",
			IsDefault:   false,
		},
		{
			ID:          "claude-haiku-4-5-20251001",
			Name:        "Claude Haiku 4.5",
			Description: "Fast and cost-effective for real-time assistants",
			IsDefault:   false,
		},
		{
			ID:          "claude-sonnet-4-20250514",
			Name:        "Claude Sonnet 4",
			Description: "Balanced performance for general tasks",
			IsDefault:   false,
		},
		{
			ID:          "claude-opus-4-1-20250805",
			Name:        "Claude Opus 4.1",
			Description: "Optimized for agentic tasks and real-world coding",
			IsDefault:   false,
		},
		{
			ID:          "claude-3-5-haiku-20241022",
			Name:        "Claude 3.5 Haiku",
			Description: "Fast and cost-effective model for simple tasks",
			IsDefault:   false,
		},
	},
	ProviderOpenAI: {
		{
			ID:          "gpt-4.1",
			Name:        "GPT-4.1",
			Description: "Latest GPT model with improved coding and instruction following",
			IsDefault:   true,
		},
		{
			ID:          "gpt-4.1-mini",
			Name:        "GPT-4.1 Mini",
			Description: "Smaller, faster version of GPT-4.1",
			IsDefault:   false,
		},
		{
			ID:          "gpt-4.1-nano",
			Name:        "GPT-4.1 Nano",
			Description: "Fastest and most affordable GPT-4.1 variant",
			IsDefault:   false,
		},
		{
			ID:          "o4-mini",
			Name:        "o4-mini",
			Description: "Reasoning model for multi-step tasks and coding",
			IsDefault:   false,
		},
		{
			ID:          "o3",
			Name:        "o3",
			Description: "Advanced reasoning model for complex problems",
			IsDefault:   false,
		},
	},
	ProviderGemini: {
		{
			ID:          "gemini-2.5-pro",
			Name:        "Gemini 2.5 Pro",
			Description: "Most powerful model for complex reasoning and coding",
			IsDefault:   true,
		},
		{
			ID:          "gemini-2.5-flash",
			Name:        "Gemini 2.5 Flash",
			Description: "Best price-performance for high-throughput tasks",
			IsDefault:   false,
		},
		{
			ID:          "gemini-2.5-flash-lite",
			Name:        "Gemini 2.5 Flash-Lite",
			Description: "Ultra-efficient for massive scale processing",
			IsDefault:   false,
		},
		{
			ID:          "gemini-3-pro-preview",
			Name:        "Gemini 3 Pro Preview",
			Description: "Latest reasoning model for agentic workflows",
			IsDefault:   false,
		},
		{
			ID:          "gemini-2.0-flash",
			Name:        "Gemini 2.0 Flash",
			Description: "Cost-effective for general-purpose tasks",
			IsDefault:   false,
		},
	},
	// For Ollama, these are just common examples - users can specify any model they have installed locally
	ProviderOllama: {
		{
			ID:          "llama3",
			Name:        "Llama 3",
			Description: "Example: Meta's Llama 3 model (use the name of any model you have installed)",
			IsDefault:   true,
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
