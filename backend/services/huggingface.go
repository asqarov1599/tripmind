package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type AIClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

var aiClient *AIClient

func InitAI() {
	model := os.Getenv("HF_MODEL")
	if model == "" {
		model = "mistralai/Mistral-7B-Instruct-v0.3"
	}

	aiClient = &AIClient{
		apiKey: os.Getenv("HUGGINGFACE_API_KEY"),
		model:  model,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}

	if aiClient.apiKey != "" {
		fmt.Println("✅ AI (HuggingFace) initialized with model:", model)
	} else {
		fmt.Println("⚠️  HUGGINGFACE_API_KEY not set — AI summaries will use fallback text")
	}
}

func GetAIClient() *AIClient {
	return aiClient
}

type hfRequest struct {
	Inputs     string       `json:"inputs"`
	Parameters hfParameters `json:"parameters"`
}

type hfParameters struct {
	MaxNewTokens   int     `json:"max_new_tokens"`
	Temperature    float64 `json:"temperature"`
	ReturnFullText bool    `json:"return_full_text"`
}

type hfResponse []struct {
	GeneratedText string `json:"generated_text"`
}

func (c *AIClient) GetRecommendations(
	budget float64,
	origin, destination, departureDate, returnDate string,
	passengers int,
	flights []Flight,
	hotels []Hotel,
	isFallbackData bool,
	returnOrigin string,
) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("huggingface API key not configured")
	}

	prompt := buildPrompt(budget, origin, destination, departureDate, returnDate, passengers, flights, hotels, isFallbackData, returnOrigin)

	reqBody := hfRequest{
		Inputs: prompt,
		Parameters: hfParameters{
			MaxNewTokens:   400,
			Temperature:    0.6,
			ReturnFullText: false,
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("https://api-inference.huggingface.co/models/%s", c.model)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 503 {
		return "", fmt.Errorf("AI model is loading, please retry in a few seconds")
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HuggingFace API error (%d): %s", resp.StatusCode, string(body))
	}

	var hfResp hfResponse
	if err := json.Unmarshal(body, &hfResp); err != nil {
		return "", fmt.Errorf("failed to parse AI response: %v", err)
	}

	if len(hfResp) == 0 || hfResp[0].GeneratedText == "" {
		return "", fmt.Errorf("empty response from AI")
	}

	return hfResp[0].GeneratedText, nil
}

func buildPrompt(
	budget float64,
	origin, destination, departureDate, returnDate string,
	passengers int,
	flights []Flight,
	hotels []Hotel,
	isFallbackData bool,
	returnOrigin string,
) string {
	dataNote := ""
	if isFallbackData {
		dataNote = " Note: prices are estimated — real-time data unavailable."
	}

	routeDesc := fmt.Sprintf("%s → %s", origin, destination)
	if returnOrigin != "" && returnOrigin != destination {
		routeDesc = fmt.Sprintf("%s → %s (returning from %s → %s, multi-city)", origin, destination, returnOrigin, origin)
	}

	prompt := fmt.Sprintf(`[INST] You are a helpful travel assistant. Analyze these options and give brief, honest recommendations.

Trip: %s | %s to %s | %d passenger(s) | Budget: $%.0f%s

Flights available (price is per person, round-trip total):
`, routeDesc, departureDate, returnDate, passengers, budget, dataNote)

	for i, f := range flights {
		if i >= 5 {
			break
		}
		prompt += fmt.Sprintf("  %d. %s — $%.0f (%d stop(s), %s)\n", i+1, f.Airline, f.Price, f.Stops, f.Duration)
	}

	prompt += "\nHotels (per night):\n"
	for i, h := range hotels {
		if i >= 5 {
			break
		}
		prompt += fmt.Sprintf("  %d. %s — $%.0f/night (★%.1f) %s\n", i+1, h.Name, h.Price, h.Rating, h.Location)
	}

	highlights := DestinationHighlights(destination)
	if highlights != "" {
		prompt += fmt.Sprintf("\nTop things to do in %s: %s\n", destination, highlights)
	}

	prompt += `
In 150 words or fewer, recommend the best flight and hotel that fit the budget. Explain why briefly. Use sections: "✈ Flight:" and "🏨 Hotel:". If space allows, add a "🗺 Highlights:" line with 2-3 must-see spots. Be direct. [/INST]`

	return prompt
}