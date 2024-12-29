package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/briandowns/spinner"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
)

const (
	Query = `Please generate a README for the attached source code. All of the files for a
given file extension have been combined into a single file called combined_source_files.[ext]
where ext is the file extension. The combined file is organized into a set of file blocks,
where each block starts with

### FILE START [filepath]

and ends with

### FILE END [filepath]

where [filepath] gives the path of the original source code file. Treat the code within
each file block as a separate file for the purposes of the README.

Please do not include any references to the combined_source_files.[ext] file containing the
combined source code. Only reference the original source code files using the file names provided.
Ensure that context is provided that explains the purpose of the code and how it can be used
where possible.`
)

func ConfigureCLICommand(ctx context.Context, cmd *cli.Command) error {
	// configure logging for application
	configureLogging(cmd.String("log-level"))
	reader := bufio.NewReader(os.Stdin)

	var client *ChatGPTAssistantClient
	// prompt user for ChatGPT access token
	token, err := getCliInput(reader, "Enter ChatGPT access token: ", func(value string) (string, error) {
		credentials := ChatGPTCredentials{
			Secret: value,
		}
		client = NewChatGPTAssistantClient("", credentials)

		// verify provided credentials using client
		if err := client.VerifyCredentials(); err != nil {
			log.Debug(fmt.Sprintf("error validating chatgpt token: %+v", err))
			return "", err
		} else {
			return value, nil
		}
	})

	if err != nil {
		return cli.Exit("error validating chatgpt access token", 1)
	}

	// get model version from CLI and validate by making request to ChatGPT
	// api to get model details using specified ID
	model, err := getCliInput(reader, "Enter ChatGPT model version (default gpt-4o-mini): ", func(value string) (string, error) {
		if len(value) == 0 {
			value = "gpt-4o-mini"
		}

		if _, err := client.GetModel(value); err != nil {
			log.Debug(fmt.Sprintf("error validating chatgpt model: %+v", err))
			return "", err
		} else {
			return value, nil
		}
	})

	if err != nil {
		return cli.Exit("error validating chatgpt model", 1)
	}

	client.Model = model

	// get vector store ID from CLI and validate by making request to ChatGPT
	// api to get vector store details using specified ID. if no ID is provided,
	// create a new vector store and use the generated ID
	vectorStoreId, err := getCliInput(reader, "Enter ChatGPT vector store ID (leave empty to create vector store): ", func(value string) (string, error) {
		if len(value) == 0 {
			id, err := client.CreateVectorStore("goreadme")
			if err != nil {
				log.Debug(fmt.Sprintf("error creating chatgpt vector store: %+v", err))
				chatGPTError := err.(ChatGPTError)
				log.Debug(fmt.Sprintf("error response: %+v", chatGPTError.Body))
				return "", err
			}
			return id, nil
		}

		if _, err := client.GetVectorStore(value); err != nil {
			log.Debug(fmt.Sprintf("error validating chatgpt vector store: %+v", err))
			return "", err
		} else {
			return value, nil
		}
	})

	if err != nil {
		return cli.Exit("error creating/validating vector store", 1)
	}

	// get assistant ID from CLI and validate by making request to ChatGPT
	// api to get assistant details using specified ID. if no ID is provided,
	// create a new assistant and use the generated ID
	assistantId, err := getCliInput(reader, "Enter ChatGPT assistant ID (leave empty to create assistant): ", func(value string) (string, error) {
		if len(value) == 0 {
			description := "You are an assistant for auto-generating READMEs and associated documentation."
			id, err := client.CreateAssistant("goreadme", description, model, vectorStoreId)
			if err != nil {
				log.Debug(fmt.Sprintf("error creating chatgpt assistant: %+v", err))
				chatGPTError := err.(ChatGPTError)
				log.Debug(fmt.Sprintf("error response: %+v", chatGPTError.Body))
				return "", err
			}
			return id, nil
		}

		assistant, err := client.GetAssistant(value)
		if err != nil {
			log.Debug(fmt.Sprintf("error validating chatgpt assistant: %+v", err))
			return "", err
		}

		if assistant.ToolResources.FileSearch.VectorStoreIds == nil {
			log.Debug("vector store ids not found in assistant tool resources")
			return "", fmt.Errorf("vector store ids not found in assistant tool resources")
		}

		if len(assistant.ToolResources.FileSearch.VectorStoreIds) == 0 {
			log.Debug("vector store ids not found in assistant tool resources")
			return "", fmt.Errorf("vector store ids not found in assistant tool resources")
		}

		return value, nil
	})

	if err != nil {
		return cli.Exit("error creating/validating assistant", 1)
	}

	defaultConfigPath := getDefaultConfigPath()
	prompt := fmt.Sprintf("Enter config path (default %s): ", defaultConfigPath)
	// read token from input and remove trailing line break. if no
	// path is provided, use default
	path, _ := getCliInput(reader, prompt, func(value string) (string, error) {
		return value, nil
	})

	if len(path) == 0 {
		// get home directory and generate path
		path = defaultConfigPath
	}

	config := Config{
		AccessToken:   token,
		ModelVersion:  model,
		VectorStoreId: vectorStoreId,
		AssistantId:   assistantId,
	}

	if err := writeConfig(config, path); err != nil {
		log.Debug(fmt.Sprintf("%+v", err))
		return cli.Exit(fmt.Sprintf("error writing config file to %s", path), 1)
	}

	return nil
}

