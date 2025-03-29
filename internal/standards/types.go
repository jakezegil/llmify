package standards

// LLMRule defines a standard enforced by the LLM.
type LLMRule struct {
	ID          string   `mapstructure:"id"`
	Description string   `mapstructure:"description"`
	Prompt      string   `mapstructure:"prompt"`
	Language    string   `mapstructure:"language,omitempty"`   // If empty, applies to all langs? Or error? Define behavior.
	AppliesTo   []string `mapstructure:"applies_to,omitempty"` // Glob patterns relative to repo root
}

// LanguageStandards holds settings for a specific language.
type LanguageStandards struct {
	Formatter string    `mapstructure:"formatter"` // e.g., "prettier", "black", "gofmt", "auto", or specific command
	Linter    string    `mapstructure:"linter"`    // e.g., "eslint", "ruff", "golangci-lint", "auto"
	LLMRules  []LLMRule `mapstructure:"llm_rules"`
	// LintFixOnEnforce is defined globally, but could be overridden here if needed.
}

// StandardsConfig represents the structure of the .llmify_standards.yaml file.
type StandardsConfig struct {
	Version          int                          `mapstructure:"version"`
	FormatOnEnforce  bool                         `mapstructure:"format_on_enforce"`
	LintOnEnforce    bool                         `mapstructure:"lint_on_enforce"`
	LintFixOnEnforce bool                         `mapstructure:"lint_fix_on_enforce"`
	Languages        map[string]LanguageStandards `mapstructure:"languages"`
	LLMRulesGeneral  []LLMRule                    `mapstructure:"llm_rules_general,omitempty"` // Rules applied regardless of specific language block
	// Could add global tool paths or configurations here
}
