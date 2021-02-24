package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"github.com/vonage/vonage-go-sdk"
)

var (
	// Key must be 16, 24 or 32 bytes long (AES-128, AES-192 or AES-256)
	key   = []byte("super-secret-key")
	store = sessions.NewCookieStore(key)
)

// UserData: Info from session that we'll use in the UI
type UserData struct {
	Name  string
	Phone string
}

var verifyClient *vonage.VerifyClient
var requestID string

func home(w http.ResponseWriter, r *http.Request) {

	session, _ := store.Get(r, "acmeinc-cookie")

	// Check if user is authenticated
	if auth, ok := session.Values["registered"].(bool); !ok || !auth {
		// Not authenticated, so user must register
		http.Redirect(w, r, "/register", 302)
	}

	userData := UserData{
		Name:  fmt.Sprintf("%v", session.Values["name"]),
		Phone: fmt.Sprintf("%v", session.Values["phoneNumber"]),
	}

	files := []string{
		"./tmpl/home.page.gohtml",
		"./tmpl/base.layout.gohtml",
	}

	/* Use the template.ParseFiles() function to read the files and store the
	templates in a template set.*/
	ts, err := template.ParseFiles(files...)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", 500)
		return
	}

	err = ts.Execute(w, userData)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", 500)
	}
}

func register(w http.ResponseWriter, r *http.Request) {
	files := []string{
		"./tmpl/register.page.gohtml",
		"./tmpl/base.layout.gohtml",
	}

	ts, err := template.ParseFiles(files...)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", 500)
		return
	}

	err = ts.Execute(w, nil)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", 500)
	}
}

func verify(w http.ResponseWriter, r *http.Request) {

	session, _ := store.Get(r, "acmeinc-cookie")
	// retrieve user's name and phone number from the submitted form
	userName := r.URL.Query().Get("name")
	phoneNumber := r.URL.Query().Get("phone_number")
	session.Values["name"] = userName
	session.Values["phoneNumber"] = phoneNumber
	session.Save(r, w)
	log.Println("Verifying...." + userName + " at " + phoneNumber)
	response, errResp, err := verifyClient.Request(phoneNumber, "GoTest", vonage.VerifyOpts{CodeLength: 6, Lg: "en-gb", WorkflowID: 4})

	if err != nil {
		fmt.Printf("%#v\n", err)
	} else if response.Status != "0" {
		fmt.Println("Error status " + errResp.Status + ": " + errResp.ErrorText)
	} else {
		requestID = response.RequestId
		fmt.Println("Request started: " + response.RequestId)
		// redirect to "check" page
		http.Redirect(w, r, "/enter-code", 302)
	}
}

func enterCode(w http.ResponseWriter, r *http.Request) {
	files := []string{
		"./tmpl/entercode.page.gohtml",
		"./tmpl/base.layout.gohtml",
	}

	ts, err := template.ParseFiles(files...)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", 500)
		return
	}

	err = ts.Execute(w, nil)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", 500)
	}
}

func checkCode(w http.ResponseWriter, r *http.Request) {
	// Retrieve the PIN code that the user entered
	session, _ := store.Get(r, "acmeinc-cookie")
	pinCode := r.URL.Query().Get("pin_code")
	response, errResp, err := verifyClient.Check(requestID, pinCode)

	if err != nil {
		fmt.Printf("%#v\n", err)
	} else if response.Status != "0" {
		fmt.Println("Error status " + errResp.Status + ": " + errResp.ErrorText)
	} else {
		fmt.Println("Request complete: " + response.RequestId)
		// Set user as authenticated and return to home page
		session.Values["registered"] = true
		session.Save(r, w)
		http.Redirect(w, r, "/", 302)
	}
}

func unregister(w http.ResponseWriter, r *http.Request) {
	// Delete the session
	session, _ := store.Get(r, "acmeinc-cookie")
	session.Options.MaxAge = -1
	session.Save(r, w)
	http.Redirect(w, r, "/", 302)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
		os.Exit(1)
	}

	apiKey := os.Getenv("VONAGE_API_KEY")
	apiSecret := os.Getenv("VONAGE_API_SECRET")

	auth := vonage.CreateAuthFromKeySecret(apiKey, apiSecret)
	verifyClient = vonage.NewVerifyClient(auth)

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/", home)
	mux.HandleFunc("/register", register)
	mux.HandleFunc("/verify", verify)
	mux.HandleFunc("/enter-code", enterCode)
	mux.HandleFunc("/check-code", checkCode)
	mux.HandleFunc("/clear", unregister)

	log.Println("Starting server on :5000")
	err = http.ListenAndServe(":5000", mux)
	log.Fatal(err)
}
