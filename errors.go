package main

import "fmt"

type InvalidConfigFileError struct {
	Path string
}

func (e InvalidConfigFileError) Error() string {
	return fmt.Sprintf("error loading config file at provided path %s", e.Path)
}

type ConfigFileNotFoundError struct {
	Path string
}

func (e ConfigFileNotFoundError) Error() string {
	return fmt.Sprintf("cannot find config file at provided path %s", e.Path)
}

type ChatGPTErrorType string

const (
	ChatGPTErrorTypeAuth ChatGPTErrorType = "authentication"
	ChatGPTErrorTypeAPI  ChatGPTErrorType = "api"
)

type ChatGPTError struct {
	Code int
	Body map[string]interface{}
	Type ChatGPTErrorType
}

func (e ChatGPTError) Error() string {
	return fmt.Sprintf("received ChatGPT error type %s: status code %d", e.Type, e.Code)
}
