package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"

	_ "github.com/lib/pq"
)

type Option struct {
	Text      string
	IsCorrect bool
}

type Question struct {
	ID          int
	Title       string
	Options     []Option
	Explanation string
}

// Database connection string
const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "your_password"
	dbname   = "quiz_db"
)

var db *sql.DB

func main() {
	// Initialize database connection
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Error connecting to database:", err)
	}
	defer db.Close()

	// Test the connection
	err = db.Ping()
	if err != nil {
		log.Fatal("Error connecting to the database:", err)
	}

	// Initialize database schema
	err = initDB()
	if err != nil {
		log.Fatal("Error initializing database:", err)
	}

	// Routes
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/submit-question", handleSubmitQuestion)
	http.HandleFunc("/check-answer", handleCheckAnswer)

	log.Println("Server starting on :8084...")
	log.Fatal(http.ListenAndServe(":8084", nil))
}

func initDB() error {
	// Create questions table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS questions (
			id SERIAL PRIMARY KEY,
			title TEXT NOT NULL,
			options JSONB NOT NULL,
			explanation TEXT NOT NULL
		)
	`)
	return err
}

func getQuestions() ([]Question, error) {
	rows, err := db.Query("SELECT id, title, options, explanation FROM questions")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var questions []Question
	for rows.Next() {
		var q Question
		var optionsJSON []byte
		err := rows.Scan(&q.ID, &q.Title, &optionsJSON, &q.Explanation)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(optionsJSON, &q.Options)
		if err != nil {
			return nil, err
		}

		questions = append(questions, q)
	}
	return questions, nil
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	questions, err := getQuestions()
	if err != nil {
		http.Error(w, "Error fetching questions", http.StatusInternalServerError)
		return
	}

	tmpl := template.Must(template.ParseFiles("templates/index.html"))
	err = tmpl.Execute(w, questions)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func handleSubmitQuestion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Create options array
	options := make([]Option, 4)
	correctOpt, err := strconv.Atoi(r.FormValue("correct_option"))
	if err != nil || correctOpt < 1 || correctOpt > 4 {
		http.Error(w, "Invalid correct option number", http.StatusBadRequest)
		return
	}

	for i := 0; i < 4; i++ {
		options[i] = Option{
			Text:      r.FormValue(fmt.Sprintf("option%d", i+1)),
			IsCorrect: (i + 1) == correctOpt,
		}
	}

	// Convert options to JSON
	optionsJSON, err := json.Marshal(options)
	if err != nil {
		http.Error(w, "Error encoding options", http.StatusInternalServerError)
		return
	}

	// Insert into database
	_, err = db.Exec(
		"INSERT INTO questions (title, options, explanation) VALUES ($1, $2, $3)",
		r.FormValue("title"),
		optionsJSON,
		r.FormValue("explanation"),
	)
	if err != nil {
		http.Error(w, "Error saving question", http.StatusInternalServerError)
		return
	}

	// Fetch updated questions
	questions, err := getQuestions()
	if err != nil {
		http.Error(w, "Error fetching questions", http.StatusInternalServerError)
		return
	}

	// Execute template with updated questions
	tmpl := template.Must(template.ParseFiles("templates/index.html"))
	err = tmpl.ExecuteTemplate(w, "question-list", questions)
	if err != nil {
		http.Error(w, "Template execution error", http.StatusInternalServerError)
		return
	}
}

func handleCheckAnswer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	questionID := r.FormValue("question_index")
	selectedOption := r.FormValue("option")

	// Fetch question from database
	var optionsJSON []byte
	var explanation string
	err := db.QueryRow(
		"SELECT options, explanation FROM questions WHERE id = $1",
		questionID,
	).Scan(&optionsJSON, &explanation)
	if err != nil {
		http.Error(w, "Error fetching question", http.StatusInternalServerError)
		return
	}

	var options []Option
	err = json.Unmarshal(optionsJSON, &options)
	if err != nil {
		http.Error(w, "Error parsing options", http.StatusInternalServerError)
		return
	}

	optIdx, err := strconv.Atoi(selectedOption)
	if err != nil || optIdx < 0 || optIdx >= len(options) {
		http.Error(w, "Invalid option index", http.StatusBadRequest)
		return
	}

	response := struct {
		Correct     bool
		Explanation string
	}{
		Correct:     options[optIdx].IsCorrect,
		Explanation: explanation,
	}

	tmpl := template.Must(template.ParseFiles("templates/index.html"))
	err = tmpl.ExecuteTemplate(w, "answer-response", response)
	if err != nil {
		http.Error(w, "Template execution error", http.StatusInternalServerError)
		return
	}
}
