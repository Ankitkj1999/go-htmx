package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
)

type Option struct {
	Text      string
	IsCorrect bool
}

type Question struct {
	Title       string
	Options     []Option
	Explanation string
}

var questions []Question

func main() {
	// Add a sample question
	questions = append(questions, Question{
		Title: "What is the capital of France?",
		Options: []Option{
			{Text: "London", IsCorrect: false},
			{Text: "Berlin", IsCorrect: false},
			{Text: "Paris", IsCorrect: true},
			{Text: "Madrid", IsCorrect: false},
		},
		Explanation: "Paris is the capital and largest city of France.",
	})

	// Routes
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/submit-question", handleSubmitQuestion)
	http.HandleFunc("/check-answer", handleCheckAnswer)

	log.Println("Server starting on :8084...")
	log.Fatal(http.ListenAndServe(":8084", nil))
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/index.html"))
	err := tmpl.Execute(w, questions)
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

	// Parse form if not already parsed
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	question := Question{
		Title:       r.FormValue("title"),
		Explanation: r.FormValue("explanation"),
		Options:     make([]Option, 4),
	}

	correctOpt := r.FormValue("correct_option")
	correctOptNum, err := strconv.Atoi(correctOpt)
	if err != nil || correctOptNum < 1 || correctOptNum > 4 {
		http.Error(w, "Invalid correct option number", http.StatusBadRequest)
		return
	}

	for i := 0; i < 4; i++ {
		question.Options[i] = Option{
			Text:      r.FormValue(fmt.Sprintf("option%d", i+1)),
			IsCorrect: (i + 1) == correctOptNum,
		}
	}

	questions = append(questions, question)

	// Load template
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, "Template parsing error", http.StatusInternalServerError)
		return
	}

	// Execute only the question-list block template
	w.Header().Set("Content-Type", "text/html")
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

	questionIndex := r.FormValue("question_index")
	selectedOption := r.FormValue("option")

	idx, err := strconv.Atoi(questionIndex)
	if err != nil || idx < 0 || idx >= len(questions) {
		http.Error(w, "Invalid question index", http.StatusBadRequest)
		return
	}

	optIdx, err := strconv.Atoi(selectedOption)
	if err != nil || optIdx < 0 || optIdx >= len(questions[idx].Options) {
		http.Error(w, "Invalid option index", http.StatusBadRequest)
		return
	}

	question := questions[idx]
	isCorrect := question.Options[optIdx].IsCorrect

	response := struct {
		Correct     bool
		Explanation string
	}{
		Correct:     isCorrect,
		Explanation: question.Explanation,
	}

	tmpl := template.Must(template.ParseFiles("templates/index.html"))
	err = tmpl.ExecuteTemplate(w, "answer-response", response)
	if err != nil {
		http.Error(w, "Template execution error", http.StatusInternalServerError)
		return
	}
}