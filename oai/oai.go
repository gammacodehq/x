package oai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type Response struct {
	Content          string
	Tools            []ToolResponse
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type Message struct {
	Content interface{} `json:"content,omitempty"`
	Role    string      `json:"role"`
}

type ResponsePart struct {
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
}

type ContentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

type ImageResponse struct {
	ImageURL string
}

type ImageURL struct {
	URL string `json:"url"`
}

type Client struct {
	apiKeyOR     string
	apiKeyRepl   string
	systemPrompt string
}

type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolResponse struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

func New(apiKeyOR, apiKeyRepl, systemPrompt string) *Client {
	return &Client{apiKeyOR: apiKeyOR, apiKeyRepl: apiKeyRepl, systemPrompt: systemPrompt}
}

func (c Client) Gen(model string, messages []Message, tools ...[]Tool) (Response, error) {
	return gen(c.apiKeyOR, c.systemPrompt, model, messages, tools...)
}

func (c Client) GenImage(prompt string, model string) (ImageResponse, error) {
	return genImage(c.apiKeyRepl, prompt, model)
}

func genImage(apiKey string, prompt string, model string) (ImageResponse, error) {
	respDummy := ImageResponse{}
	url := fmt.Sprintf("https://api.replicate.com/v1/models/%s/predictions", model)
	requestBody, err := json.Marshal(
		map[string]interface{}{
			"input": map[string]interface{}{
				"prompt": prompt,
			},
		})
	if err != nil {
		return respDummy, err
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return respDummy, err
	}

	req.Header.Set("Authorization", "Token "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "wait")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return respDummy, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return respDummy, err
	}

	var response struct {
		Status string      `json:"status"`
		Output []string    `json:"output"`
		Error  interface{} `json:"error"`
	}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return respDummy, err
	}

	if response.Status != "succeeded" {
		log.Println("generation failed", string(body))
		return respDummy, nil
	}

	if len(response.Output) == 0 {
		log.Println("No choices", string(body))
		return respDummy, nil
	}
	respDummy.ImageURL = response.Output[0]
	return respDummy, nil
}

// Gen generates a response for the given prompt
func gen(apiKey string, systemPrompt string, model string, messages []Message, tools ...[]Tool) (Response, error) {
	// Initialize default response
	respDummy := Response{}
	// Handle local mode
	if apiKey == "" {
		respDummy.Content = "Local mode response"
		return respDummy, nil
	}

	// Map messages to API format
	mappedMessages := make([]map[string]interface{}, 0, len(messages))
	if len(messages) > 1 && messages[0].Role != "system" {
		mappedMessages = append(mappedMessages, map[string]interface{}{
			"role":    "system",
			"content": systemPrompt,
		})
	}
	for _, msg := range messages {
		mappedMessages = append(mappedMessages, map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}
	// Prepare the request body
	reqMap := map[string]interface{}{
		"model":      model,
		"max_tokens": 5120,
		"usage": map[string]bool{
			"include": true,
		},
		"messages": mappedMessages,
	}
	if len(tools) > 0 && len(tools[0]) > 0 {
		reqMap["tools"] = tools[0]
	}
	requestBody, err := json.Marshal(reqMap)
	if err != nil {
		return respDummy, err
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(
		context.Background(),
		"POST",
		"https://openrouter.ai/api/v1/chat/completions",
		bytes.NewBuffer(requestBody),
	)
	if err != nil {
		return respDummy, err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return respDummy, err
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return respDummy, err
	}

	// Parse response
	var response struct {
		Choices []struct {
			Message struct {
				Content   string         `json:"content"`
				ToolCalls []ToolResponse `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return respDummy, err
	}
	if len(response.Choices) == 0 {
		log.Println("No choices", string(body))
		return respDummy, nil
	}
	respDummy.Content = response.Choices[0].Message.Content
	respDummy.Tools = response.Choices[0].Message.ToolCalls
	respDummy.PromptTokens = response.Usage.PromptTokens
	respDummy.CompletionTokens = response.Usage.CompletionTokens
	respDummy.TotalTokens = response.Usage.TotalTokens
	return respDummy, nil
}
