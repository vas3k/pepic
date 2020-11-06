package cmd

import (
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/spf13/cobra"
	"github.com/vas3k/pepic/pepic/config"
	"log"
)

var (
	configFile string
	rootCmd    = &cobra.Command{
		Use: "pepic",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Usage()
		},
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file")

	rootCmd.AddCommand(serveCmd)
	// Add more commands here
}

func initConfig() {
	var err error

	if configFile != "" {
		err = cleanenv.ReadConfig(configFile, &config.App)
	} else {
		err = cleanenv.ReadConfig("/etc/pepic/config.yml", &config.App)
	}

	if err != nil {
		log.Fatalf("config error: %s", err)
	}
}
