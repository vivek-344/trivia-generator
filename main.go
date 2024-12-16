package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/generative-ai-go/genai"
	"github.com/vivek-344/trivia-generator/util"
	"google.golang.org/api/option"
)

var session *genai.ChatSession

type Question struct {
	Topic           string   `json:"topic"`
	Tags            []string `json:"tags"`
	Question        string   `json:"question"`
	Options         []string `json:"options"`
	CorrectAnswer   string   `json:"correct_answer"`
	Explanation     string   `json:"explanation"`
	DifficultyLevel int      `json:"difficulty_level"`
	Hints           []string `json:"hints"`
}

func main() {
	// Initialize context
	ctx := context.Background()

	// Load the config
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal("cannot load config: ", err)
	}

	apiKey := config.GeminiKey

	// Create the client
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()

	// Define the model to use
	model := client.GenerativeModel("gemini-1.5-flash")

	// Set model parameters
	model.SetTemperature(1)
	model.SetTopK(40)
	model.SetTopP(0.95)
	model.SetMaxOutputTokens(8192)
	model.ResponseMIMEType = "application/json"

	// Start a chat session
	session = model.StartChat()

	// Read the prompt and context files
	promptData, err := os.ReadFile("./prompt.txt")
	if err != nil {
		log.Fatalf("Error reading prompt file: %v", err)
	}
	// Read the context.json file or initialize it if it doesn't exist
	contextData, err := os.ReadFile("./context.json")
	if err != nil {
		if os.IsNotExist(err) {
			// Initialize with an empty JSON array and create the file
			contextData = []byte("[]")
			err := os.WriteFile("./context.json", contextData, 0644)
			if err != nil {
				log.Fatalf("Error creating context.json: %v", err)
			}
		} else {
			log.Fatalf("Error reading context file: %v", err)
		}
	}

	// Set the history for the session
	if contextData != nil {
		session.History = []*genai.Content{
			{
				Role: "user",
				Parts: []genai.Part{
					genai.Text(string(promptData)),
				},
			},
			{
				Role: "model",
				Parts: []genai.Part{
					genai.Text(string(contextData)),
				},
			},
		}
	} else {
		session.History = []*genai.Content{
			{
				Role: "user",
				Parts: []genai.Part{
					genai.Text(string(promptData)),
				},
			},
		}
	}

	router := gin.Default()
	router.GET("/questions", getQuestions)

	router.Run()
}

func mergeAndWrite(old_json []byte, new_questions []Question) {
	var previous_questions []Question
	if len(old_json) > 0 {
		if err := json.Unmarshal(old_json, &previous_questions); err != nil {
			log.Fatal(err)
		}
	}

	// Merge the two slices
	mergedQuestions := append(previous_questions, new_questions...)

	// Marshal the merged result back into JSON (if needed)
	mergedJSON, err := json.MarshalIndent(mergedQuestions, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	// Write the merged result to context.json
	err = os.WriteFile("./context.json", mergedJSON, 0644)
	if err != nil {
		log.Fatalf("Error writing to context.json: %v", err)
	}
}

func getQuestions(ctx *gin.Context) {
	topic := ctx.Query("topic")
	difficulty := ctx.Query("difficulty")
	quantity := ctx.Query("quantity")

	// Validate inputs
	if difficulty == "" || quantity == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing query parameters. Please provide topic (optional), difficulty, and quantity.",
		})
		return
	}

	// Create a prompt based on query parameters
	prompt := "Topic: " + topic + "\nDifficulty: " + difficulty + "\nQuantity: " + quantity

	// Send a message to the model using the generated prompt
	resp, err := session.SendMessage(ctx, genai.Text(prompt))
	if err != nil {
		log.Printf("Error sending message: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate questions."})
		return
	}

	var questions []Question
	for _, part := range resp.Candidates[0].Content.Parts {
		textContent := part.(genai.Text)
		if err := json.Unmarshal([]byte(textContent), &questions); err != nil {
			log.Printf("Error unmarshalling response content: %v", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse questions."})
			return
		}
	}

	contextData, err := os.ReadFile("./context.json")
	if err != nil {
		log.Fatalf("Error reading context file: %v", err)
	}

	mergeAndWrite(contextData, questions)

	ctx.JSON(http.StatusOK, questions)
}
