package main

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
// We use a map for quick lookups.
var DefaultBinaryExtensions = map[string]struct{}{
	".png": {}, ".jpg": {}, ".jpeg": {}, ".gif": {}, ".bmp": {}, ".tiff": {}, ".ico": {},
	".pdf": {},
	".mp3": {}, ".wav": {}, ".ogg": {}, ".flac": {},
	".mp4": {}, ".avi": {}, ".mov": {}, ".mkv": {},
	".zip": {}, ".gz": {}, ".tar": {}, ".rar": {}, ".7z": {},
	".exe": {}, ".dll": {}, ".so": {}, ".dylib": {}, ".app": {},
	".o": {}, ".a": {}, ".obj": {},
	".class": {}, ".jar": {},
	".pyc": {}, ".pyo": {},
	".sqlite": {}, ".db": {},
	".woff": {}, ".woff2": {}, ".ttf": {}, ".otf": {}, ".eot": {},
	".DS_Store": {}, // Common macOS file
}

// IsLikelyTextFile checks if a file is likely text-based.
// It focuses on content-based detection by analyzing a small chunk to detect binary indicators.
func IsLikelyTextFile(path string) (bool, error) {
	// We still use extension checks as a fast first pass
	ext := strings.ToLower(filepath.Ext(path))
	if _, isBinaryExt := DefaultBinaryExtensions[ext]; isBinaryExt {
		return false, nil
	}

	// Try reading a small chunk to detect binary content
	file, err := os.Open(path)
	if err != nil {
		// If we can't open it, we probably can't read it anyway
		return false, fmt.Errorf("could not open file %s: %w", path, err)
	}
	defer file.Close()

	// Read a larger chunk (4KB) for more accurate detection
	buffer := make([]byte, 4096)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return false, fmt.Errorf("could not read file %s: %w", path, err)
	}

	if n == 0 {
		return true, nil // Empty file is considered text
	}

	chunk := buffer[:n]

	// Check for non-UTF8 sequences
	if !utf8.Valid(chunk) {
		// Before declaring it binary, check if it might be UTF-16 BOM
		if len(chunk) >= 2 {
			if (chunk[0] == 0xFE && chunk[1] == 0xFF) || (chunk[0] == 0xFF && chunk[1] == 0xFE) {
				// Looks like UTF-16, which we won't handle correctly. Treat as binary.
				return false, nil
			}
		}
		return false, nil // Contains invalid UTF-8 sequences
	}

	// Check for excessive null bytes (common in binary files)
	// Allow a very small number for text files with occasional nulls
	nullCount := bytes.Count(chunk, []byte{0})
	if nullCount > 2 { // More than 2 null bytes in first 4KB suggests binary
		return false, nil
	}

	return true, nil
}

// ReadFileContent reads file content as a string, trying UTF-8 then Latin-1.
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
	// Add a warning comment? Maybe not, let the LLM figure it out.
	// fmt.Fprintf(os.Stderr, "Warning: File %s was not valid UTF-8, read as Latin-1.\n", path)
	return latin1Builder.String(), nil

	// Alternative: Use iconv or a more robust decoding library if needed,
	// but for LLM context, Latin-1 fallback is often sufficient.
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
