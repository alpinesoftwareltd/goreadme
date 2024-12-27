package main

import (
	"errors"
	"os"
	"testing"
)

// TestLoadConfig tests the loadConfig function to ensure it correctly loads
// configuration from a JSON file and populates the Config struct fields.
// It checks for the following:
// - AccessToken should be "TestToken"
// - ModelVersion should be "test-model"
// - AssistantId should be "assistant_test-id"
// - VectorStoreId should be "vectorstore_test-id"
// If any of these conditions are not met, the test will fail.
func TestLoadConfig(t *testing.T) {
	config, err := loadConfig("tests/config.json")
	if err != nil {
		t.Fatal(err)
	}

	if config.AccessToken != "TestToken" {
		t.Fatalf("expected access token %s, got %s", "TestToken", config.AccessToken)
	}

	if config.ModelVersion != "test-model" {
		t.Fatalf("expected model version %s, got %s", "test-model", config.ModelVersion)
	}

	if config.AssistantId != "assistant_test-id" {
		t.Fatalf("expected assistant id %s, got %s", "assistant_test-id", config.AssistantId)
	}

	if config.VectorStoreId != "vectorstore_test-id" {
		t.Fatalf("expected vector store id  %s, got %s", "vectorstore_test-id", config.VectorStoreId)
	}
}

// TestLoadConfigPartial tests the loadConfig function with a partial configuration file.
// It expects an error to be returned when loading a partial config file.
// The test checks if the error is of type InvalidConfigFileError.
func TestLoadConfigPartial(t *testing.T) {
	_, err := loadConfig("tests/partial_config.json")
	if err == nil {
		t.Fatal("expected error while loading partial config")
	}

	var invalidConfigErr InvalidConfigFileError
	if !errors.As(err, &invalidConfigErr) {
		t.Fatalf("expected InvalidConfigFileError, got %+v", err)
	}
}

// TestLoadConfigNotFound tests the loadConfig function to ensure it returns an error
// when attempting to load a configuration file that does not exist. It verifies that
// the error returned is of type ConfigFileNotFoundError.
func TestLoadConfigNotFound(t *testing.T) {
	_, err := loadConfig("tests/not_found_config.json")
	if err == nil {
		t.Fatal("expected error while loading not found config")
	}

	var configNotFound ConfigFileNotFoundError
	if !errors.As(err, &configNotFound) {
		t.Fatalf("expected ConfigFileNotFoundError, got %+v", err)
	}
}

// TestWriteConfig tests the functionality of loading a configuration file,
// updating a specific field, writing the updated configuration back to a file,
// and verifying that the changes were correctly saved. It also ensures that
// the temporary updated configuration file is deleted after the test.
func TestWriteConfig(t *testing.T) {
	config, err := loadConfig("tests/config.json")
	if err != nil {
		t.Fatalf("error loading config: %+v", err)
	}

	config.AccessToken = "test-token-updated"

	if err := writeConfig(config, "tests/config_updated.json"); err != nil {
		t.Fatalf("error writing updated config: %+v", err)
	}

	updated, err := loadConfig("tests/config_updated.json")
	if err != nil {
		t.Fatalf("error loading updated config: %+v", err)
	}

	if updated.AccessToken != "test-token-updated" {
		t.Fatalf("expected token %s in updated config, got %s", "test-token-updated", updated.AccessToken)
	}

	if err := os.Remove("tests/config_updated.json"); err != nil {
		t.Fatalf("error deleting updated config file: %+v", err)
	}
}
