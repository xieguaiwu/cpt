package cmd

import (
	"fmt"
	"os"

	"cpt/internal"

	"github.com/spf13/cobra"
)

var (
	testDir     string
	testTimeout int
	testSample  int
)

var testCmd = &cobra.Command{
	Use:   "test <source_file>",
	Short: "Compile and run a solution against sample test cases",
	Long: `Auto-detect language from file extension, compile the source (for compiled languages),
and run the resulting binary against sample input/output files.

Supported languages:
  .cpp  → g++ -std=c++17 -O2
  .c    → gcc -std=c11 -O2
  .py   → python3 (interpreted, no compile)
  .java → javac → java
  .rs   → rustc -O
  .go   → go build`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		source := args[0]

		// Check source exists
		if _, err := os.Stat(source); os.IsNotExist(err) {
			return fmt.Errorf("source file not found: %s", source)
		}

		lang := internal.DetectLang(source)
		if lang == "unknown" {
			return fmt.Errorf("unknown language for file: %s (supported: .cpp, .c, .py, .java, .rs, .go)", source)
		}

		fmt.Printf("🔍 Detected language: %s\n", lang)

		binary, err := internal.Compile(source, lang)
		if err != nil {
			return fmt.Errorf("compilation failed: %w", err)
		}

		// The compiled binary is intentionally kept in the source directory
		// (named after the source, e.g. main.cpp → ./main) so it can be reused
		// for custom samples, e.g. `./main < in.txt` or `cpt run ./main`.
		// Java already keeps its .class files; Python produces no artifact.

		return internal.RunAll(binary, testDir, testTimeout, testSample)
	},
}

func init() {
	rootCmd.AddCommand(testCmd)
	testCmd.Flags().StringVarP(&testDir, "dir", "d", "samples", "Directory containing sample .in/.out files")
	testCmd.Flags().IntVarP(&testTimeout, "timeout", "t", 2, "Timeout in seconds per test case")
	testCmd.Flags().IntVarP(&testSample, "sample", "s", 0, "Run only sample N (0 = all)")
}
