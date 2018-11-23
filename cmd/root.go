package cmd

import (
	"fmt"
	"os"

	"git.digineo.de/digineo/zackup/config"
	"github.com/digineo/goldflags"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	log = logrus.WithField("prefix", "commands")

	tree         = config.NewTree("")
	treeRoot     string
	treeCallback func(config.Tree)
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "zackup",
	Short:   "A small utility to backup remote hosts into local ZFS datasets.",
	Version: goldflags.VersionString(),
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
//
// The (optional) callback function is called once the config tree was (re-) loaded.
func Execute(callback func(config.Tree)) {
	if callback != nil && treeCallback == nil {
		treeCallback = callback
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&treeRoot, "root", "",
		fmt.Sprintf("config root directory (default %q)", config.DefaultRoot))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if treeRoot == "" {
		if envRoot := os.Getenv("ZACKUP_ROOT"); envRoot != "" {
			treeRoot = envRoot
		}
	}

	if err := tree.SetRoot(treeRoot); err != nil {
		log.Fatalf("failed to read config tree: %v", err)
	}

	if treeCallback != nil {
		treeCallback(tree)
	}

	hosts := tree.Hosts()
	injectHostArgs(hosts, runCmd)
	injectHostArgs(hosts, statusCmd)
}

func injectHostArgs(hosts []string, cmd *cobra.Command) {
	cmd.ValidArgs = hosts
	cmd.Args = cobra.OnlyValidArgs
}