package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// Task represents the JSON payload received from competitive-companion.
type Task struct {
	Name        string          `json:"name"`
	Group       string          `json:"group"`
	URL         string          `json:"url"`
	Interactive bool            `json:"interactive"`
	TimeLimit   int             `json:"timeLimit"`
	MemoryLimit int             `json:"memoryLimit"`
	TestType    string          `json:"testType"`
	Input       StreamType      `json:"input"`
	Output      StreamType      `json:"output"`
	Languages   json.RawMessage `json:"languages"`
	Batch       BatchInfo       `json:"batch"`
	Tests       []TestCase      `json:"tests"`
}

// StreamType describes input/output stream configuration.
type StreamType struct {
	Type string `json:"type"`
}

// BatchInfo describes batch problem configuration.
type BatchInfo struct {
	ID   string `json:"id"`
	Size int    `json:"size"`
}

// TestCase holds a single sample input/output pair.
type TestCase struct {
	Input  string `json:"input"`
	Output string `json:"output"`
}

// MaxSamples is the maximum number of test cases accepted per problem.
const MaxSamples = 100

// SaveSamples writes test cases to individual .in/.out files and returns the count.
func SaveSamples(task Task, dir string) (int, error) {
	if len(task.Tests) > MaxSamples {
		return 0, fmt.Errorf("too many test cases: %d (max %d)", len(task.Tests), MaxSamples)
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return 0, fmt.Errorf("create samples directory: %w", err)
	}

	for i, test := range task.Tests {
		idx := i + 1
		inFile := filepath.Join(dir, strconv.Itoa(idx)+".in")
		outFile := filepath.Join(dir, strconv.Itoa(idx)+".out")

		if err := os.WriteFile(inFile, []byte(test.Input), 0600); err != nil {
			return i, fmt.Errorf("write %s: %w", inFile, err)
		}
		if err := os.WriteFile(outFile, []byte(test.Output), 0600); err != nil {
			return i, fmt.Errorf("write %s: %w", outFile, err)
		}
	}

	return len(task.Tests), nil
}
