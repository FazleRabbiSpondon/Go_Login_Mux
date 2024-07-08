package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

var db *sql.DB

func main() {
	var err error
	connStr := "host=172.17.0.2 port=5432 user=spondon password=1234 dbname=temp_db sslmode=disable"
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	err = createTable()
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()

	mux.Handle("GET /users", http.HandlerFunc(getUsers))
	mux.Handle("POST /users", http.HandlerFunc(createUser))

	mux.Handle("DELETE /user/", http.HandlerFunc(deleteUser))
	mux.Handle("GET /user/", http.HandlerFunc(getUser))
	mux.Handle("PUT /user/", http.HandlerFunc(updateUser))

	mux.Handle("POST /user/", http.HandlerFunc(userlogin))

	// http.HandleFunc("/users", usersHandler)
	// http.HandleFunc("/user/", userHandler)

	// http.HandleFunc("/userlogin", userlogin)

	fmt.Println("Server is running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func userlogin(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if user.Name == "" {
		http.Error(w, "Please mention the name", http.StatusBadRequest)
		return
	}

	if user.Password == "" {
		http.Error(w, "Please mention the PASSWORD", http.StatusBadRequest)
		return
	}

	var hash string
	stmt := "SELECT password FROM users2 WHERE name = $1"
	row := db.QueryRow(stmt, user.Name)
	fmt.Println(row)
	err := row.Scan(&hash)
	fmt.Println("Password from database:", hash)

	err2 := bcrypt.CompareHashAndPassword([]byte(hash), []byte(user.Password))

	if err != nil {
		fmt.Println("error selecting PASSWORD in db by Username")
		fmt.Fprint(w, "A problem occured please go back and try again")
		//return
	}

	if err2 == nil {
		fmt.Fprint(w, "You have successfully logged in :)")
		return
	}

	fmt.Fprint(w, "incorrect password")
}

func createTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS users2 (
		id SERIAL PRIMARY KEY,
		name VARCHAR(100) UNIQUE,
		password VARCHAR(100)
	);`
	_, err := db.Exec(query)
	return err
}


func getUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name, password FROM users2")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Name, &user.Password); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func createUser(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if user.Name == "" {
		http.Error(w, "Please mention the name", http.StatusBadRequest)
		return
	}

	if user.Password == "" {
		http.Error(w, "Please mention the e-mail", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)

	if err != nil {
		fmt.Fprint(w, "Couldn't hash the PASSWORD")
	}

	err = db.QueryRow("INSERT INTO users2 (name, password) VALUES ($1, $2) RETURNING id", user.Name, hash).Scan(&user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		fmt.Fprint(w, "User Created")
	}
	// w.Header().Set("Content-Type", "application/json")
	// json.NewEncoder(w).Encode(user)


}

func getUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/user/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var user User
	err = db.QueryRow("SELECT id, name, password FROM users2 WHERE id = $1", id).Scan(&user.ID, &user.Name, &user.Password)
	if err == sql.ErrNoRows {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func updateUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/user/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := "UPDATE users2 SET"
	params := []interface{}{}
	paramID := 1

	if user.Name != "" {
		query += " name = $" + strconv.Itoa(paramID)
		params = append(params, user.Name)
		paramID++
	}

	if user.Password != "" {
		if paramID > 1 {
			query += ","
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)

		if err != nil {
			fmt.Fprint(w, "Couldn't hash the PASSWORD")
		}

		user.Password = string(hash)

		query += " password = $" + strconv.Itoa(paramID)
		params = append(params, user.Password)
		paramID++
	}

	if paramID == 1 {
		http.Error(w, "No fields to update", http.StatusBadRequest)
		return
	}

	query += " WHERE id = $" + strconv.Itoa(paramID)
	params = append(params, id)

	_, err2 := db.Exec(query, params...)
	if err2 != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//w.WriteHeader(http.StatusNoContent)
	fmt.Fprint(w, "User updated")
}

func deleteUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/user/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	_, err2 := db.Exec("DELETE FROM users2 WHERE id = $1", id)
	if err2 != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		http.Error(w, "User deleted", http.StatusNotFound)
	}

	w.WriteHeader(http.StatusNoContent)
}
