package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	APIUrl = "https://api.openai.com/v1"
)

func NewChatGPTError(response *http.Response) error {
	// read contents of response body
	// and parse JSON structure
	buffer, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(buffer, &payload); err != nil {
		return err
	}

	gptError := ChatGPTError{
		Code: response.StatusCode,
		Body: payload,
	}
	// add error type to error interface
	if response.StatusCode == http.StatusUnauthorized {
		gptError.Type = ChatGPTErrorTypeAuth
	} else {
		gptError.Type = ChatGPTErrorTypeAPI
	}
	return gptError
}

func NewChatGPTAssistantClient(model string, credentials ChatGPTCredentials) *ChatGPTAssistantClient {
	return &ChatGPTAssistantClient{
		Model:       model,
		Credentials: credentials,
		Client:      &http.Client{},
	}
}

type ChatGPTService interface {
	VerifyCredentials() error
	GetAssistant(id string) (Assistant, error)
	CreateAssistant(name, description, vectorStoreId string) (string, error)
	GetVectorStore(id string) (VectorStore, error)
	GetModel(model string) (Model, error)
	CreateVectorStore(name string) (string, error)
	CreateThread(files []io.Reader) (string, error)
	RunThread(threadId string) (string, error)
	UploadFile(file io.Reader) (string, error)
	GetThreadMessages(threadId string) ([]ThreadMessageContent, error)
	WaitForRunCompletion(runId string) (ThreadRun, error)
}

type ChatGPTAssistantClient struct {
	Credentials ChatGPTCredentials
	Model       string
	*http.Client
}

// ExecuteChatGPTRequest sends an HTTP request to the specified URL using the provided method and payload.
// It sets the necessary headers for authorization and content type.
//
// Parameters:
//   - method: The HTTP method to use for the request (e.g., "GET", "POST").
//   - url: The URL to which the request is sent.
//   - payload: The data to be sent in the request body. It can be of any type.
//
// Returns:
//   - *http.Response: The HTTP response received from the server.
//   - error: An error if the request could not be created or executed.
func (client *ChatGPTAssistantClient) ExecuteChatGPTRequest(method, url string, payload any, headers map[string]string) (*http.Response, error) {
	var buffer io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		buffer = bytes.NewBuffer(encoded)
	}

	request, err := http.NewRequest(method, url, buffer)
	if err != nil {
		return nil, err
	}
	// add required request headers
	request.Header.Add("Authorization", "Bearer "+client.Credentials.Secret)
	request.Header.Add("Content-Type", "application/json")

	for k, v := range headers {
		request.Header.Add(k, v)
	}

	r, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	log.Debug(fmt.Sprintf("received http(s) response: %s %s - %d", method, url, r.StatusCode))

	return r, nil
}

// VerifyCredentials checks the validity of the client's credentials by making a request
// to the /models endpoint of the ChatGPT API. If the credentials are valid, the function
// returns nil. Otherwise, it returns an error indicating the failure reason.
func (client *ChatGPTAssistantClient) VerifyCredentials() error {
	// check credentials using /models endpoint
	response, err := client.ExecuteChatGPTRequest(http.MethodGet, APIUrl+"/models", nil, nil)
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK {
		return NewChatGPTError(response)
	}
	return nil
}

// GetAssistant retrieves an assistant by its ID from the ChatGPT API.
// It sends a GET request to the API endpoint with the specified assistant ID
// and includes necessary headers.
//
// Parameters:
//   - id: The unique identifier of the assistant to retrieve.
//
// Returns:
//   - Assistant: The assistant object retrieved from the API.
//   - error: An error object if the request fails or the response cannot be parsed.
func (client *ChatGPTAssistantClient) GetAssistant(id string) (Assistant, error) {
	var assistant Assistant

	headers := map[string]string{
		"OpenAI-Beta": "assistants=v2",
	}

	response, err := client.ExecuteChatGPTRequest(http.MethodGet, APIUrl+"/assistants/"+id, nil, headers)
	if err != nil {
		return assistant, err
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
		data, err := io.ReadAll(response.Body)
		if err != nil {
			return assistant, err
		}

		var assistant Assistant
		if err := json.Unmarshal(data, &assistant); err != nil {
			return assistant, err
		} else {
			return assistant, nil
		}

	default:
		return assistant, NewChatGPTError(response)
	}
}

