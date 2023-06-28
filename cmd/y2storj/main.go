/*
Copyright © 2023 Tom Béné <himself@fantomebeig.net>
*/

package main

import (
	"fmt"
	"os"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/FantomeBeignet/y2storj"
)

var config y2storj.Config

var rootCmd = &cobra.Command{
	Use:   "y2storj [flags] video destination",
	Short: "Download a Youtube video and store it in a Storj bucket",
	Long: `Download a Youtube video from its video ID and put it in a Storj bucket.
You will need an access grant to the Storj project where you wish to store the video.`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		err := y2storj.DownloadAndStore(
			args[0],
			args[1],
			config.Storj.AccessGrant,
			config.Video.Quality,
		)
		cobra.CheckErr(err)
	},
}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.Flags().
		String("access-grant", "", "The access grant to your Storj project")
	rootCmd.Flags().StringP("quality", "q", "", "The quality with which to download the video")
}

func initConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	home, err := homedir.Dir()
	if err != nil {
		panic(fmt.Errorf("failed to get home dir: %s", err))
	}
	viper.AddConfigPath(home + ".config/y2storj")
	viper.SetDefault("video.quality", "best")
	viper.BindPFlag("storj.access_grant", rootCmd.Flags().Lookup("access-grant"))
	viper.BindPFlag("video.quality", rootCmd.Flags().Lookup("quality"))
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return
		} else {
			panic(fmt.Errorf("error reading config: %w\n", err))
		}
	}
	viper.Unmarshal(&config)
}
