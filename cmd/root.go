package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"allaboutapps.dev/aw/de-witt/internal/config"
	"allaboutapps.dev/aw/de-witt/internal/jira"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Version: config.GetFormattedBuildArgs(),
	Use:     "app",
	Short:   config.ModuleName,
	Long: fmt.Sprintf(`%v

A tool to transfer your external Jira bookings to the internal one.
Requires configuration through config.yml.
Created by Witcher.`, config.ModuleName),
	Run: main,
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
	rootCmd.SetVersionTemplate(`{{printf "%s\n" .Version}}`)
	rootCmd.Flags().StringP("config", "c", "", "(Optional) config file path")
}

func main(cmd *cobra.Command, args []string) {
	// config
	config := config.Config{
		Logger: config.Logger{
			Level:              zerolog.DebugLevel,
			PrettyPrintConsole: true,
		},
	}

	// logger
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.SetGlobalLevel(config.Logger.Level)
	if config.Logger.PrettyPrintConsole {
		log.Logger = log.Output(zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
			w.TimeFormat = "15:04:05"
		}))
	}

	// read config from file
	ex, err := os.Executable()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get executable path")
	}
	exPath := filepath.Dir(ex)
	configPath := filepath.Join(exPath, "config.yml")

	// check if config file is provided
	configFlag, err := cmd.Flags().GetString("config")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get config flag")
	}
	if configFlag != "" {
		configPath = configFlag
	}

	configFile, err := os.Open(configPath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to open config file")
	}
	defer configFile.Close()

	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to read config file")
	}
	if err := yaml.Unmarshal(configBytes, &config); err != nil {
		log.Fatal().Err(err).Msg("failed to unmarshal config file")
	}

	service, err := jira.NewService(config.AppConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create jira service")
	}

	// date := time.Date(2023, time.December, 6, 10, 0, 0, 0, time.UTC)
	date := time.Now()
	if err := service.BookInternal(date); err != nil {
		log.Fatal().Err(err).Msg("failed to book internal issues")
	}

}

func getEnv(key string, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}

	return defaultVal
}
