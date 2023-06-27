/*
Copyright © 2023 Tom Béné <himself@fantomebeig.net>
*/

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/FantomeBeignet/y2storj/internal"
)

var config internal.Config

var rootCmd = &cobra.Command{
	Use:   "y2storj [flags] video destination",
	Short: "Download a Youtube video and store it in a Storj bucket",
	Long: `Download a Youtube video from its video ID and put it in a Storj bucket.
You will need an access grant to the Storj project where you wish to store the video.`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.Flags().
		String("storj.access-grant", "", "The access grant to your Storj project")
	rootCmd.Flags().String("video.quality", "", "The quality with which to download the video")
}

func initConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$XDG_CONFIG_HOME/y2storj")
	viper.BindPFlags(rootCmd.Flags())
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return
		} else {
			panic(fmt.Errorf("error reading config: %w\n", err))
		}
	}
	viper.Unmarshal(&config)
}
