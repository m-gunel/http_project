package main

import (
	"encoding/json"
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
)

type UsersBefore struct {
	Id         int    `xml:"id"`
	First_name string `xml:"first_name"`
	Last_name  string `xml:"last_name"`
	Age        int    `xml:"age"`
	About      string `xml:"about"`
	Gender     string `xml:"gender"`
}

var usersBefore struct {
	Rows []UsersBefore `xml:"row"`
}

func SearchServer(w http.ResponseWriter, req *http.Request) {
	file, err := os.Open("dataset.xml")
	if err != nil {
		http.Error(w, "Error opening file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		http.Error(w, "Error reading file", http.StatusInternalServerError)
		return
	}

	err = xml.Unmarshal(data, &usersBefore)
	if err != nil {
		http.Error(w, "Error unmarshalling XML", http.StatusInternalServerError)
		return
	}

	limit, err := strconv.Atoi(req.FormValue("limit"))
	if err != nil || limit < 0 {
		http.Error(w, "Invalid limit", http.StatusBadRequest)
		return
	}

	offset, err := strconv.Atoi(req.FormValue("offset"))
	if err != nil || offset < 0 {
		http.Error(w, "Invalid offset", http.StatusBadRequest)
		return
	}

	query := req.FormValue("query")
	orderField := req.FormValue("order_field")
	if orderField == "" {
		orderField = "Name"
	}


	if orderField != "Age" && orderField != "Id" && orderField != "Name" {
		http.Error(w, "Invalid order field", http.StatusBadRequest)
		return
	}

	orderBy, err := strconv.Atoi(req.FormValue("order_by"))
	if err != nil || (orderBy != 1 && orderBy != -1) {
		http.Error(w, "Invalid orderBy", http.StatusBadRequest)
		return
	}

	var users []User
	for _, el := range usersBefore.Rows {
		name := strings.TrimSpace(el.First_name + " " + el.Last_name)
		queryLower := strings.ToLower(query)
		nameLower := strings.ToLower(name)
		aboutLower := strings.ToLower(el.About)

		if query == "" || strings.Contains(nameLower, queryLower) || strings.Contains(aboutLower, queryLower) {
			users = append(users, User{
				Id:     el.Id,
				Name:   name,
				Age:    el.Age,
				About:  el.About,
				Gender: el.Gender,
			})
		}
	}

	sort.Slice(users, func(i, j int) bool {
		switch orderField {
		case "Age":
			if orderBy == 1 {
				return users[i].Age < users[j].Age
			}
			return users[i].Age > users[j].Age
		case "Id":
			if orderBy == 1 {
				return users[i].Id < users[j].Id
			}
			return users[i].Id > users[j].Id
		case "Name":
			if orderBy == 1 {
				return users[i].Name < users[j].Name
			}
			return users[i].Name > users[j].Name
		}
		return false
	})

	if offset+limit > len(users) {
		limit = len(users) - offset
	}
	if offset >= len(users) {
		limit = 0
	}

	users = users[offset : offset+limit]

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(users)
	if err != nil {
		http.Error(w, "Error marshalling JSON", http.StatusInternalServerError)
		return
	}
}


func TestSearchServer(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer mockServer.Close()

	req, err := http.NewRequest("GET", mockServer.URL+"?limit=10&offset=0&query=Boyd&order_field=Name&order_by=-1", nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %s", resp.Status)
	}

	var users []User
	err = json.NewDecoder(resp.Body).Decode(&users)
	if err != nil {
		t.Fatal(err)
	}

	if len(users) == 0 {
		t.Errorf("Expected users, got 0")
	}

	
}