// TestCLICommand is a function that tests a CLI command by loading and applying a configuration.
// It takes a context and a CLI command as parameters and returns an error if the configuration
// cannot be loaded.
//
// Parameters:
//   - ctx: The context in which the command is executed.
//   - cmd: The CLI command to be tested.
//
// Returns:
//   - error: An error if the configuration cannot be loaded, otherwise nil.
func TestCLICommand(ctx context.Context, cmd *cli.Command) error {
	// configure logging for application
	configureLogging(cmd.String("log-level"))

	cfgPath := cmd.String("config-path")
	log.Debug(fmt.Sprintf("loading new configuration from path %s", cfgPath))

	config, err := loadConfig(cfgPath)
	if err != nil {
		return cli.Exit("error loading config file", 1)
	}
	log.Debug(fmt.Sprintf("loaded configuration %+v", config))

	client := NewChatGPTAssistantClient(config.ModelVersion, ChatGPTCredentials{
		Secret: config.AccessToken,
	})

	if err := client.VerifyCredentials(); err != nil {
		log.Debug(fmt.Sprintf("error verifying chatgpt credentials: %+v", err))
		return cli.Exit("error validating chatgpt credentials", 1)
	}

	_, err = client.GetModel(config.ModelVersion)
	if err != nil {
		log.Debug(fmt.Sprintf("error fetching model %s from chatgpt api: %+v", config.ModelVersion, err))
		return cli.Exit("error validating chatgpt model", 1)
	}

	_, err = client.GetVectorStore(config.VectorStoreId)
	if err != nil {
		log.Debug(fmt.Sprintf("error fetching vector store %s from chatgpt api: %+v", config.VectorStoreId, err))
		return cli.Exit("error validating chatgpt vector store", 1)
	}

	_, err = client.GetAssistant(config.AssistantId)
	if err != nil {
		log.Debug(fmt.Sprintf("error fetching assistant %s from chatgpt api: %+v", config.AssistantId, err))
		return cli.Exit("error validating chatgpt assistant", 1)
	}

	return nil
}

