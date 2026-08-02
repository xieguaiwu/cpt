package internal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DetectLang determines the programming language from a file extension.
func DetectLang(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".cpp", ".cc", ".cxx":
		return "cpp"
	case ".c":
		return "c"
	case ".py", ".py3":
		return "python"
	case ".java":
		return "java"
	case ".rs":
		return "rust"
	case ".go":
		return "go"
	default:
		return "unknown"
	}
}

// Compile compiles or prepares the source file for execution.
// Returns the command/executable to run and any compilation error.
func Compile(source string, lang string) (string, error) {
	switch lang {
	case "cpp":
		return compileGXX(source)
	case "c":
		return compileGCC(source)
	case "rust":
		return compileRustc(source)
	case "go":
		return compileGo(source)
	case "java":
		return compileJavac(source)
	case "python":
		return "python3 " + source, nil
	default:
		return "", fmt.Errorf("unsupported language: %s", lang)
	}
}

func compileGXX(source string) (string, error) {
	out := binaryPath(source)
	// Flags aligned with ~/.config/nvim/lua/utils.lua (CompileRun):
	// g++ -std=c++17 -O2 -Wall -Wextra -Wshadow
	cmd := exec.Command("g++", "-std=c++17", "-O2", "-Wall", "-Wextra", "-Wshadow", "-o", out, source)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("g++ compilation failed: %w", err)
	}
	return out, nil
}

func compileGCC(source string) (string, error) {
	out := binaryPath(source)
	cmd := exec.Command("gcc", "-std=c11", "-O2", "-o", out, source)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gcc compilation failed: %w", err)
	}
	return out, nil
}

func compileRustc(source string) (string, error) {
	out := binaryPath(source)
	cmd := exec.Command("rustc", "-O", "-o", out, source)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("rustc compilation failed: %w", err)
	}
	return out, nil
}

func compileGo(source string) (string, error) {
	out := binaryPath(source)
	cmd := exec.Command("go", "build", "-o", out, source)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("go build failed: %w", err)
	}
	return out, nil
}

func compileJavac(source string) (string, error) {
	// javac compiles .java → .class files in same directory
	cmd := exec.Command("javac", source)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("javac compilation failed: %w", err)
	}

	// Extract class name (filename without .java extension)
	base := filepath.Base(source)
	className := strings.TrimSuffix(base, filepath.Ext(base))
	dir := filepath.Dir(source)

	return "java -cp " + dir + " " + className, nil
}

// binaryPath returns the output path for the compiled binary: same directory
// as the source, named after the source basename without extension
// (main.cpp → ./main), mirroring nvim's `-o %<` behavior.
// The binary is intentionally kept after testing so it can be reused,
// e.g. for running custom samples manually or via `cpt run ./main`.
// Note: a bare "main" would make exec fall back to $PATH lookup and fail,
// so relative paths get an explicit "./" prefix.
func binaryPath(source string) string {
	base := filepath.Base(source)
	name := strings.TrimSuffix(base, filepath.Ext(source))
	dir := filepath.Dir(source)
	if dir == "." {
		return "./" + name
	}
	return filepath.Join(dir, name)
}
