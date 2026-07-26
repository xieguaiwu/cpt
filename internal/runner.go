package internal

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
)

var (
	colorAC  = color.New(color.FgGreen, color.Bold)
	colorWA  = color.New(color.FgRed, color.Bold)
	colorTLE = color.New(color.FgYellow, color.Bold)
	colorRE  = color.New(color.FgMagenta, color.Bold)
	colorDim = color.New(color.Faint)
)

// RunAll finds and executes all sample tests against the given binary.
func RunAll(binary string, samplesDir string, timeoutSec int, sampleNum int) error {
	// Find all .in files
	entries, err := os.ReadDir(samplesDir)
	if err != nil {
		return fmt.Errorf("read samples directory %s: %w", samplesDir, err)
	}

	// Collect sample numbers
	samples := make(map[int]struct{})
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".in") {
			numStr := strings.TrimSuffix(e.Name(), ".in")
			if n, err := strconv.Atoi(numStr); err == nil && n > 0 {
				samples[n] = struct{}{}
			}
		}
	}

	if len(samples) == 0 {
		return fmt.Errorf("no sample files found in %s/ (expected 1.in/1.out, 2.in/2.out, ...)", samplesDir)
	}

	// Sort sample numbers
	nums := make([]int, 0, len(samples))
	for n := range samples {
		nums = append(nums, n)
	}
	sort.Ints(nums)

	// Filter if specific sample requested
	if sampleNum > 0 {
		if _, ok := samples[sampleNum]; !ok {
			return fmt.Errorf("sample %d not found in %s/", sampleNum, samplesDir)
		}
		nums = []int{sampleNum}
	}

	// Run tests
	type result struct {
		num     int
		verdict string
		diff    string
		elapsed time.Duration
		err     error
	}

	results := make([]result, 0, len(nums))

	fmt.Println()
	fmt.Println("═══════════════════════════════════")

	passed := 0
	for _, n := range nums {
		inFile := filepath.Join(samplesDir, strconv.Itoa(n)+".in")
		outFile := filepath.Join(samplesDir, strconv.Itoa(n)+".out")

		verdict, diff, elapsed, err := RunTest(binary, inFile, outFile, timeoutSec)
		if err != nil {
			fmt.Fprintf(os.Stderr, "   Sample %d: error: %v\n", n, err)
			continue
		}

		results = append(results, result{n, verdict, diff, elapsed, err})

		// Print single-sample result
		switch verdict {
		case "AC":
			colorAC.Printf("  Sample %d  ✅ AC", n)
			passed++
		case "WA":
			colorWA.Printf("  Sample %d  ❌ WA", n)
		case "TLE":
			colorTLE.Printf("  Sample %d  ⏱ TLE", n)
		case "RE":
			colorRE.Printf("  Sample %d  💥 RE", n)
		default:
			fmt.Printf("  Sample %d  ? %s", n, verdict)
		}
		colorDim.Printf("   (%s)\n", elapsed.Round(time.Millisecond))

		if diff != "" {
			fmt.Println("───────────────────────────────────")
			printDiff(diff)
		}
	}

	fmt.Println("═══════════════════════════════════")

	if passed == len(nums) {
		colorAC.Printf("  Passed: %d/%d\n", passed, len(nums))
	} else {
		colorWA.Printf("  Passed: %d/%d\n", passed, len(nums))
	}
	fmt.Println()

	return nil
}

