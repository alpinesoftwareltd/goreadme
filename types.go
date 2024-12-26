package main

type Config struct {
	AccessToken   string `json:"accessToken" validate:"required"`
	ModelVersion  string `json:"modelVersion" validate:"required"`
	AssistantId   string `json:"assistantId" validate:"required"`
	VectorStoreId string `json:"vectorStoreId" validate:"required"`
}

type ChatGPTCredentials struct {
	Secret string `json:"secret"`
}

type VectorStore struct {
	Id string `json:"id"`
}

type AssistantToolResources struct {
	FileSearch struct {
		VectorStoreIds []string `json:"vector_store_ids"`
	} `json:"file_search"`
}

type Assistant struct {
	Id            string                 `json:"id"`
	ToolResources AssistantToolResources `json:"tool_resources"`
}

type Model struct {
	Id string `json:"id"`
}

type Thread struct {
	Id string `json:"id"`
}

type Tool struct {
	Type string `json:"type"`
}

type FileAttachment struct {
	FileId string `json:"file_id"`
	Tools  []Tool `json:"tools"`
}

type ThreadMessage struct {
	Role        string           `json:"role"`
	Content     string           `json:"content"`
	Attachments []FileAttachment `json:"attachments"`
}

type ThreadMessageContent struct {
	Type string `json:"type"`
	Text struct {
		Value string `json:"value"`
	} `json:"text"`
}

type ThreadMessageResponse struct {
	Role        string                 `json:"role"`
	Content     []ThreadMessageContent `json:"content"`
	Attachments []FileAttachment       `json:"attachments"`
}

type ThreadRun struct {
	Id       string `json:"id"`
	ThreadId string `json:"thread_id"`
	Status   string `json:"status"`
}
