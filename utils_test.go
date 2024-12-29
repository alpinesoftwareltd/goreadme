package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// TestIsValidDir tests the isValidDir function to ensure it correctly identifies
// valid and invalid directory paths. It runs a series of subtests with different
// directory paths and checks if the function's output matches the expected result.
// The test cases include:
// - A valid directory
// - A valid nested directory
// - An invalid directory (a file)
// - An invalid directory (non-existent)
func TestIsValidDir(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "valid directory",
			path: "tests",
			want: true,
		},
		{
			name: "valid directory nested",
			path: filepath.Join("tests", "src"),
			want: true,
		},
		{
			name: "invalid directory file",
			path: "main.go",
			want: false,
		},
		{
			name: "invalid directory non existent",
			path: "invalid",
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := isValidDir(test.path)
			if got != test.want {
				t.Errorf("got: %v, want: %v", got, test.want)
			}
		})
	}
}

// TestGetCliInput tests the getCliInput function by simulating user input from a buffered reader.
// It verifies that the function correctly reads the input and processes it using the provided action function.
// The test will fail if the output does not match the expected value or if an error occurs.
func TestGetCliInput(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("test cli input\n"))
	action := func(value string) (string, error) {
		return value, nil
	}

	output, err := getCliInput(reader, "Test prompt: ", action)
	if err != nil {
		t.Fatal(err)
	}

	if output != "test cli input" {
		t.Errorf("got: %s, want: %s", output, "test cli input")
	}
}

// TestGetCliInputWithAction tests the getCliInput function by simulating
// user input and applying an action function to the input. It verifies
// that the output matches the expected result after the action is applied.
//
// The test sets up a bufio.Reader with a predefined input string, defines
// an action function that appends " plus some extra" to the input, and
// checks if the output of getCliInput matches the expected modified input.
//
// If the output does not match the expected result, the test fails with
// an appropriate error message.
func TestGetCliInputWithAction(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("test cli input\n"))
	action := func(value string) (string, error) {
		return value + " plus some extra", nil
	}

	output, err := getCliInput(reader, "Test prompt: ", action)
	if err != nil {
		t.Fatal(err)
	}

	if output != "test cli input plus some extra" {
		t.Errorf("got: %s, want: %s", output, "test cli input plus some extra")
	}
}

// TestGetCliInputWithError tests the getCliInput function to ensure it correctly handles
// an error returned by the provided action function. It simulates user input and verifies
// that the expected error is returned and properly handled.
func TestGetCliInputWithError(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("test cli input\n"))
	action := func(value string) (string, error) {
		return value, fmt.Errorf("test error")
	}

	_, err := getCliInput(reader, "Test prompt: ", action)
	if err == nil {
		t.Fatal("expected an error, but got nil")
	}

	if err.Error() != "test error" {
		t.Errorf("got: %s, want: %s", err.Error(), "test error")
	}
}

// TestIsAllowedFile tests the isAllowedFile function to ensure it correctly
// identifies whether a given filename is allowed or not. It runs a series of
// subtests with different filenames and expected outcomes:
// - "allowed file": a simple allowed file (main.py).
// - "allowed file with dir": an allowed file within a directory (src/main.py).
// - "allowed file in whitelist": a file explicitly allowed (Dockerfile.test).
// - "disallowed file csv": a disallowed file type (data.csv).
// - "disallowed file json": another disallowed file type (example.json).
// The test will fail if the actual result from isAllowedFile does not match
// the expected result.
func TestIsAllowedFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{
			name:     "allowed file",
			filename: "main.py",
			want:     true,
		},
		{
			name:     "allowed file with dir",
			filename: "src/main.py",
			want:     true,
		},
		{
			name:     "disallowed file csv",
			filename: "data.csv",
			want:     false,
		},
		{
			name:     "disallowed file json",
			filename: "example.json",
			want:     false,
		},
		{
			name:     "disallowed node modules file",
			filename: "app/node_modules/something.js",
			want:     false,
		},
		{
			name:     "disallowed pycache",
			filename: "app/__pycache__/something.py",
			want:     false,
		},
		{
			name:     "allowed explicit mapping",
			filename: "test.vue",
			want:     true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, got := isAllowedFile(test.filename)
			if got != test.want {
				t.Errorf("got: %v, want: %v", got, test.want)
			}
		})
	}
}