// GetVectorStore retrieves a VectorStore by its ID from the ChatGPT API.
// It sends a GET request to the API endpoint with the provided ID and returns the VectorStore object.
//
// Parameters:
//   - id: The ID of the VectorStore to retrieve.
//
// Returns:
//   - VectorStore: The retrieved VectorStore object.
//   - error: An error if the request fails or the response cannot be parsed.
//
// The function sets a custom header "OpenAI-Beta" with the value "assistants=v2" for the request.
// It handles the response by checking the status code and unmarshaling the JSON response body into a VectorStore object.
// If the status code is not 200 OK, it returns a ChatGPTError.
func (client *ChatGPTAssistantClient) GetVectorStore(id string) (VectorStore, error) {
	var vectorStore VectorStore

	headers := map[string]string{
		"OpenAI-Beta": "assistants=v2",
	}

	response, err := client.ExecuteChatGPTRequest(http.MethodGet, APIUrl+"/vector_stores/"+id, nil, headers)
	if err != nil {
		return vectorStore, err
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
		data, err := io.ReadAll(response.Body)
		if err != nil {
			return vectorStore, err
		}
		if err := json.Unmarshal(data, &vectorStore); err != nil {
			return vectorStore, err
		} else {
			return vectorStore, nil
		}
	default:
		return vectorStore, NewChatGPTError(response)
	}
}

// GetModel retrieves the details of a specific model from the ChatGPT API.
// It takes the model name as a parameter and returns the Model data and an error, if any.
//
// Parameters:
//   - model: The name of the model to retrieve.
//
// Returns:
//   - Model: The details of the requested model.
//   - error: An error if the request fails or the response cannot be parsed.
func (client *ChatGPTAssistantClient) GetModel(model string) (Model, error) {
	var modelData Model

	url := fmt.Sprintf("%s/models/%s", APIUrl, model)
	response, err := client.ExecuteChatGPTRequest(http.MethodGet, url, nil, nil)
	if err != nil {
		return modelData, err
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
		data, err := io.ReadAll(response.Body)
		if err != nil {
			return modelData, err
		}

		if err := json.Unmarshal(data, &modelData); err != nil {
			return modelData, err
		} else {
			return modelData, nil
		}

	default:
		return modelData, NewChatGPTError(response)
	}
}

// CreateAssistant creates a new assistant with the specified name, description, model, and vector store ID.
// It sends a POST request to the ChatGPT API to create the assistant and returns the assistant's ID if successful.
//
// Parameters:
//   - name: The name of the assistant.
//   - description: A brief description of the assistant.
//   - model: The model to be used by the assistant.
//   - vectorStoreId: The ID of the vector store to be used for file search.
//
// Returns:
//   - string: The ID of the created assistant.
//   - error: An error if the request fails or the response cannot be parsed.
func (client *ChatGPTAssistantClient) CreateAssistant(name, description, model, vectorStoreId string) (string, error) {
	// generate new JSON payload
	payload := map[string]interface{}{
		"model":       model,
		"name":        name,
		"description": description,
		"tools": []map[string]string{
			{
				"type": "file_search",
			},
		},
		"tool_resources": map[string]interface{}{
			"file_search": map[string]interface{}{
				"vector_store_ids": []string{
					vectorStoreId,
				},
			},
		},
	}

	headers := map[string]string{
		"OpenAI-Beta": "assistants=v2",
	}

	response, err := client.ExecuteChatGPTRequest(http.MethodPost, APIUrl+"/assistants", payload, headers)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
		// read contents of response body
		// and parse JSON structure
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return "", err
		}

		var data struct {
			Id string `json:"id"`
		}
		if err := json.Unmarshal(body, &data); err != nil {
			return "", err
		}
		return data.Id, nil

	default:
		return "", NewChatGPTError(response)
	}
}

// CreateVectorStore creates a new vector store with the given name.
// It sends a POST request to the ChatGPT API to create the vector store.
//
// Parameters:
//   - name: The name of the vector store to be created.
//
// Returns:
//   - string: The ID of the created vector store.
//   - error: An error if the request fails or the response cannot be parsed.
//
// The function generates a JSON payload with the provided name and sets the necessary headers.
// It then executes the request and handles the response. If the request is successful, it returns
// the ID of the created vector store. Otherwise, it returns an error.
func (client *ChatGPTAssistantClient) CreateVectorStore(name string) (string, error) {
	// generate new JSON payload
	payload := map[string]interface{}{
		"name": name,
	}
	headers := map[string]string{
		"OpenAI-Beta": "assistants=v2",
	}

	response, err := client.ExecuteChatGPTRequest(http.MethodPost, APIUrl+"/vector_stores", payload, headers)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
		// read contents of response body
		// and parse JSON structure
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return "", err
		}

		var data struct {
			Id string `json:"id"`
		}
		if err := json.Unmarshal(body, &data); err != nil {
			return "", err
		}
		return data.Id, nil

	default:
		return "", NewChatGPTError(response)
	}
}

