package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
)

// configureLogging takes a log level in string format
// and configures the sirupsen/logrus package. the provided
// log level string is case insensitive.
func configureLogging(level string) {
	switch strings.ToUpper(level) {
	case "DEBUG":
		log.SetLevel(log.DebugLevel)
	case "INFO":
		log.SetLevel(log.InfoLevel)
	case "WARN":
		log.SetLevel(log.WarnLevel)
	case "ERROR":
		log.SetLevel(log.ErrorLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}
}

// isValidDir takes a target directory path and checks
// that the path exists, and that the path corresponds
// to a directory.
func isValidDir(target string) bool {
	stat, err := os.Stat(target)
	return err == nil && stat.IsDir()
}

// getDefaultConfigPath retrieves the default config path
// based on the provided OS (usually his is ~/.goreadme/config.json).
// os.UserHomeDir is used to fin the root directory for the
// specified user.
func getDefaultConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".goreadme", "config.json")
}

// getCliInput retrieves a given value from std using the
// provided CLI. A follow on action can be optionally provided
func getCliInput(reader *bufio.Reader, prompt string, action func(value string) (string, error)) (string, error) {
	fmt.Print(prompt)
	// read value from stdin and remove \n characters
	value, _ := reader.ReadString('\n')
	value = strings.Replace(value, "\n", "", -1)

	// execute post action and return function
	value, err := action(value)
	if err != nil {
		return value, err
	} else {
		return value, nil
	}
}

// uploadFiles uploads multiple files concurrently using the provided ChatGPTAssistantClient.
// It limits the number of concurrent uploads using a semaphore with a weight of 5.
//
// Parameters:
//   - client: A pointer to a ChatGPTAssistantClient used to upload the files.
//   - files: A slice of io.Reader representing the files to be uploaded.
//
// Returns:
//   - A slice of strings containing the file IDs of the successfully uploaded files.
//   - A slice of errors containing any errors that occurred during the upload process.
func uploadFiles(client *ChatGPTAssistantClient, files map[string]io.Reader) ([]string, []error) {
	errors := []error{}
	fileIds := []string{}

	semaphore := semaphore.NewWeighted(5)

	var wg sync.WaitGroup
	wg.Add(len(files))

	for filename, filecontent := range files {

		if err := semaphore.Acquire(context.Background(), 1); err != nil {
			errors = append(errors, err)
			continue
		}

		go func(name string, content io.Reader) {
			defer wg.Done()
			defer semaphore.Release(1)

			fileId, err := client.UploadFile(name, content)
			if err != nil {
				errors = append(errors, err)
			} else {
				fileIds = append(fileIds, fileId)
			}
		}(filename, filecontent)
	}
	wg.Wait()

	return fileIds, errors
}

func deleteFiles(client *ChatGPTAssistantClient, fileIds []string) []error {
	errors := []error{}

	semaphore := semaphore.NewWeighted(5)

	var wg sync.WaitGroup
	wg.Add(len(fileIds))

	for _, fid := range fileIds {

		if err := semaphore.Acquire(context.Background(), 1); err != nil {
			errors = append(errors, err)
			continue
		}

		go func(id string) {
			defer wg.Done()
			defer semaphore.Release(1)

			if err := client.DeleteFile(id); err != nil {
				errors = append(errors, err)
			}
		}(fid)
	}
	wg.Wait()

	return errors
}

