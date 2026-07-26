package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cpt",
	Short: "Competitive Programming Tool - terminal companion for competitive-companion",
	Long: `cpt is a terminal companion for the competitive-companion browser extension.
It receives problem data via HTTP, saves sample test cases, compiles solutions,
and runs them against samples with colored result reporting.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
