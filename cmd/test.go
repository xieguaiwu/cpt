package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"cpt/internal"

	"github.com/spf13/cobra"
)

var (
	testDir      string
	testTimeout  int
	testSample   int
	testWait     bool
	testWaitSecs int
	testPort     int
	testHost     string
	testSecret   string
)

var testCmd = &cobra.Command{
	Use:   "test <source_file>",
	Short: "Compile and run a solution against sample test cases",
	Long: `Auto-detect language from file extension, compile the source (for compiled languages),
and run the resulting binary against sample input/output files.

If no samples exist yet (or --wait is given), cpt starts a temporary HTTP server
and blocks until competitive-companion delivers a problem — merging serve + test
into a single command.

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

		// Wait mode: if no samples are present (or --wait forces it), start a
		// temporary server and block until a problem arrives, then test.
		if testWait || !hasSamples(testDir) {
			if err := waitForSamples(testHost, testPort, testWaitSecs, testSecret, testDir); err != nil {
				if errors.Is(err, errWaitAborted) {
					return nil // user aborted — exit cleanly
				}
				return err
			}
		}

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

// hasSamples reports whether any sample input file (*.in) exists in dir.
func hasSamples(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".in") {
			return true
		}
	}
	return false
}

// clearSamples removes stale sample files so a freshly received problem is
// the only one in the directory (avoids leftover tests when the new problem
// has fewer samples).
func clearSamples(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".in") || strings.HasSuffix(name, ".out") {
			os.Remove(filepath.Join(dir, name))
		}
	}
}

// errWaitAborted is returned when the user aborts waiting with Ctrl+C.
var errWaitAborted = errors.New("waiting aborted")

// waitForSamples starts a temporary HTTP server and blocks until the first
// problem arrives from competitive-companion, then stops the server.
func waitForSamples(host string, port, timeoutSecs int, secret, dir string) error {
	// A wait is always for a fresh problem: drop stale samples first.
	clearSamples(dir)

	srv := internal.NewServer(dir, "", secret)

	received := make(chan int, 1)
	srv.SetOnProblem(func(count int) { received <- count })

	// Graceful abort on Ctrl+C
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	errCh := make(chan error, 1)
	go func() {
		if err := srv.Start(host, port); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	fmt.Printf("⏳ No samples in %s/ — waiting for competitive-companion…\n", dir)
	fmt.Printf("   Open a problem page and click the extension button (server on http://%s:%d)\n", host, port)
	if timeoutSecs > 0 {
		fmt.Printf("   Timeout: %d s — Ctrl+C to abort\n", timeoutSecs)
	} else {
		fmt.Println("   Waiting indefinitely — Ctrl+C to abort")
	}

	var timeout <-chan time.Time
	if timeoutSecs > 0 {
		t := time.NewTimer(time.Duration(timeoutSecs) * time.Second)
		defer t.Stop()
		timeout = t.C
	}

	select {
	case count := <-received:
		srv.Stop()
		fmt.Printf("   📥 Received problem with %d sample(s) — testing now\n", count)
		return nil
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	case <-timeout:
		srv.Stop()
		return fmt.Errorf("timed out after %d s waiting for samples", timeoutSecs)
	case <-sigCh:
		srv.Stop()
		fmt.Println("\n   Waiting aborted")
		return errWaitAborted
	}
}

func init() {
	rootCmd.AddCommand(testCmd)
	testCmd.Flags().StringVarP(&testDir, "dir", "d", "samples", "Directory containing sample .in/.out files")
	testCmd.Flags().IntVarP(&testTimeout, "timeout", "t", 2, "Timeout in seconds per test case")
	testCmd.Flags().IntVarP(&testSample, "sample", "s", 0, "Run only sample N (0 = all)")
	testCmd.Flags().BoolVar(&testWait, "wait", false, "Wait for a new problem even if samples already exist")
	testCmd.Flags().IntVar(&testWaitSecs, "wait-timeout", 0, "Max seconds to wait for samples (0 = wait forever)")
	testCmd.Flags().IntVarP(&testPort, "port", "p", 27121, "Port for the temporary server (wait mode)")
	testCmd.Flags().StringVarP(&testHost, "host", "H", "127.0.0.1", "Host to bind for the temporary server (wait mode)")
	testCmd.Flags().StringVar(&testSecret, "secret", "", "Shared secret for the temporary server (wait mode)")
}