// isAllowedFile checks if the given filename has an allowed extension.
// It returns true if the filename ends with one of the allowed extensions, otherwise false.
func isAllowedFile(filename string) (string, bool) {

	blackListedRegex := []string{
		`(^|[\/])node_modules([\/]|$)`,
		`(^|[\/])__pycache__([\/]|$)`,
		`(^|[\/])dist([\/]|$)`,
		`(^|[\/])bin([\/]|$)`,
	}

	for _, e := range blackListedRegex {
		exp := regexp.MustCompile(e)
		if exp.MatchString(filename) {
			return filename, false
		}
	}

	allowedExtensions := []string{
		".c",
		".cpp",
		".css",
		".go",
		".html",
		".java",
		".js",
		".php",
		".pkl",
		".py",
		".rb",
		".tar",
		".tex",
		".ts",
		".sh",
		".bash",
		".zsh",
		".ps1",
	}

	renames := map[string]string{
		".vue": ".vue.txt",
		".jsx": ".js",
		".tsx": ".tx",
	}

	fileType := filepath.Ext(filename)
	mapping, ok := renames[fileType]
	if ok {
		mapped := strings.Replace(filename, fileType, mapping, 1)
		return mapped, true
	}

	for _, ext := range allowedExtensions {
		if fileType == ext {
			return filename, true
		}

	}

	return filename, false
}

// getFilesToUpload reads all files in the specified directory and returns a slice of io.Reader
// containing the contents of each file.
//
// Parameters:
//   - path: The directory path where the files are located.
//
// Returns:
//   - []io.Reader: A slice of io.Reader containing the contents of each file.
//
// Note:
//   - If there is an error opening or reading a file, the function will silently ignore the error
//     and continue processing the next file.
func getFilesToUpload(path string) (map[string]io.Reader, error) {
	files := map[string]io.Reader{}

	err := filepath.WalkDir(path, func(f string, d os.DirEntry, e error) error {
		// if entry is a directory, skip
		if d.IsDir() {
			return nil
		}
		// if entry is not in the allowed file types, skip
		mappedFilename, allowed := isAllowedFile(f)
		if !allowed {
			return nil
		}
		log.Debug(fmt.Sprintf("adding file %s", f))

		file, err := os.OpenFile(f, os.O_RDONLY, 0644)
		if err != nil {
			log.Warn(fmt.Sprintf("error opening file %s: %+v", f, err))
			return nil
		}
		defer file.Close()

		content, _ := io.ReadAll(file)
		buffer := bytes.NewBuffer(content)
		files[mappedFilename] = buffer
		return nil
	})

	return files, err
}

// combineFiles takes a map of filenames to io.Reader objects and combines their contents
// into a single io.Reader. Each file's content is prefixed with a header containing the
// filename. The combined content is separated by two newlines.
//
// Parameters:
//   - files: A map where the key is the filename (string) and the value is an io.Reader
//     containing the file's content.
//
// Returns:
//   - An io.Reader containing the combined content of all files, with each file's content
//     prefixed by a header with the filename and separated by two newlines.
func combineFiles(files map[string]io.Reader) io.Reader {
	var combinedFiles bytes.Buffer

	for path, content := range files {
		combinedFiles.WriteString(fmt.Sprintf("### FILE START %s\n\n", path))
		if _, err := combinedFiles.ReadFrom(content); err != nil {
			log.Warn(fmt.Sprintf("error reading content from file %s: %+v", path, err))
			return &combinedFiles
		}

		combinedFiles.WriteString(fmt.Sprintf("\n\n### FILE END %s\n\n", path))
	}

	return &combinedFiles
}

// groupFilesByExtension groups a map of file names and their corresponding io.Reader content
// by their file extensions. It returns a nested map where the keys are file extensions and
// the values are maps of file names and their io.Reader content.
//
// Parameters:
//   - files: A map where the keys are file names and the values are io.Reader instances
//     representing the content of the files.
//
// Returns:
//   - A nested map where the keys are file extensions (including the dot, e.g., ".txt")
//     and the values are maps of file names and their corresponding io.Reader content.
func groupFilesByExtension(files map[string]io.Reader) map[string]map[string]io.Reader {
	groupedFiles := make(map[string]map[string]io.Reader)

	for filename, content := range files {
		ext := filepath.Ext(filename)
		if _, ok := groupedFiles[ext]; !ok {
			groupedFiles[ext] = make(map[string]io.Reader)
		}
		groupedFiles[ext][filename] = content
	}
	return groupedFiles
}
