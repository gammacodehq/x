package gen

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"log"
)

type Response struct {
	Content          string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type Message struct {
	Content string
	Role    string
}


type Client struct {
	apiKey string
	systemPrompt string
}

func New(apiKey, systemPrompt string) *Client {
	return &Client{apiKey: apiKey, systemPrompt: systemPrompt}
}

func (c Client) Gen(model string, messages []Message) (Response, error) {
	return gen(c.apiKey, c.systemPrompt, model, messages)
}

// Gen generates a response for the given prompt
func gen(apiKey string, systemPrompt string, model string, messages []Message) (Response, error) {
	// Initialize default response
	respDummy := Response{}

	// Handle local mode
	if apiKey == "" {
		respDummy.Content = "Local mode response"
		return respDummy, nil
	}

	// Map messages to API format
	mappedMessages := make([]map[string]string, 0, len(messages))

	if len(messages) > 1 && messages[0].Role != "system" {
		mappedMessages = append(mappedMessages, map[string]string{
			"role": "system",
			"content": systemPrompt,
		})
	}
	for _, msg := range messages {
		mappedMessages = append(mappedMessages, map[string]string{
			"role": msg.Role,
			"content": msg.Content,
		})
	}


	// Prepare the request body
	requestBody, err := json.Marshal(map[string]interface{}{
		"model":      model,
		"max_tokens": 5120,
		"usage": map[string]bool{
			"include": true,
		},
		"messages": mappedMessages,
	})
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
				Content string `json:"content"`
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
	respDummy.PromptTokens = response.Usage.PromptTokens
	respDummy.CompletionTokens = response.Usage.CompletionTokens
	respDummy.TotalTokens = response.Usage.TotalTokens

	return respDummy, nil
}