// GenerateCLICommand is a CLI command handler that generates a new README file for a specified target directory.
// It performs the following steps:
// 1. Configures logging based on the provided log level.
// 2. Loads the configuration from the specified config path.
// 3. Validates that the target directory exists and is a valid directory.
//
// Parameters:
// - ctx: The context for the command execution.
// - cmd: The CLI command containing the arguments and flags.
//
// Returns:
// - An error if any step fails, otherwise nil.
func GenerateCLICommand(ctx context.Context, cmd *cli.Command) error {
	spinner := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	spinner.Prefix = "Loading configuration file "
	spinner.Start()

	// configure logging for application
	configureLogging(cmd.String("log-level"))

	cfgPath := cmd.String("config-path")
	log.Debug(fmt.Sprintf("loading new configuration from path %s", cfgPath))

	config, err := loadConfig(cfgPath)
	if err != nil {
		return cli.Exit("error loading config file", 1)
	}
	log.Debug(fmt.Sprintf("loaded configuration %+v", config))

	target := cmd.String("target")
	log.Debug(fmt.Sprintf("generating new README for target dir %s", target))

	spinner.Prefix = "Validating target directory "
	// check that provided path is a valid directory
	if !isValidDir(target) {
		return cli.Exit(fmt.Sprintf("path %s either does not exist or is not a valid directory", target), 1)
	}

	spinner.Prefix = "Checking CLI inputs and config settings "

	// get all files that need to be uploaded and group
	// by file extension/type.
	files, err := getFilesToUpload(target)
	if err != nil {
		log.Debug(fmt.Sprintf("error reading source code files: %+v", err))
		return cli.Exit("error generating README", 1)
	}

	log.Debug(fmt.Sprintf("found %d files to upload", len(files)))
	grouped := groupFilesByExtension(files)

	toUpload := map[string]io.Reader{}
	// combine all files of the same type into a single file
	log.Debug(fmt.Sprintf("found %d unique file extensions", len(grouped)))
	for ext, files := range grouped {
		combined := combineFiles(files)
		log.Debug(fmt.Sprintf("combined %d files of type %s", len(files), ext))
		filename := "combined_source_files" + ext
		toUpload[filename] = combined
	}

	spinner.Prefix = fmt.Sprintf("Analyzing %d files", len(toUpload))
	// upload files to ChatGPT assistant
	client := NewChatGPTAssistantClient(config.ModelVersion, ChatGPTCredentials{
		Secret: config.AccessToken,
	})

	spinner.Prefix = fmt.Sprintf("Uploading %d files to ChatGPT assistant", len(toUpload))
	fileIds, errors := uploadFiles(client, toUpload)

	if len(errors) > 0 {
		for _, e := range errors {
			log.Debug(fmt.Sprintf("error uploading file: %+v", e))
			chatGPTError := e.(ChatGPTError)
			log.Debug(fmt.Sprintf("error response: %+v", chatGPTError.Body))
		}
		log.Debug(fmt.Sprintf("found %d errors during file upload", len(errors)))
		return cli.Exit("error generating README", 1)
	}

	attachments := []FileAttachment{}
	for _, id := range fileIds {
		attachments = append(attachments, FileAttachment{
			FileId: id,
			Tools: []Tool{
				{
					Type: "file_search",
				},
			},
		})
	}

	messages := []ThreadMessage{
		{
			Role:        "user",
			Content:     Query,
			Attachments: attachments,
		},
	}

	spinner.Prefix = "Generating README using ChatGPT assistant "
	run, err := client.CreateThreadAndRun(config.AssistantId, config.VectorStoreId, messages)
	if err != nil {
		log.Debug(fmt.Sprintf("error creating thread and run: %+v", err))
		chatGPTError := err.(ChatGPTError)
		log.Debug(fmt.Sprintf("error creating thread: %+v", chatGPTError.Body))
		return cli.Exit("error generating README", 1)
	}

	result, err := client.WaitForRunCompletion(run.ThreadId, run.Id)
	if err != nil {
		log.Debug(fmt.Sprintf("error waiting for run completion: %+v", err))
		return cli.Exit("error generating README", 1)
	} else if result.Status != "completed" {
		log.Debug(fmt.Sprintf("run status is %s", result.Status))
		return cli.Exit("error generating README", 1)
	}

	spinner.Prefix = "Downloading README content from ChatGPT assistant "
	threadMessages, err := client.GetThreadMessages(run.ThreadId)
	if err != nil {
		log.Debug(fmt.Sprintf("error retrieving messages: %+v", err))
		chatGPTError := err.(ChatGPTError)
		log.Debug(fmt.Sprintf("error creating thread: %+v", chatGPTError.Body))
		return cli.Exit("error generating README", 1)
	}

	content := threadMessages[0].Content[0].Text.Value
	output := filepath.Join(target, "README.md")

	spinner.Prefix = "Writing README content to file "
	file, err := os.Create(output)
	if err != nil {
		log.Debug(fmt.Sprintf("error opening file %s: %+v", output, err))
		return cli.Exit("error generating README", 1)
	}
	defer file.Close()

	if _, err := file.WriteString(content); err != nil {
		log.Debug(fmt.Sprintf("error writing file content: %+v", err))
		return cli.Exit("error generating README", 1)
	}
	return nil
}
