package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type LLMConfig struct {
	Provider string `mapstructure:"provider"`
	Model    string `mapstructure:"model"`
	// Add provider-specific fields if needed, e.g.:
	OllamaBaseURL string `mapstructure:"ollama_base_url"`
	// API keys are typically handled via environment variables
}

type CommitConfig struct {
	Model string `mapstructure:"model"` // Optional override
}

type DocsConfig struct {
	Model string `mapstructure:"model"` // Optional override
	// Could add patterns for doc files here:
	// Patterns []string `mapstructure:"patterns"`
}

type Config struct {
	LLM    LLMConfig    `mapstructure:"llm"`
	Commit CommitConfig `mapstructure:"commit"`
	Docs   DocsConfig   `mapstructure:"docs"`
}

var GlobalConfig Config

func LoadConfig() error {
	v := viper.New()

	// 1. Set Defaults
	v.SetDefault("llm.provider", "openai")
	v.SetDefault("llm.model", "gpt-4") // Adjust default model as needed
	v.SetDefault("llm.ollama_base_url", "http://localhost:11434")
	// Defaults for Commit and Docs models will inherit from llm.model if not set

	// 2. Set config file paths
	home, _ := os.UserHomeDir()
	configName := "config"
	configType := "yaml"
	configPaths := []string{
		".", // Project root .llmifyrc.yaml (or .llmifyrc)
	}
	if home != "" {
		configPaths = append(configPaths, filepath.Join(home, ".config", "llmify")) // ~/.config/llmify/config.yaml
	}

	v.SetConfigName(configName) // Name of config file (without extension)
	v.SetConfigType(configType)
	for _, p := range configPaths {
		v.AddConfigPath(p)
	}
	v.SetConfigName(".llmifyrc") // Also support .llmifyrc.yaml in project root
	v.AddConfigPath(".")

	// 3. Read config file (optional)
	err := v.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Config file was found but another error was produced
			return fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found; ignore error if it's just not found
		fmt.Fprintln(os.Stderr, "Info: No config file found, using defaults and environment variables.")
	}

	// 4. Set environment variable binding
	v.SetEnvPrefix("LLMIFY") // e.g., LLMIFY_LLM_PROVIDER
	v.AutomaticEnv()
	// Allow specific API keys to be picked up directly
	v.BindEnv("llm.api_key.openai", "OPENAI_API_KEY")
	v.BindEnv("llm.api_key.anthropic", "ANTHROPIC_API_KEY")
	// Add others as needed

	// 5. Unmarshal into GlobalConfig
	err = v.Unmarshal(&GlobalConfig)
	if err != nil {
		return fmt.Errorf("unable to decode config: %w", err)
	}

	// Apply overrides if specific models aren't set
	if GlobalConfig.Commit.Model == "" {
		GlobalConfig.Commit.Model = GlobalConfig.LLM.Model
	}
	if GlobalConfig.Docs.Model == "" {
		GlobalConfig.Docs.Model = GlobalConfig.LLM.Model
	}

	// For API Keys, prefer specific env vars if not set via LLMIFY_LLM_API_KEY_PROVIDER
	// This logic might be better placed within the client factory, but shown here for clarity
	if viper.GetString("llm.api_key.openai") != "" {
		// Store it somewhere accessible if needed, but often the SDK reads it directly
	}

	if viper.GetBool("verbose") { // Assuming verbose flag sets this globally via viper
		fmt.Fprintf(os.Stderr, "Loaded Config: %+v\n", GlobalConfig)
	}

	return nil
}

// Helper to get API key for the current provider
func GetAPIKey(provider string) string {
	// Viper reads bound env vars automatically
	key := viper.GetString(fmt.Sprintf("llm.api_key.%s", strings.ToLower(provider)))
	if key == "" {
		// Fallback to standard env vars if Viper binding didn't pick it up
		switch strings.ToLower(provider) {
		case "openai":
			key = os.Getenv("OPENAI_API_KEY")
		case "anthropic":
			key = os.Getenv("ANTHROPIC_API_KEY")
			// Add other cases
		}
	}
	return key
}