// CreateThreadAndRun creates a new thread with the given assistant ID and vector store ID,
// and runs it with the provided messages. It returns the created Thread and an error, if any.
//
// Parameters:
//   - assistantId: The ID of the assistant to be used for creating the thread.
//   - vectorStoreId: The ID of the vector store to be used for file search within the thread.
//   - messages: A slice of ThreadMessage representing the messages to be included in the thread.
//
// Returns:
//   - Thread: The created thread.
//   - error: An error object if there was an issue creating or running the thread.
func (client *ChatGPTAssistantClient) CreateThreadAndRun(assistantId, vectorStoreId string, messages []ThreadMessage) (ThreadRun, error) {
	var run ThreadRun

	payload := map[string]interface{}{
		"assistant_id": assistantId,
		"thread": map[string]interface{}{
			"messages": messages,
			"tool_resources": map[string]interface{}{
				"file_search": map[string]interface{}{
					"vector_store_ids": []string{vectorStoreId},
				},
			},
		},
	}

	headers := map[string]string{
		"OpenAI-Beta": "assistants=v2",
	}

	response, err := client.ExecuteChatGPTRequest(http.MethodPost, APIUrl+"/threads/runs", payload, headers)
	if err != nil {
		return run, err
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
		content, err := io.ReadAll(response.Body)
		if err != nil {
			return run, err
		}
		if err := json.Unmarshal(content, &run); err != nil {
			return run, err
		} else {
			return run, nil
		}

	default:
		return run, NewChatGPTError(response)
	}
}

func (client *ChatGPTAssistantClient) UploadFile(filename string, content io.Reader) (string, error) {

	var data bytes.Buffer
	writer := multipart.NewWriter(&data)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", err
	}
	// write file content to the form
	if _, err := io.Copy(part, content); err != nil {
		return "", err
	}
	// add purpose field to the form
	if err := writer.WriteField("purpose", "assistants"); err != nil {
		return "", err
	}
	// close the writer to finalize the form
	if err := writer.Close(); err != nil {
		return "", err
	}

	request, err := http.NewRequest(http.MethodPost, APIUrl+"/files", &data)
	if err != nil {
		return "", err
	}
	// add required request headers
	request.Header.Add("Authorization", "Bearer "+client.Credentials.Secret)
	request.Header.Add("Content-Type", writer.FormDataContentType())

	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	log.Debug(fmt.Sprintf("received http(s) response: POST %s - %d", APIUrl+"/files", response.StatusCode))

	switch response.StatusCode {
	case http.StatusOK:
		content, err := io.ReadAll(response.Body)
		if err != nil {
			return "", err
		}

		var payload struct {
			Id string `json:"id"`
		}
		if err := json.Unmarshal(content, &payload); err != nil {
			return "", err
		} else {
			return payload.Id, nil
		}

	default:
		return "", NewChatGPTError(response)
	}
}

// WaitForRunCompletion waits for the completion of a thread run with the given runId.
// It continuously polls the ChatGPT API until the run status is "completed", "cancelled", or "failed".
// The function returns the final ThreadRun object or an error if the request fails.
//
// Parameters:
//   - runId: The ID of the thread run to wait for.
//
// Returns:
//   - ThreadRun: The final state of the thread run.
//   - error: An error if the request fails or if the response cannot be parsed.
func (client *ChatGPTAssistantClient) WaitForRunCompletion(threadId, runId string) (ThreadRun, error) {
	var run ThreadRun

	headers := map[string]string{
		"OpenAI-Beta": "assistants=v2",
	}

	for {
		url := fmt.Sprintf("%s/threads/%s/runs/%s", APIUrl, threadId, runId)
		response, err := client.ExecuteChatGPTRequest(http.MethodGet, url, nil, headers)
		if err != nil {
			return run, err
		}
		defer response.Body.Close()

		switch response.StatusCode {
		case http.StatusOK:
			content, err := io.ReadAll(response.Body)
			if err != nil {
				return run, err
			}

			if err := json.Unmarshal(content, &run); err != nil {
				return run, err
			}

		default:
			return run, NewChatGPTError(response)
		}

		switch run.Status {
		case "completed", "cancelled", "failed":
			return run, nil
		}

		time.Sleep(time.Second * 3)
	}
}

func (client *ChatGPTAssistantClient) GetThreadMessages(threadId string) ([]ThreadMessageResponse, error) {

	headers := map[string]string{
		"OpenAI-Beta": "assistants=v2",
	}

	url := fmt.Sprintf("%s/threads/%s/messages", APIUrl, threadId)
	response, err := client.ExecuteChatGPTRequest(http.MethodGet, url, nil, headers)
	if err != nil {
		return []ThreadMessageResponse{}, err
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
		content, err := io.ReadAll(response.Body)
		if err != nil {
			return []ThreadMessageResponse{}, err
		}

		var payload struct {
			Data []ThreadMessageResponse `json:"data"`
		}
		if err := json.Unmarshal(content, &payload); err != nil {
			return payload.Data, err
		} else {
			return payload.Data, nil
		}

	default:
		return []ThreadMessageResponse{}, NewChatGPTError(response)
	}
}
