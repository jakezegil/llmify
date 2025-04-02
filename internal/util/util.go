package util

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

// IsLikelyTextFile checks if a file is likely to be a text file.
func IsLikelyTextFile(filePath string) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, fmt.Errorf("opening file %s: %w", filePath, err)
	}
	defer file.Close()

	// Read first 1024 bytes
	buf := make([]byte, 1024)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return false, fmt.Errorf("reading file %s: %w", filePath, err)
	}

	// Check if the content is valid UTF-8
	if !utf8.Valid(buf[:n]) {
		return false, nil
	}

	// Check for common binary file signatures
	// This is a basic check - you might want to add more signatures
	binarySignatures := [][]byte{
		{0x00, 0x00, 0x00}, // Null bytes
		{0xFF, 0xD8, 0xFF}, // JPEG
		{0x89, 0x50, 0x4E}, // PNG
		{0x47, 0x49, 0x46}, // GIF
		{0x49, 0x49, 0x2A}, // TIFF
		{0x4D, 0x4D, 0x00}, // TIFF
		{0x25, 0x50, 0x44}, // PDF
		{0x50, 0x4B, 0x03}, // ZIP
		{0x1F, 0x8B, 0x08}, // GZIP
		{0x37, 0x7A, 0xBC}, // 7Z
		{0x52, 0x61, 0x72}, // RAR
		{0x4D, 0x5A, 0x90}, // EXE/DLL
		{0x7F, 0x45, 0x4C}, // ELF
		{0xCA, 0xFE, 0xBA}, // Java class
		{0xFE, 0xED, 0xFA}, // Mach-O
		{0x00, 0x00, 0xFE}, // Mach-O
	}

	for _, sig := range binarySignatures {
		if bytes.HasPrefix(buf[:n], sig) {
			return false, nil
		}
	}

	// Check for high ratio of control characters
	controlChars := 0
	for i := 0; i < n; i++ {
		if buf[i] < 32 && buf[i] != 9 && buf[i] != 10 && buf[i] != 13 { // Tab, LF, CR
			controlChars++
		}
	}
	if float64(controlChars)/float64(n) > 0.3 { // More than 30% control characters
		return false, nil
	}

	return true, nil
}

// ReadFileContent reads a file's content, handling different encodings.
func ReadFileContent(path string) (string, error) {
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading file %s: %w", path, err)
	}

	if utf8.Valid(contentBytes) {
		return string(contentBytes), nil
	}

	// If not valid UTF-8, try Latin-1 (ISO-8859-1) as a fallback
	var latin1Builder strings.Builder
	latin1Builder.Grow(len(contentBytes))
	for _, b := range contentBytes {
		latin1Builder.WriteRune(rune(b))
	}
	return latin1Builder.String(), nil
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

// CopyToClipboard copies a string to the clipboard.
func CopyToClipboard(content string) error {
	// For Windows
	if runtime.GOOS == "windows" {
		cmd := exec.Command("powershell", "-command", "Set-Clipboard", "-Value", content)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("windows clipboard error: %w - %s", err, stderr.String())
		}
		return nil
	}

	// For macOS
	if runtime.GOOS == "darwin" {
		cmd := exec.Command("pbcopy")
		cmd.Stdin = strings.NewReader(content)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("macOS clipboard error: %w", err)
		}
		return nil
	}

	// For Linux/Unix
	// Try xclip first
	if _, err := exec.LookPath("xclip"); err == nil {
		cmd := exec.Command("xclip", "-selection", "clipboard")
		cmd.Stdin = strings.NewReader(content)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("xclip error: %w", err)
		}
		return nil
	}

	// Try xsel if xclip not available
	if _, err := exec.LookPath("xsel"); err == nil {
		cmd := exec.Command("xsel", "--clipboard", "--input")
		cmd.Stdin = strings.NewReader(content)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("xsel error: %w", err)
		}
		return nil
	}

	// Try wl-copy for Wayland
	if _, err := exec.LookPath("wl-copy"); err == nil {
		cmd := exec.Command("wl-copy")
		cmd.Stdin = strings.NewReader(content)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("wl-copy error: %w", err)
		}
		return nil
	}

	return fmt.Errorf("no clipboard utility available")
}
