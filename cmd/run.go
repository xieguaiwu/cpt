package cmd

import (
	"cpt/internal"

	"github.com/spf13/cobra"
)

var (
	runDir     string
	runTimeout int
	runSample  int
)

var runCmd = &cobra.Command{
	Use:   "run <binary>",
	Short: "Run a compiled binary against saved sample test cases",
	Long: `Execute a compiled binary against sample input/output files in the samples directory.
For each sample, pipes the input file to stdin, captures stdout, and compares with the expected output.
Reports colored verdicts: AC (Accepted), WA (Wrong Answer), TLE (Timeout), RE (Runtime Error).`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		binary := args[0]
		return internal.RunAll(binary, runDir, runTimeout, runSample)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&runDir, "dir", "d", "samples", "Directory containing sample .in/.out files")
	runCmd.Flags().IntVarP(&runTimeout, "timeout", "t", 2, "Timeout in seconds per test case")
	runCmd.Flags().IntVarP(&runSample, "sample", "s", 0, "Run only sample N (0 = all)")
}
