# Trivia Generator API  

The **Trivia Generator API** is a Go-based service that dynamically generates trivia questions based on specified topics, difficulty levels, and quantities. It utilizes **Gemini Flash 1.5** for AI-driven question generation through intelligent prompts. The API supports seamless integration via RESTful endpoints and delivers structured JSON responses for enhanced client compatibility.

---

## Features  

- **Dynamic Question Generation**: Generate trivia questions tailored to specific topics, difficulty levels, and quantities.  
- **AI Integration**: Leverages **Gemini Flash 1.5** for prompt-based question generation.  
- **JSON Response Format**: Provides structured output including questions, options, correct answers, explanations, tags, hints, and difficulty levels.  
- **Session Management**: Maintains a session history for context-aware trivia generation.  
- **Concurrency-Safe Operations**: Ensures thread safety with a mutex for managing shared resources.  

---

## Technology Stack  

- **Language**: Go  
- **Framework**: Gin (for building RESTful APIs)  
- **AI Integration**: Gemini Flash 1.5 via the `generative-ai-go` library  
- **Concurrency**: Mutex for thread-safe operations  
- **Logging**: Google’s `slog` for structured logging  

---

## Installation  

### Prerequisites  
- Go 1.20+ installed  
- API key for **Gemini Flash 1.5**  

### Steps  
1. Clone the repository:  
   ```bash
   git clone https://github.com/your-repo/trivia-generator
   cd trivia-generator
   ```  
2. Set up the environment variables by creating a `.env` file:  
   ```env
   GEMINI_API_KEY=your_gemini_api_key
   ```  
3. Install dependencies:  
   ```bash
   go mod tidy
   ```  
4. Run the service:  
   ```bash
   go run main.go
   ```  

---

## Endpoints  

### 1. **GET `/questions`**  
Generates trivia questions based on input parameters.  

#### Query Parameters:  
- `topic` (optional): Topic of the questions (e.g., "GoLang", "History").  
- `difficulty` (required): Difficulty level (e.g., "7").  
- `quantity` (required): Number of questions to generate (e.g., "5", "Medium").  

#### Example Request:  
```http
GET /questions?topic=GoLang&difficulty=7&quantity=5
```  

#### Example Response:  
```json
[
  {
    "topic": "GoLang",
    "tags": ["Go", "programming", "language", "concurrency"],
    "question": "What is the primary data structure used for goroutine communication?",
    "options": ["channel", "map", "array", "slice"],
    "correct_answer": "channel",
    "explanation": "Channels are the primary mechanism for communication and synchronization between goroutines in Go.",
    "difficulty_level": 7,
    "hints": [
      "It's a built-in type in Go.",
      "It facilitates data transfer between concurrent processes.",
      "It handles synchronization issues inherent in concurrency."
    ]
  }
]
```  

---

## Key Components  

### `Question` Struct  
Defines the structure of a trivia question:  
```go
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
```  

### Trivia Service  
Handles AI interactions and session management:  
- **`generateQuestions`**: Sends prompts to Gemini Flash 1.5 and parses the response into structured questions.  
- **`loadSessionHistory`**: Initializes session history from saved files.  
- **`mergeAndWriteQuestions`**: Updates and saves the context for future use.  

---

Feel free to contribute to the project by submitting issues or pull requests!  