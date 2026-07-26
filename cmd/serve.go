package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"cpt/internal"

	"github.com/spf13/cobra"
)

var (
	servePort    int
	serveDir     string
	serveRunBin  string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start HTTP server to receive problems from competitive-companion",
	Long: `Start an HTTP server on the specified port that listens for POST requests
from the competitive-companion browser extension. Receives problem data as JSON,
saves sample test cases, and optionally auto-runs a compiled binary against them.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If run binary is specified, verify it exists
		if serveRunBin != "" {
			if _, err := os.Stat(serveRunBin); os.IsNotExist(err) {
				return fmt.Errorf("binary not found: %s", serveRunBin)
			}
		}

		srv := internal.NewServer(serveDir, serveRunBin)
		addr := fmt.Sprintf(":%d", servePort)

		// Graceful shutdown on Ctrl+C
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-sigCh
			fmt.Println("\nShutting down...")
			srv.Stop()
			os.Exit(0)
		}()

		fmt.Printf("🚀 cpt server listening on http://localhost%s\n", addr)
		fmt.Printf("   Samples directory: %s\n", serveDir)
		if serveRunBin != "" {
			fmt.Printf("   Auto-run binary: %s\n", serveRunBin)
		}
		fmt.Println("   Waiting for competitive-companion to send problems...")

		if err := srv.Start(addr); err != nil {
			return fmt.Errorf("server error: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 27121, "Port to listen on")
	serveCmd.Flags().StringVarP(&serveDir, "dir", "d", "samples", "Directory for sample files")
	serveCmd.Flags().StringVarP(&serveRunBin, "run", "r", "", "Binary to auto-run after receiving problem")
}