// RunTest executes a binary against a single test case.
// Returns verdict (AC/WA/TLE/RE), diff text, elapsed time, and error.
func RunTest(binary string, inputFile string, outputFile string, timeoutSec int) (verdict string, diff string, elapsed time.Duration, err error) {
	// Parse the binary command (it may be "java -cp dir ClassName" or "python3 file.py")
	parts := strings.Fields(binary)
	if len(parts) == 0 {
		return "", "", 0, fmt.Errorf("empty binary command")
	}

	// Verify the executable exists (for single-binary commands, not interpreters)
	if len(parts) == 1 {
		if _, statErr := os.Stat(parts[0]); os.IsNotExist(statErr) {
			return "RE", fmt.Sprintf("  Binary not found: %s", parts[0]), 0, nil
		}
	}

	// Read input file
	input, err := os.ReadFile(inputFile)
	if err != nil {
		return "", "", 0, fmt.Errorf("read input file %s: %w", inputFile, err)
	}

	// Read expected output
	expected, err := os.ReadFile(outputFile)
	if err != nil {
		return "", "", 0, fmt.Errorf("read output file %s: %w", outputFile, err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	// Build command
	var cmd *exec.Cmd
	if len(parts) == 1 {
		cmd = exec.CommandContext(ctx, parts[0])
	} else {
		cmd = exec.CommandContext(ctx, parts[0], parts[1:]...)
	}

	cmd.Stdin = bytes.NewReader(input)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	runErr := cmd.Run()
	elapsed = time.Since(start)

	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		return "TLE", "", elapsed, nil
	}

	// Check for runtime error
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			diff := fmt.Sprintf("  Exit code: %d\n  Stderr:\n%s", exitErr.ExitCode(), stderr.String())
			return "RE", diff, elapsed, nil
		}
		return "RE", fmt.Sprintf("  Error: %v\n  Stderr:\n%s", runErr, stderr.String()), elapsed, nil
	}

	// Compare output
	got := stdout.String()
	expectedStr := string(expected)

	if compareOutput(got, expectedStr) {
		return "AC", "", elapsed, nil
	}

	diff = formatDiff(expectedStr, got)
	return "WA", diff, elapsed, nil
}

// compareOutput compares two outputs, trimming trailing whitespace from each line
// and ignoring trailing newline differences.
func compareOutput(got, expected string) bool {
	gotLines := strings.Split(got, "\n")
	expLines := strings.Split(expected, "\n")

	// Trim trailing empty lines
	for len(gotLines) > 0 && strings.TrimSpace(gotLines[len(gotLines)-1]) == "" {
		gotLines = gotLines[:len(gotLines)-1]
	}
	for len(expLines) > 0 && strings.TrimSpace(expLines[len(expLines)-1]) == "" {
		expLines = expLines[:len(expLines)-1]
	}

	if len(gotLines) != len(expLines) {
		return false
	}

	for i := range gotLines {
		gotTrimmed := trimTrailing(gotLines[i])
		expTrimmed := trimTrailing(expLines[i])
		if gotTrimmed != expTrimmed {
			return false
		}
	}

	return true
}

// trimTrailing removes trailing spaces and tabs from a string.
func trimTrailing(s string) string {
	return strings.TrimRight(s, " \t")
}

// formatDiff creates a human-readable diff between expected and got output.
func formatDiff(expected, got string) string {
	expLines := strings.Split(expected, "\n")
	gotLines := strings.Split(got, "\n")

	// Trim trailing empty lines for display
	for len(expLines) > 0 && expLines[len(expLines)-1] == "" {
		expLines = expLines[:len(expLines)-1]
	}
	for len(gotLines) > 0 && gotLines[len(gotLines)-1] == "" {
		gotLines = gotLines[:len(gotLines)-1]
	}

	var buf bytes.Buffer
	maxLines := len(expLines)
	if len(gotLines) > maxLines {
		maxLines = len(gotLines)
	}

	for i := 0; i < maxLines; i++ {
		var exp, g string
		if i < len(expLines) {
			exp = expLines[i]
		}
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if exp != g {
			buf.WriteString(fmt.Sprintf("  Line %d:\n", i+1))
			buf.WriteString(fmt.Sprintf("    Expected: %s\n", exp))
			buf.WriteString(fmt.Sprintf("    Got:      %s\n", g))
		}
	}
	return buf.String()
}

// printDiff prints the diff with colored headers.
func printDiff(diff string) {
	lines := strings.Split(strings.TrimRight(diff, "\n"), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "    Expected:") {
			color.Green(line)
		} else if strings.HasPrefix(line, "    Got:") {
			color.Red(line)
		} else {
			colorDim.Println(line)
		}
	}
}