// TestGetFilesToUpload tests the getFilesToUpload function.
// It sets the base directory to "tests/src" and checks if the function
// returns exactly 3 files to upload. If the number of files is not 3,
// or if there is an error, the test will fail.
func TestGetFilesToUpload(t *testing.T) {
	basedir := "tests/src"

	toUpload, err := getFilesToUpload(basedir)
	if err != nil {
		t.Fatal(err)
	}

	if len(toUpload) != 3 {
		t.Errorf("got: %d, want: %d", len(toUpload), 3)
	}
}

// TestCombineFiles tests the combineFiles function by opening a set of predefined
// file paths, reading their contents, and combining them into a single reader.
// It then compares the combined contents to an expected string to ensure the
// combineFiles function works correctly. If the combined contents do not match
// the expected string, the test fails with a descriptive error message.
func TestCombineFiles(t *testing.T) {
	paths := []string{
		"tests/src/main.py",
		"tests/src/nested/__init__.py",
		"tests/src/nested/example.py",
	}

	files := map[string]io.Reader{}
	for _, p := range paths {
		file, err := os.Open(p)
		if err != nil {
			t.Fatalf("error opening test file %s: %+v", p, err)
		}
		defer file.Close()

		contents, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("error reading test file %s: %+v", p, err)
		}
		buffer := bytes.NewBuffer(contents)
		files[p] = buffer
	}

	combined := combineFiles(files)
	bytesContent, err := io.ReadAll(combined)
	if err != nil {
		t.Fatalf("error reading combined file: %+v", err)
	}

	stringContent := string(bytesContent)
	expected := `### FILE START tests/src/main.py


def foo():
    return "bar"


### FILE END tests/src/main.py

### FILE START tests/src/nested/__init__.py



### FILE END tests/src/nested/__init__.py

### FILE START tests/src/nested/example.py


def some_example_function(bar: str):
    return "foo"


### FILE END tests/src/nested/example.py

`
	if stringContent != expected {
		t.Fatalf("combined files contents does not match expected: got %s, expected %s", stringContent, expected)
	}
}

// TestGroupFilesByExtension tests the groupFilesByExtension function to ensure
// that it correctly groups files by their extensions. It creates a map of file
// names to io.Reader objects, calls the groupFilesByExtension function, and
// verifies that the files are grouped as expected. The test checks that the
// number of groupings matches the expected count and that each expected
// grouping is present in the result.
func TestGroupFilesByExtension(t *testing.T) {
	var buffer bytes.Buffer

	files := map[string]io.Reader{
		"main.py":     &buffer,
		"example1.py": &buffer,
		"example2.py": &buffer,
		"source.txt":  &buffer,
		"data.json":   &buffer,
		"example3.py": &buffer,
	}

	grouped := groupFilesByExtension(files)

	groupings := []string{}
	for ext, groupedFiles := range grouped {
		for f := range groupedFiles {
			groupings = append(groupings, fmt.Sprintf("%s:%s", ext, f))
		}
	}

	expected := []string{
		".py:main.py",
		".py:example1.py",
		".py:example2.py",
		".txt:source.txt",
		".json:data.json",
		".py:example3.py",
	}
	if len(groupings) != len(expected) {
		t.Fatalf("expected %d unique groupings, got %d", len(expected), len(groupings))
	}

	for _, item := range expected {
		if !slices.Contains(groupings, item) {
			t.Fatalf("%s not in evaluated groupings", item)
		}
	}
}
