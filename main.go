package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type User struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	PassportNumber string `json:"passportNumber"`
}

type Task struct {
	ID          int     `json:"id"`
	UserID      int     `json:"user_id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
	Rate        float64 `json:"rate"`
	Deadline    int     `json:"deadline"`
	Earned      float64 `json:"earned"`
}

type TemplateData struct {
	Users  []User
	Tasks  []Task
	Filter struct {
		UserID int
		SortBy string
	}
}

func getUsers() ([]User, error) {
	resp, err := http.Get("http://localhost:8080/api/users?offset=0&limit=100")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var users []User
	err = json.Unmarshal(body, &users)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func getTasks(userID int, sortBy string) ([]Task, error) {
	var url string
	if userID > 0 {
		url = "http://localhost:8080/api/tasks?user_id=" + strconv.Itoa(userID) + "&sort_by=" + sortBy
	} else {
		url = "http://localhost:8080/api/tasks?sort_by=" + sortBy
	}
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tasks []Task
	err = json.Unmarshal(body, &tasks)
	if err != nil {
		return nil, err
	}

	return tasks, nil
}

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		userID, _ := strconv.Atoi(r.URL.Query().Get("user_id"))
		sortBy := r.URL.Query().Get("sort_by")

		users, err := getUsers()
		if err != nil {
			log.Println(err)
			http.Error(w, "Failed to load users", http.StatusInternalServerError)
			return
		}

		tasks, err := getTasks(userID, sortBy)
		if err != nil {
			log.Println(err)
			http.Error(w, "Failed to load tasks", http.StatusInternalServerError)
			return
		}

		t, err := template.ParseFiles("templates/index.html")
		if err != nil {
			log.Println("Error loading template:", err)
			http.Error(w, "Failed to load template: "+err.Error(), http.StatusInternalServerError)
			return
		}

		data := TemplateData{
			Users: users,
			Tasks: tasks,
		}
		data.Filter.UserID = userID
		data.Filter.SortBy = sortBy

		err = t.Execute(w, data)
		if err != nil {
			log.Println("Error executing template:", err)
			http.Error(w, "Failed to execute template: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}).Methods("GET")

	r.HandleFunc("/task/create", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			// Parse form values
			err := r.ParseForm()
			if err != nil {
				http.Error(w, "Failed to parse form", http.StatusInternalServerError)
				return
			}

			userID, _ := strconv.Atoi(r.FormValue("user_id"))
			description := r.FormValue("description")
			rate, _ := strconv.ParseFloat(r.FormValue("rate"), 64)
			deadline, _ := strconv.Atoi(r.FormValue("deadline"))

			// Create task map to match the expected JSON structure
			task := map[string]interface{}{
				"userid":      userID,
				"description": description,
				"rate":        rate,
				"deadline":    deadline,
			}

			body, err := json.Marshal(task)
			if err != nil {
				http.Error(w, "Failed to marshal task", http.StatusInternalServerError)
				return
			}

			resp, err := http.Post("http://localhost:8080/api/tasks", "application/json", bytes.NewBuffer(body))
			if err != nil {
				http.Error(w, "Failed to create task", http.StatusInternalServerError)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusCreated {
				http.Error(w, "Failed to create task", resp.StatusCode)
				return
			}

			http.Redirect(w, r, "/", http.StatusSeeOther)
		}
	}).Methods("POST")

	r.HandleFunc("/task/delete", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			taskID, _ := strconv.Atoi(r.FormValue("task_id"))
			req, err := http.NewRequest("DELETE", "http://localhost:8080/api/tasks/"+strconv.Itoa(taskID), nil)
			if err != nil {
				http.Error(w, "Failed to delete task", http.StatusInternalServerError)
				return
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				http.Error(w, "Failed to delete task", http.StatusInternalServerError)
				return
			}
			defer resp.Body.Close()

			http.Redirect(w, r, "/", http.StatusSeeOther)
		}
	}).Methods("POST")

	r.HandleFunc("/task/update", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			taskID, _ := strconv.Atoi(r.FormValue("task_id"))
			action := r.FormValue("action")

			var url string
			if action == "start" {
				url = "http://localhost:8080/api/tasks/" + strconv.Itoa(taskID) + "/start"
			} else if action == "stop" {
				url = "http://localhost:8080/api/tasks/" + strconv.Itoa(taskID) + "/stop"
			}

			req, err := http.NewRequest("POST", url, nil)
			if err != nil {
				http.Error(w, "Failed to update task", http.StatusInternalServerError)
				return
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				http.Error(w, "Failed to update task", http.StatusInternalServerError)
				return
			}
			defer resp.Body.Close()

			http.Redirect(w, r, "/", http.StatusSeeOther)
		}
	}).Methods("POST")

	r.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		users, err := getUsers()
		if err != nil {
			log.Println(err)
			http.Error(w, "Failed to load users", http.StatusInternalServerError)
			return
		}

		t, err := template.ParseFiles("templates/users.html")
		if err != nil {
			log.Println("Error loading template:", err)
			http.Error(w, "Failed to load template: "+err.Error(), http.StatusInternalServerError)
			return
		}

		err = t.Execute(w, users)
		if err != nil {
			log.Println("Error executing template:", err)
			http.Error(w, "Failed to execute template: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}).Methods("GET")

	r.HandleFunc("/user/create", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			name := r.FormValue("name")
			passport := r.FormValue("passport")
			passportSeries := passport[:4]
			passportNumber := passport[5:]

			user := map[string]interface{}{
				"name":           name,
				"passportNumber": passportSeries + " " + passportNumber,
			}

			body, err := json.Marshal(user)
			if err != nil {
				http.Error(w, "Failed to marshal user", http.StatusInternalServerError)
				return
			}

			resp, err := http.Post("http://localhost:8080/api/users", "application/json", bytes.NewBuffer(body))
			if err != nil {
				http.Error(w, "Failed to create user", http.StatusInternalServerError)
				return
			}
			defer resp.Body.Close()

			http.Redirect(w, r, "/users", http.StatusSeeOther)
		}
	}).Methods("POST")

	r.HandleFunc("/user/update", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			err := r.ParseForm()
			if err != nil {
				http.Error(w, "Failed to parse form", http.StatusInternalServerError)
				return
			}

			userIDStr := r.FormValue("user_id")
			userID, err := strconv.Atoi(userIDStr)
			if err != nil {
				log.Printf("Invalid user ID: %s, error: %v", userIDStr, err)
				http.Error(w, "Invalid user ID", http.StatusBadRequest)
				return
			}

			name := r.FormValue("user_name")
			passport := r.FormValue("new_passport")
			if len(passport) < 6 {
				http.Error(w, "Invalid passport format", http.StatusBadRequest)
				return
			}
			passportSeries := passport[:4]
			passportNumber := passport[5:]

			user := map[string]interface{}{
				"name":           name,
				"passportNumber": passportSeries + " " + passportNumber,
			}

			body, err := json.Marshal(user)
			if err != nil {
				http.Error(w, "Failed to marshal user", http.StatusInternalServerError)
				return
			}

			req, err := http.NewRequest("PUT", "http://localhost:8080/api/users/"+strconv.Itoa(userID), bytes.NewBuffer(body))
			if err != nil {
				http.Error(w, "Failed to create request", http.StatusInternalServerError)
				return
			}
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				http.Error(w, "Failed to update user", http.StatusInternalServerError)
				return
			}
			defer resp.Body.Close()

			respBody, _ := ioutil.ReadAll(resp.Body)
			log.Printf("Response status: %d, body: %s", resp.StatusCode, string(respBody))

			if resp.StatusCode != http.StatusOK {
				http.Error(w, "Failed to update user", resp.StatusCode)
				return
			}

			http.Redirect(w, r, "/users", http.StatusSeeOther)
		}
	}).Methods("POST")

	r.HandleFunc("/user/delete", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			userID, _ := strconv.Atoi(r.FormValue("user_id"))
			req, err := http.NewRequest("DELETE", "http://localhost:8080/api/users/"+strconv.Itoa(userID), nil)
			if err != nil {
				http.Error(w, "Failed to delete user", http.StatusInternalServerError)
				return
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				http.Error(w, "Failed to delete user", http.StatusInternalServerError)
				return
			}
			defer resp.Body.Close()

			http.Redirect(w, r, "/users", http.StatusSeeOther)
		}
	}).Methods("POST")

	http.Handle("/", r)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	log.Fatal(http.ListenAndServe(":8081", nil))
}
