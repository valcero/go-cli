package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/term"
)

func main() {
	db, err := ConnectDB()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	fmt.Println("Successfully connected to the database!")
	fmt.Println("Welcome! Type 'help' for a list of commands.")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break // Exit loop on EOF (e.g., Ctrl+D)
		}

		cmd := strings.TrimSpace(scanner.Text())
		cmd = strings.ToLower(cmd)

		switch cmd {
		case "register":
			handleRegister(db, scanner)
		case "login":
			handleLogin(db, scanner)
		case "help":
			fmt.Println("Available commands:")
			fmt.Println("  register - Create a new account")
			fmt.Println("  login    - Log into your account")
			fmt.Println("  help     - Show this message")
			fmt.Println("  exit     - Quit the application")
		case "exit":
			fmt.Println("Goodbye!")
			return
		case "":
			continue // ignore empty input
		default:
			fmt.Printf("Unknown command: %s. Type 'help' for options.\n", cmd)
		}
	}
}

func handleRegister(db *sql.DB, scanner *bufio.Scanner) {
	fmt.Print("Enter username: ")
	if !scanner.Scan() {
		return
	}
	username := strings.TrimSpace(scanner.Text())

	fmt.Print("Enter password: ")
	passBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // ReadPassword doesn't echo the newline, so we print one
	if err != nil {
		fmt.Println("Error reading password:", err)
		return
	}

	err = RegisterUser(db, username, string(passBytes))
	if err != nil {
		fmt.Println("Registration failed:", err)
		return
	}
	fmt.Println("Registration successful! You can now log in.")
}

func handleLogin(db *sql.DB, scanner *bufio.Scanner) {
	fmt.Print("Enter username: ")
	if !scanner.Scan() {
		return
	}
	username := strings.TrimSpace(scanner.Text())

	fmt.Print("Enter password: ")
	passBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		fmt.Println("Error reading password:", err)
		return
	}

	user, err := LoginUser(db, username, string(passBytes))
	if err != nil {
		fmt.Println("Login failed:", err)
		return
	}

	fmt.Printf("Login successful! Welcome, %s.\n", user.Username)
}
