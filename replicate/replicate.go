package replicate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type Client struct {
	apiKey       string
	systemPrompt string
}

type ImageResponse struct {
	ImageURL string
}

func New(apiKey, systemPrompt string) *Client {
	return &Client{apiKey: apiKey, systemPrompt: systemPrompt}
}

func (c Client) GenImage(prompt string, model string) (ImageResponse, error) {
	return genImage(c.apiKey, prompt, model)
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
