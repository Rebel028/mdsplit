package main

import (
	"os"

	"github.com/Rebel028/mdsplit/internal/splitter"
	"github.com/spf13/cobra"
)

// version is set at link time via -ldflags "-X 'main.version=...'"
var version = "dev"

var opts splitter.Options

var rootCmd = &cobra.Command{
	Use:   "mdsplit [input-file]",
	Short: "Split a markdown file into multiple files by headings",
	Long:  `A robust CLI tool to parse and split monolithic markdown documents into discrete chunks based on ATX headings.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return splitter.Split(args[0], opts)
	},
}

func main() {
	rootCmd.Version = version

	rootCmd.Flags().IntVarP(&opts.MaxLevel, "max-level", "l", 1, "maximum heading level to split (1-6)")
	rootCmd.Flags().BoolVarP(&opts.TOC, "table-of-contents", "t", false, "generate a table of contents")
	rootCmd.Flags().BoolVarP(&opts.Navigation, "navigation", "n", false, "add a navigation footer on each page")
	rootCmd.Flags().StringVarP(&opts.OutputDir, "output", "o", "", "path to output folder (must not exist)")
	rootCmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "write into output folder even if it already exists")
	rootCmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, "enable detailed processing output logs")
	rootCmd.Flags().BoolVarP(&opts.NumberedPrefix, "numbered", "u", false, "enable numeric prefix for files and multi-level ordered lists in toc")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
