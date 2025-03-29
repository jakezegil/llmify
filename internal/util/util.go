package util

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// DefaultBinaryExtensions is a set of common binary file extensions.
var DefaultBinaryExtensions = map[string]struct{}{
	// Common Images
	".png": {}, ".jpg": {}, ".jpeg": {}, ".gif": {}, ".bmp": {}, ".tiff": {}, ".ico": {}, ".webp": {},
	// Documents
	".pdf": {}, ".doc": {}, ".docx": {}, ".ppt": {}, ".pptx": {}, ".xls": {}, ".xlsx": {}, ".odt": {}, ".odp": {}, ".ods": {},
	// Audio
	".mp3": {}, ".wav": {}, ".ogg": {}, ".flac": {}, ".aac": {}, ".m4a": {},
	// Video
	".mp4": {}, ".avi": {}, ".mov": {}, ".mkv": {}, ".wmv": {}, ".flv": {}, ".webm": {},
	// Archives
	".zip": {}, ".gz": {}, ".tar": {}, ".rar": {}, ".7z": {}, ".bz2": {}, ".xz": {}, ".tgz": {},
	// Executables & Libraries
	".exe": {}, ".dll": {}, ".so": {}, ".dylib": {}, ".app": {}, ".msi": {}, ".deb": {}, ".rpm": {},
	// Compiled Code / Intermediates
	".o": {}, ".a": {}, ".obj": {}, ".lib": {}, ".class": {}, ".jar": {}, ".pyc": {}, ".pyo": {}, ".wasm": {},
	// Databases
	".sqlite": {}, ".db": {}, ".mdb": {}, ".accdb": {}, ".sqlite3": {},
	// Fonts
	".woff": {}, ".woff2": {}, ".ttf": {}, ".otf": {}, ".eot": {},
	// System / Misc
	".DS_Store": {}, "Thumbs.db": {}, ".bak": {}, ".tmp": {}, ".swp": {}, ".swo": {},
	// Package locks
	"package-lock.json": {}, "yarn.lock": {}, "composer.lock": {}, "go.sum": {}, "Cargo.lock": {}, "Gemfile.lock": {}, "Pipfile.lock": {}, "poetry.lock": {}, "pnpm-lock.yaml": {},
}

// IsLikelyTextFile checks if a file is likely text-based.
// It uses extension check first, then content analysis.
func IsLikelyTextFile(path string) (bool, error) {
	// 1. Check by filename/extension for known binary types or locks
	baseName := filepath.Base(path)
	ext := strings.ToLower(filepath.Ext(baseName))

	if _, isKnownBinary := DefaultBinaryExtensions[ext]; isKnownBinary {
		return false, nil
	}
	// Check full filename for files like lockfiles without extensions
	if _, isKnownBinary := DefaultBinaryExtensions[baseName]; isKnownBinary {
		return false, nil
	}

	// 2. Check content for binary indicators
	file, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("could not open file %s: %w", path, err)
	}
	defer file.Close()

	// Read a chunk (e.g., first 4KB)
	buffer := make([]byte, 4096)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return false, fmt.Errorf("could not read file %s: %w", path, err)
	}

	if n == 0 {
		return true, nil // Empty file is considered text
	}

	chunk := buffer[:n]

	// Check for UTF-16 BOMs (treat as binary for simplicity for now)
	if len(chunk) >= 2 {
		if (chunk[0] == 0xFE && chunk[1] == 0xFF) || (chunk[0] == 0xFF && chunk[1] == 0xFE) {
			return false, nil // UTF-16 BOM detected
		}
	}

	// Check for excessive null bytes (common in binary files)
	nullCount := bytes.Count(chunk, []byte{0})
	if n > 0 && float64(nullCount)/float64(n) > 0.1 { // More than 10% null bytes suggests binary
		return false, nil
	}

	// Check for non-printable characters / non-UTF8
	if !utf8.Valid(chunk) {
		return false, nil
	}

	return true, nil // Likely text if it passes checks
}

// ReadFileContent reads file content as a string, assuming UTF-8.
func ReadFileContent(path string) (string, error) {
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading file %s: %w", path, err)
	}
	if !utf8.Valid(contentBytes) {
		return "", fmt.Errorf("file %s contains invalid UTF-8 sequences", path)
	}
	return string(contentBytes), nil
}

// WriteStringToFile writes a string to a file, creating directories if needed.
func WriteStringToFile(filePath string, content string) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("creating file %s: %w", filePath, err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	_, err = writer.WriteString(content)
	if err != nil {
		return fmt.Errorf("writing to file %s: %w", filePath, err)
	}
	err = writer.Flush()
	if err != nil {
		return fmt.Errorf("flushing file %s: %w", filePath, err)
	}
	return nil
}

// CleanLLMResponse removes markdown code fences and other formatting from LLM responses
func CleanLLMResponse(response string) string {
	// Trim leading/trailing whitespace
	cleaned := strings.TrimSpace(response)

	// Common language identifiers used in markdown fences
	codeBlockPrefixes := []string{
		"```typescript", "```tsx", "```javascript", "```js",
		"```python", "```py",
		"```go", "```golang",
		"```rust", "```rs",
		"```java", "```kotlin", "```scala",
		"```csharp", "```cs",
		"```c", "```cpp", "```objectivec",
		"```php",
		"```ruby", "```rb",
		"```swift",
		"```bash", "```sh", "```zsh",
		"```yaml", "```yml",
		"```json",
		"```html", "```xml", "```css", "```scss", "```less",
		"```markdown", "```md",
		"```sql",
		"```text",
		"```", // Generic fence
	}

	// Remove opening code fence and optional language identifier
	cleanedLower := strings.ToLower(cleaned)
	for _, prefix := range codeBlockPrefixes {
		if strings.HasPrefix(cleanedLower, prefix) {
			// Find the first newline after the prefix
			firstNewline := strings.Index(cleaned[len(prefix):], "\n")
			if firstNewline != -1 {
				cleaned = strings.TrimSpace(cleaned[len(prefix)+firstNewline:])
				break
			} else {
				// Handle case where fence is the whole line or no content after it
				cleaned = strings.TrimSpace(cleaned[len(prefix):])
				break
			}
		}
	}

	// Remove closing code fence ```
	if strings.HasSuffix(cleaned, "```") {
		cleaned = strings.TrimSuffix(cleaned, "```")
		cleaned = strings.TrimSpace(cleaned)
	}

	return cleaned
}

// LimitString truncates a string to a max length for display.
func LimitString(s string, maxLen int) string {
	if len(s) > maxLen {
		// Try to find a newline near the limit for cleaner truncation
		safeCut := maxLen
		lastNL := strings.LastIndex(s[:maxLen], "\n")
		if lastNL > maxLen/2 { // Only use newline if it's reasonably far in
			safeCut = lastNL
		}
		return s[:safeCut] + "..."
	}
	return s
}
