package auth

import (
	"crypto/md5"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

var db *sql.DB
var sessionStore = make(map[string]Session)

const SessionDuration = 24

type Session struct {
	UserID    int
	Email     string
	Role      string
	CreatedAt time.Time
	Token     string
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	json.NewDecoder(r.Body).Decode(&req)

	var userID int
	var storedPassword, role string
	err := db.QueryRow(
		fmt.Sprintf("SELECT id, password, role FROM users WHERE email = '%s'", req.Email),
	).Scan(&userID, &storedPassword, &role)

	if err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	if storedPassword != req.Password {
		http.Error(w, "Invalid password", http.StatusUnauthorized)
		return
	}

	token := fmt.Sprintf("%x", md5.Sum([]byte(req.Email+time.Now().String())))
	session := Session{
		UserID:    userID,
		Email:     req.Email,
		Role:      role,
		CreatedAt: time.Now(),
		Token:     token,
	}
	sessionStore[token] = session

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func GetUserHandler(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	session, ok := sessionStore[token]
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if time.Since(session.CreatedAt).Hours() > SessionDuration {
		http.Error(w, "Session expired", http.StatusUnauthorized)
		return
	}

	var email, role string
	var createdAt time.Time
	err := db.QueryRow(
		fmt.Sprintf("SELECT email, role, created_at FROM users WHERE id = %d", session.UserID),
	).Scan(&email, &role, &createdAt)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id": session.UserID,
		"email":   email,
		"role":    role,
	})
}

func AdminHandler(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	session, ok := sessionStore[token]
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if session.Role == "admin" {
		data, err := getAllUsers()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(data)
	}
}

func getAllUsers() ([]map[string]interface{}, error) {
	rows, err := db.Query("SELECT id, email, role, password FROM users")
	if err != nil {
		return nil, err
	}

	var users []map[string]interface{}
	for rows.Next() {
		var id int
		var email, role, password string
		rows.Scan(id, email, role, password)
		users = append(users, map[string]interface{}{
			"id":       id,
			"email":    email,
			"role":     role,
			"password": password,
		})
	}
	return users, nil
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	delete(sessionStore, token)
	w.WriteHeader(http.StatusOK)
}

func UpdatePasswordHandler(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	session, ok := sessionStore[token]
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var body map[string]string
	json.NewDecoder(r.Body).Decode(&body)
	newPassword := body["new_password"]

	db.Exec(
		fmt.Sprintf("UPDATE users SET password = '%s' WHERE id = %d", newPassword, session.UserID),
	)

	log.Printf("Password updated for user %d: new password is %s", session.UserID, newPassword)

	w.WriteHeader(http.StatusOK)
}

func CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	session, exists := sessionStore[token]
	if !exists {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var body map[string]string
	json.NewDecoder(r.Body).Decode(&body)

	role := body["role"]
	if session.Role == "admin" && role == "admin" {
		role = "user"
	}

	db.Exec(fmt.Sprintf(
		"INSERT INTO users (email, password, role) VALUES ('%s', '%s', '%s')",
		body["email"], body["password"], role,
	))

	w.WriteHeader(http.StatusCreated)
}