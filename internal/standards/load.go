package standards

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gobwas/glob"              // For glob pattern matching
	"github.com/jake/llmify/internal/git" // Assuming git package is available
	"github.com/spf13/viper"
)

const DefaultStandardsFilename = ".llmify_standards.yaml"

// LoadStandards loads the standards configuration file.
// It searches for the file in the current directory and ancestors up to the repo root.
func LoadStandards(configPath string) (*StandardsConfig, string, error) { // Returns config, path found, error
	v := viper.New()
	v.SetConfigType("yaml")

	var foundConfigPath string

	if configPath != "" {
		// User specified a path
		v.SetConfigFile(configPath)
		foundConfigPath = configPath
	} else {
		// Search for default file
		v.SetConfigName(DefaultStandardsFilename) // Register config file name (no extension needed here)
		v.SetConfigType("yaml")

		// Start search from CWD
		cwd, err := os.Getwd()
		if err != nil {
			return nil, "", fmt.Errorf("failed to get current working directory: %w", err)
		}

		// Find repo root to stop search
		repoRoot, err := git.GetRepoRoot()
		if err != nil {
			log.Printf("Warning: Could not find repo root, standards search limited to current dir: %v", err)
			repoRoot = cwd // Fallback
		}
		absRepoRoot, _ := filepath.Abs(repoRoot)

		// Search upwards
		currentDir := cwd
		for {
			v.AddConfigPath(currentDir) // Add current dir to search paths

			// Check if file exists directly (Viper sometimes needs help finding it without extension)
			potentialPath := filepath.Join(currentDir, DefaultStandardsFilename)
			if _, statErr := os.Stat(potentialPath); statErr == nil {
				// Found it, tell Viper to use this specific file
				v.SetConfigFile(potentialPath)
				foundConfigPath = potentialPath
				log.Printf("Found standards config at: %s", foundConfigPath)
				break
			}

			// Stop if we reached repo root or filesystem root
			parent := filepath.Dir(currentDir)
			absCurrent, _ := filepath.Abs(currentDir)
			if parent == currentDir || absCurrent == absRepoRoot || parent == "" || parent == "/" {
				break // Reached top
			}
			currentDir = parent
		}
	}

	var config StandardsConfig

	if foundConfigPath == "" {
		return nil, "", fmt.Errorf("standards configuration file not found (searched for %s)", DefaultStandardsFilename)
	}

	// Read the config file explicitly found or specified
	if err := v.ReadInConfig(); err != nil {
		return nil, foundConfigPath, fmt.Errorf("failed to read standards config file '%s': %w", foundConfigPath, err)
	}

	// Unmarshal the config
	if err := v.Unmarshal(&config); err != nil {
		return nil, foundConfigPath, fmt.Errorf("failed to unmarshal standards config from '%s': %w", foundConfigPath, err)
	}

	// Basic validation
	if config.Version != 1 {
		log.Printf("Warning: Unsupported standards config version '%d'. Expected version 1.", config.Version)
		// Potentially return error depending on compatibility policy
	}
	if config.Languages == nil {
		config.Languages = make(map[string]LanguageStandards) // Initialize map if empty
	}

	log.Printf("Successfully loaded standards config version %d from %s", config.Version, foundConfigPath)
	return &config, foundConfigPath, nil
}

// GetApplicableRules returns the LLM rules relevant for a given file path and language.
// filePath should be relative to the repository root for glob matching.
func GetApplicableRules(cfg *StandardsConfig, filePathRel string, lang string, ruleIDs []string) ([]LLMRule, error) {
	applicable := []LLMRule{}
	ruleIDFilter := make(map[string]struct{})
	useIDFilter := len(ruleIDs) > 0
	if useIDFilter {
		for _, id := range ruleIDs {
			ruleIDFilter[id] = struct{}{}
		}
	}

	// 1. Check General Rules
	for _, rule := range cfg.LLMRulesGeneral {
		if useIDFilter {
			if _, ok := ruleIDFilter[rule.ID]; !ok {
				continue // Skip if not in the specific list requested
			}
		}
		if len(rule.Language) > 0 && rule.Language != lang {
			continue // Skip if language doesn't match (should general rules have lang?)
		}
		if checkAppliesTo(rule.AppliesTo, filePathRel) {
			applicable = append(applicable, rule)
		}
	}

	// 2. Check Language-Specific Rules
	if langSettings, ok := cfg.Languages[lang]; ok {
		for _, rule := range langSettings.LLMRules {
			// Check rule ID filter first
			if useIDFilter {
				if _, ok := ruleIDFilter[rule.ID]; !ok {
					continue
				}
			}
			// Check applies_to patterns
			if checkAppliesTo(rule.AppliesTo, filePathRel) {
				// Add rule if it passes filter and applies_to checks
				// Avoid duplicates if a rule ID is somehow listed twice (unlikely with good config)
				alreadyAdded := false
				for _, added := range applicable {
					if added.ID == rule.ID {
						alreadyAdded = true
						break
					}
				}
				if !alreadyAdded {
					applicable = append(applicable, rule)
				}
			}
		}
	}

	return applicable, nil
}

// checkAppliesTo checks if a relative file path matches any of the glob patterns.
// If patterns list is empty, it's considered a match.
func checkAppliesTo(patterns []string, filePathRel string) bool {
	if len(patterns) == 0 {
		return true // No patterns means applies to all files of the language
	}
	// Ensure forward slashes for glob matching
	matchPath := filepath.ToSlash(filePathRel)
	for _, pattern := range patterns {
		g, err := glob.Compile(pattern)
		if err != nil {
			log.Printf("Warning: Invalid glob pattern '%s' in standards config: %v", pattern, err)
			continue // Skip invalid patterns
		}
		if g.Match(matchPath) {
			return true // Match found
		}
	}
	return false // No patterns matched
}
