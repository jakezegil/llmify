package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "llmify",
	Short: "LLM-powered code refactoring and documentation tool",
	Long: `llmify is a command-line tool that uses LLMs to help you refactor code
and update documentation. It supports multiple languages and can process
both single files and entire directories.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().Int("llm-timeout", 300, "Timeout in seconds for LLM requests")

	// Bind flags to viper
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("llm.timeout_seconds", rootCmd.PersistentFlags().Lookup("llm-timeout"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Set default values
	viper.SetDefault("llm.timeout_seconds", 300)
	viper.SetDefault("verbose", false)

	// Read environment variables
	viper.SetEnvPrefix("LLMIFY")
	viper.AutomaticEnv()
}
