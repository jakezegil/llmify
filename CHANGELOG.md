markdown
# Changelog

## v0.0.8 (2025-03-30)

- Update `llmify docs` command to allow updating documentation without specifying a file or directory, defaulting to the current directory.
- Enhance `llmify docs` command with a default prompt for documentation updates if none is provided.
- Refactor `llmify refactor` command to only target a single file, with improved handling for language detection and file processing.
- Introduce support for formatting and linting tools in the `llmify refactor` command based on file type.
- Add support for loading standards configuration from `.llmify_standards.yaml` for LLM rules.
- Improve error handling and logging for file processing in both `llmify docs` and `llmify refactor` commands.
- Update internal utilities for better file type detection and handling of binary files.

## v0.0.7 (2025-03-29)

- Add `llmify commit` command for AI-powered commit message generation
- Add support for automatic documentation updates with `--docs` flag
- Add configuration system with `.llmifyrc.yaml` and environment variables
- Add support for multiple LLM providers (OpenAI, Anthropic, Ollama)
- Add global flags for verbose output and LLM timeout
- Improve error handling and user feedback

## v0.0.6 (2025-03-26)

- update binaries

## v0.0.5 (2025-03-26)

- npm readme

## v0.0.4 (2025-03-26)

- update readme

## v0.0.3 (2025-03-26)

- add intelligent defaults

## v0.0.2 (2025-03-26)

- add dummy build script
- initial release