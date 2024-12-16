package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/generative-ai-go/genai"
	"github.com/vivek-344/trivia-generator/util"
	"google.golang.org/api/option"
)

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

type TriviaService struct {
	client  *genai.Client
	model   *genai.GenerativeModel
	session *genai.ChatSession
	logger  *slog.Logger
	mutex   sync.RWMutex
}

func NewTriviaService(apiKey string) (*TriviaService, error) {
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	model := client.GenerativeModel("gemini-1.5-flash")
	model.SetTemperature(1)
	model.SetTopK(40)
	model.SetTopP(0.95)
	model.SetMaxOutputTokens(8192)
	model.ResponseMIMEType = "application/json"

	session := model.StartChat()

	return &TriviaService{
		client:  client,
		model:   model,
		session: session,
		logger:  logger,
	}, nil
}

func (ts *TriviaService) Close() {
	ts.client.Close()
}

func (ts *TriviaService) loadSessionHistory(promptPath, contextPath string) error {
	promptData, err := os.ReadFile(promptPath)
	if err != nil {
		return fmt.Errorf("error reading prompt file: %w", err)
	}

	contextData, err := ts.ensureContextFile(contextPath)
	if err != nil {
		return err
	}

	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	ts.session.History = []*genai.Content{
		{
			Role:  "user",
			Parts: []genai.Part{genai.Text(string(promptData))},
		},
		{
			Role:  "model",
			Parts: []genai.Part{genai.Text(string(contextData))},
		},
	}

	return nil
}

func (ts *TriviaService) ensureContextFile(contextPath string) ([]byte, error) {
	contextData, err := os.ReadFile(contextPath)
	if err != nil {
		if os.IsNotExist(err) {
			contextData = []byte("[]")
			if err := os.WriteFile(contextPath, contextData, 0644); err != nil {
				return nil, fmt.Errorf("error creating context.json: %w", err)
			}
		} else {
			return nil, fmt.Errorf("error reading context file: %w", err)
		}
	}
	return contextData, nil
}

func (ts *TriviaService) generateQuestions(ctx context.Context, prompt string) ([]Question, error) {
	resp, err := ts.session.SendMessage(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("error sending message: %w", err)
	}

	var questions []Question
	for _, part := range resp.Candidates[0].Content.Parts {
		textContent := part.(genai.Text)
		if err := json.Unmarshal([]byte(textContent), &questions); err != nil {
			return nil, fmt.Errorf("error parsing questions: %w", err)
		}
	}

	return questions, nil
}

func (ts *TriviaService) mergeAndWriteQuestions(contextData []byte, newQuestions []Question, contextPath string) error {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	var previousQuestions []Question
	if len(contextData) > 0 {
		if err := json.Unmarshal(contextData, &previousQuestions); err != nil {
			return fmt.Errorf("error unmarshaling previous questions: %w", err)
		}
	}

	mergedQuestions := append(previousQuestions, newQuestions...)
	mergedJSON, err := json.MarshalIndent(mergedQuestions, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling merged questions: %w", err)
	}

	return os.WriteFile(contextPath, mergedJSON, 0644)
}

func main() {
	// Load configuration
	config, err := util.LoadConfig(".")
	if err != nil {
		slog.Error("Cannot load config", "error", err)
		os.Exit(1)
	}

	// Create trivia service
	triviaService, err := NewTriviaService(config.GeminiKey)
	if err != nil {
		slog.Error("Failed to create trivia service", "error", err)
		os.Exit(1)
	}
	defer triviaService.Close()

	// Load session history
	if err := triviaService.loadSessionHistory("./prompt.txt", "./context.json"); err != nil {
		slog.Error("Failed to load session history", "error", err)
		os.Exit(1)
	}

	// Setup router
	router := setupRouter(triviaService)

	// Start server
	if err := router.Run(); err != nil {
		slog.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
}

func setupRouter(triviaService *TriviaService) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	router.GET("/questions", triviaService.getQuestions)

	return router
}

func (ts *TriviaService) getQuestions(c *gin.Context) {
	topic := c.Query("topic")
	difficulty := c.Query("difficulty")
	quantity := c.Query("quantity")

	// Validate inputs
	if difficulty == "" || quantity == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing query parameters. Please provide topic (optional), difficulty, and quantity.",
		})
		return
	}

	// Create a prompt
	prompt := fmt.Sprintf("Topic: %s\nDifficulty: %s\nQuantity: %s", topic, difficulty, quantity)

	// Generate questions
	questions, err := ts.generateQuestions(c, prompt)
	if err != nil {
		ts.logger.Error("Failed to generate questions", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate questions"})
		return
	}

	// Read and merge context
	contextData, err := os.ReadFile("./context.json")
	if err != nil {
		ts.logger.Error("Failed to read context", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read context"})
		return
	}

	// Merge and write questions
	if err := ts.mergeAndWriteQuestions(contextData, questions, "./context.json"); err != nil {
		ts.logger.Error("Failed to merge questions", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to merge questions"})
		return
	}

	c.JSON(http.StatusOK, questions)
}
