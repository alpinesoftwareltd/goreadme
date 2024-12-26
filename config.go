package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-playground/validator/v10"
	log "github.com/sirupsen/logrus"
)

// loadConfig loads the configuration from the specified file path.
// It returns a Config struct and an error if any issues are encountered.
//
// Parameters:
//   - path: The file path to the configuration file.
//
// Returns:
//   - Config: The loaded configuration struct.
//   - error: An error if the configuration file is not found, is a directory,
//     cannot be read, is invalid JSON, or fails validation.
//
// Possible errors:
//   - ConfigFileNotFoundError: If the configuration file does not exist.
//   - InvalidConfigFileError: If the path is a directory, the file cannot be read,
//     the JSON is invalid, or the configuration fails validation.
func loadConfig(path string) (Config, error) {
	var config Config

	stat, err := os.Stat(path)
	if err != nil {
		log.Debug(fmt.Sprintf("cannot find config file at path %s: %+v", path, err))
		return config, ConfigFileNotFoundError{
			Path: path,
		}
	} else if stat.IsDir() {
		log.Debug(fmt.Sprintf("cannot load config %s: path is directory, expected file", path))
		return config, InvalidConfigFileError{
			Path: path,
		}
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		log.Debug(fmt.Sprintf("error reading config file: %+v", err))
		return config, InvalidConfigFileError{
			Path: path,
		}
	}

	if err := json.Unmarshal(contents, &config); err != nil {
		log.Debug(fmt.Sprintf("error decoding config file: %+v", err))
		return config, InvalidConfigFileError{
			Path: path,
		}
	}

	// validate contents of config file using validator package
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(config); err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			log.Debug(fmt.Sprintf("config validation error: %+v", err))
		}
		return config, InvalidConfigFileError{
			Path: path,
		}
	}
	return config, nil
}

// writeConfig writes the given configuration to a specified file path in JSON format.
// It ensures that the directory path exists, creating any necessary directories.
// If the directory path is invalid, it returns an error.
//
// Parameters:
//   - config: The configuration struct to be written to the file.
//   - path: The file path where the configuration should be written.
//
// Returns:
//   - error: An error if the directory path is invalid, if there is an issue creating directories,
//     if there is an error converting the struct to JSON, or if there is an error writing the file.
func writeConfig(config Config, path string) error {
	// Ensure the directory path exists
	dir := filepath.Dir(path)
	if dir == "." || dir == "/" {
		return errors.New("invalid file path or directory")
	}
	// create any directories that need to be created
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	// Convert the struct to JSON (with pretty formatting)
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	// Write the JSON to the file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}
	return nil
}
