# Documentation

### Introduction

`goreadme` is a CLI tool that generates README documentation for source code using the ChatGPT assistants API.

#### How It Works

`goreadme` first combines all of the project files with a given type extension present in the specified target directory, and sends the resulting file batches to ChatGPT for analysis. Files need to be combined as ChatGPT assistants only allow a maximum if 10 attachments per thread, and most code bases typically have more than 10 files. All of the files are combined into a single thread execution along with a message to ChatGPT to perform the codebase analysis. The resulting thread run is then monitored until complete. Once complete, the resulting README is downloaded from the assistant, and saved to the local project.

### Installation

To use the CLI, either clone the repository and build locally using

```bash
$ git clone https://github.com/alpinesoftwareltd/goreadme .
$ go build . -o <output-path>
```

or download one of the binaries provided in the `bin` directory.

### Usage

`goreadme` has 3 main commands

1. `goreadme configure` - provide configuration settings to access ChatGPT services
2. `goreadme test` - test access and connection using provided configuration settings
3. `goreademe generate` - generate new README documentation

Note that configuration __must__ be done before any READMEs can be generated.

#### Configuration

In order run the CLI, a number of configuration settings need to be specified including

* ChatGPT access token
* ChatGPT model
* ChatGPT vector store ID (optional)
* ChatGPT assistant ID (optional)

All of the fields that are labeled as optional do not need to be provided, and can be generated at configuration time. `goreadme` maintains a JSON file containing all the required configuration settings. By default, this is kept at `~/.goreadme/config.json`. The easiest way to configure the CLI correctly is to run

```bash
$ goreadme configure
```

This will take you through an interactive prompt that will collect and verify all the required items via the terminal.

__IMPORTANT__: `goreadme` requires a ChatGPT vector store and assistant to work. Both will automatically be created by `goreadme configure` if the respective prompts are left blank. If you provide a custom vector store ID or assistant ID, `goreadme` will validate the provided ID using the ChatGPT API.

Alternatively, you can provide a prepared JSON file that contains all the required settings. The JSON file __must__ have the following structure

```json
{
    "accessToken": "TestToken",
    "modelVersion": "test-model",
    "vectorStoreId": "vectorstore_test-id",
    "assistantId": "assistant_test-id"
}
```

#### Testing Configuration Settings

To test all provided configuration settings, run

```bash
$ goreadme test
```

This will load the config and validate all settings using the ChatGPT API.

#### Generating READMEs

Once configured, you can generate READMEs using

```bash
$ goreadme generate <path-to-source-code>
```

Note that `path-to-source-code` must be the path to a __directory__, not a file. To generate a readme for the current directory, use

```bash
$ goreadme generate .
```

### Global Arguments

There are a number of global configuration flags that can be used with all commands

* `--log-level` -  set to `DEBUG` for detailed logging, including what requests are made and what the response codes are. This useful when debugging issues.
* `--config-path` - required if using a custom configuration path.
