package walker

import (
	"io/fs"
	"log"
	"path/filepath"
	"strings"

	"github.com/jake/llmify/internal/language"
	"github.com/jake/llmify/internal/util"
	gitignore "github.com/sabhiram/go-gitignore"
	"github.com/spf13/viper"
)

// WalkCallback is the function signature for the callback used by WalkProjectFiles.
type WalkCallback func(repoRoot, filePathRel string, lang string, d fs.DirEntry) error

// WalkProjectFiles walks the directory structure, detects language, checks ignores,
// and calls the callback for relevant text files.
func WalkProjectFiles(repoRoot string, absStartPath string, ignorer *gitignore.GitIgnore, callback WalkCallback) error {
	verbose := viper.GetBool("verbose")
	absRepoRoot, _ := filepath.Abs(repoRoot) // Assume repoRoot is valid

	// Load .llmignore if it exists
	llmIgnorer, err := gitignore.CompileIgnoreFile(filepath.Join(absRepoRoot, ".llmignore"))
	if err != nil && verbose {
		log.Printf("Note: No .llmignore file found: %v", err)
	}

	return filepath.WalkDir(absStartPath, func(absPath string, d fs.DirEntry, err error) error {
		if err != nil {
			// Error accessing file/directory, report and potentially skip
			log.Printf("Warning: Error accessing %s: %v. Skipping.", absPath, err)
			if d != nil && d.IsDir() {
				return filepath.SkipDir // Skip contents of this directory if possible
			}
			return nil // Skip this file/entry
		}

		// Get relative path for matching and reporting
		relPath, err := filepath.Rel(absStartPath, absPath)
		if err != nil {
			log.Printf("Warning: Could not get relative path for %s (start: %s): %v. Skipping.", absPath, absStartPath, err)
			return nil // Skip if relative path fails
		}

		// --- Filtering Logic ---
		// 1. Skip ignored files/dirs (using absolute path for matching convenience with go-gitignore)
		// Ensure paths use forward slashes for consistent matching with gitignore patterns
		matchPathForIgnore := absPath // Use absolute for go-gitignore
		if d.IsDir() {
			// Some ignore patterns require a trailing slash for dirs
			matchPathForIgnore = strings.TrimSuffix(matchPathForIgnore, string(filepath.Separator)) + "/"
		}

		// Check both .gitignore and .llmignore
		if ignorer != nil && ignorer.MatchesPath(matchPathForIgnore) {
			if verbose {
				log.Printf("Walker: Gitignore rule matched %s", relPath)
			}
			if d.IsDir() {
				return filepath.SkipDir // Skip ignored directories
			}
			return nil // Skip ignored files
		}

		if llmIgnorer != nil && llmIgnorer.MatchesPath(matchPathForIgnore) {
			if verbose {
				log.Printf("Walker: LLMignore rule matched %s", relPath)
			}
			if d.IsDir() {
				return filepath.SkipDir // Skip ignored directories
			}
			return nil // Skip ignored files
		}

		// 2. Skip directories themselves (we only process files in the callback)
		if d.IsDir() {
			// Skip common directories that should be ignored
			if d.Name() == "node_modules" || d.Name() == "vendor" || d.Name() == ".git" {
				return filepath.SkipDir
			}
			// Skip common hidden/build directories explicitly if not caught by ignores
			name := d.Name()
			if name != "." && strings.HasPrefix(name, ".") && name != ".github" && name != ".vscode" { // Keep .github, .vscode
				if verbose {
					log.Printf("Walker: Skipping hidden directory: %s", relPath)
				}
				return filepath.SkipDir
			}
			// Could add more explicit dir skips like node_modules, vendor etc. here
			// if ignorer isn't reliable or present
			return nil // Continue walking into non-ignored dirs
		}

		// 3. Detect language
		lang := language.Detect(absPath)
		if lang == "" {
			if verbose {
				log.Printf("Walker: Skipping file with unknown language/type: %s", relPath)
			}
			return nil // Skip files we can't identify
		}

		// 4. Check if likely text file
		isText, textCheckErr := util.IsLikelyTextFile(absPath)
		if textCheckErr != nil {
			log.Printf("Warning: Failed to check file type for %s: %v. Skipping.", absPath, textCheckErr)
			return nil
		}
		if !isText {
			if verbose {
				log.Printf("Walker: Skipping likely binary file: %s", relPath)
			}
			return nil
		}

		// --- If all checks pass, call the callback ---
		// Pass the path relative to the *repo root* for consistency
		if verbose {
			log.Printf("Walker: Processing file: %s (lang: %s)", relPath, lang)
		}
		return callback(absRepoRoot, relPath, lang, d)
	})
}
