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

		// For interpreted languages, clean up is not needed
		// For compiled languages, binary is a temp file - clean up after
		if lang != "python" && lang != "java" {
			defer os.Remove(binary)
		}

		return internal.RunAll(binary, testDir, testTimeout, testSample)
	},
}

func init() {
	rootCmd.AddCommand(testCmd)
	testCmd.Flags().StringVarP(&testDir, "dir", "d", "samples", "Directory containing sample .in/.out files")
	testCmd.Flags().IntVarP(&testTimeout, "timeout", "t", 2, "Timeout in seconds per test case")
	testCmd.Flags().IntVarP(&testSample, "sample", "s", 0, "Run only sample N (0 = all)")
}
