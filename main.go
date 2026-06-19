package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/pquerna/otp/totp"
	"golang.org/x/term"
)

var (
	currentUser    *User
	currentSession *Session
)

func main() {
	db, err := ConnectDB()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		log.Fatalf("Failed to set raw mode: %v", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	rw := struct {
		io.Reader
		io.Writer
	}{os.Stdin, os.Stdout}

	t := term.NewTerminal(rw, "> ")

	t.AutoCompleteCallback = func(line string, pos int, key rune) (newLine string, newPos int, ok bool) {
		var cmds []string
		if currentUser != nil {
			cmds = []string{"whoami", "enable-2fa", "disable-2fa", "logout", "help", "exit"}
		} else {
			cmds = []string{"register", "login", "help", "exit"}
		}

		if key == '\t' {
			for _, cmd := range cmds {
				if strings.HasPrefix(cmd, line) {
					return cmd, len(cmd), true
				}
			}
		}
		return "", 0, false
	}

	fmt.Fprintln(t, "Successfully connected to the database!")
	fmt.Fprintln(t, "Welcome! Type 'help' for a list of commands.")

	for {
		if currentUser != nil {
			t.SetPrompt(fmt.Sprintf("%s> ", currentUser.Username))
		} else {
			t.SetPrompt("> ")
		}

		line, err := t.ReadLine()
		if err != nil {
			break
		}

		cmd := strings.TrimSpace(line)
		cmd = strings.ToLower(cmd)
		if cmd == "" {
			continue
		}

		if currentUser != nil {
			switch cmd {
			case "whoami":
				handleWhoami(t)
			case "enable-2fa":
				handleEnable2FA(db, t)
			case "disable-2fa":
				handleDisable2FA(db, t)
			case "logout":
				handleLogout(db, t)
			case "help":
				fmt.Fprintln(t, "Available commands:")
				fmt.Fprintln(t, "  whoami      - Show user and session details")
				fmt.Fprintln(t, "  enable-2fa  - Setup Google Authenticator 2FA")
				fmt.Fprintln(t, "  disable-2fa - Remove 2FA from your account")
				fmt.Fprintln(t, "  logout      - End your current session")
				fmt.Fprintln(t, "  help        - Show this message")
				fmt.Fprintln(t, "  exit        - Quit the application")
			case "exit":
				fmt.Fprintln(t, "Goodbye!")
				return
			default:
				fmt.Fprintf(t, "Unknown command: %s. Type 'help' for options.\n", cmd)
			}
		} else {
			switch cmd {
			case "register":
				handleRegister(db, t)
			case "login":
				handleLogin(db, t)
			case "help":
				fmt.Fprintln(t, "Available commands:")
				fmt.Fprintln(t, "  register - Create a new account")
				fmt.Fprintln(t, "  login    - Log into your account")
				fmt.Fprintln(t, "  help     - Show this message")
				fmt.Fprintln(t, "  exit     - Quit the application")
			case "exit":
				fmt.Fprintln(t, "Goodbye!")
				return
			default:
				fmt.Fprintf(t, "Unknown command: %s. Type 'help' for options.\n", cmd)
			}
		}
	}
}

func handleRegister(db *sql.DB, t *term.Terminal) {
	t.SetPrompt("Enter username: ")
	username, err := t.ReadLine()
	if err != nil {
		return
	}
	username = strings.TrimSpace(username)

	password, err := t.ReadPassword("Enter password: ")
	if err != nil {
		fmt.Fprintln(t, "Error reading password:", err)
		return
	}

	err = RegisterUser(db, username, password)
	if err != nil {
		fmt.Fprintln(t, "Registration failed:", err)
		return
	}
	fmt.Fprintln(t, "Registration successful! You can now log in.")
}

func handleLogin(db *sql.DB, t *term.Terminal) {
	t.SetPrompt("Enter username: ")
	username, err := t.ReadLine()
	if err != nil {
		return
	}
	username = strings.TrimSpace(username)

	password, err := t.ReadPassword("Enter password: ")
	if err != nil {
		fmt.Fprintln(t, "Error reading password:", err)
		return
	}

	user, session, err := LoginUser(db, username, password, "")
	if err != nil {
		if err.Error() == "totp_required" {
			totpCode, err := t.ReadPassword("Enter 2FA Code: ")
			if err != nil {
				return
			}
			totpCode = strings.TrimSpace(totpCode)
			user, session, err = LoginUser(db, username, password, totpCode)
			if err != nil {
				fmt.Fprintln(t, "Login failed:", err)
				return
			}
		} else {
			fmt.Fprintln(t, "Login failed:", err)
			return
		}
	}

	currentUser = user
	currentSession = session
	fmt.Fprintf(t, "Login successful! Welcome, %s.\n", user.Username)
	
	// Automatically display whoami as per assignment requirement
	handleWhoami(t)
}

func handleWhoami(t *term.Terminal) {
	fmt.Fprintln(t, "User Profile:")
	fmt.Fprintf(t, "  Username:      %s\n", currentUser.Username)
	fmt.Fprintf(t, "  Registered:    %s\n", currentUser.CreatedAt.Format(time.RFC1123))
	
	lastLogin := "Never"
	if currentUser.LastLogin.Valid {
		lastLogin = currentUser.LastLogin.Time.Format(time.RFC1123)
	}
	fmt.Fprintf(t, "  Last Login:    %s\n", lastLogin)
	fmt.Fprintf(t, "  MFA Enabled:   %t\n", currentUser.TotpEnabled)
	fmt.Fprintf(t, "  Session Ends:  %s\n", currentSession.ExpiresAt.Format(time.RFC1123))
}

func handleEnable2FA(db *sql.DB, t *term.Terminal) {
	if currentUser.TotpEnabled {
		fmt.Fprintln(t, "2FA is already enabled on your account.")
		return
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Go CLI App",
		AccountName: currentUser.Username,
	})
	if err != nil {
		fmt.Fprintln(t, "Error generating TOTP secret:", err)
		return
	}

	fmt.Fprintln(t, "1. Open your authenticator app (e.g., Google Authenticator).")
	fmt.Fprintf(t, "2. Add a new account manually using this secret key: %s\n", key.Secret())
	
	code, err := t.ReadPassword("3. Enter the 6-digit code generated by the app to verify: ")
	if err != nil {
		return
	}
	code = strings.TrimSpace(code)

	valid := totp.Validate(code, key.Secret())
	if !valid {
		fmt.Fprintln(t, "Verification failed! The code was incorrect. 2FA not enabled.")
		return
	}

	err = Enable2FA(db, currentUser.ID, key.Secret())
	if err != nil {
		fmt.Fprintln(t, "Failed to save 2FA settings to the database:", err)
		return
	}

	currentUser.TotpEnabled = true
	fmt.Fprintln(t, "2FA successfully enabled!")
}

func handleDisable2FA(db *sql.DB, t *term.Terminal) {
	if !currentUser.TotpEnabled {
		fmt.Fprintln(t, "2FA is not enabled on your account.")
		return
	}

	err := Disable2FA(db, currentUser.ID)
	if err != nil {
		fmt.Fprintln(t, "Failed to disable 2FA in the database:", err)
		return
	}

	currentUser.TotpEnabled = false
	fmt.Fprintln(t, "2FA has been disabled.")
}

func handleLogout(db *sql.DB, t *term.Terminal) {
	err := LogoutUser(db, currentSession.ID)
	if err != nil {
		fmt.Fprintln(t, "Warning: Failed to clear session from DB:", err)
	}
	currentUser = nil
	currentSession = nil
	fmt.Fprintln(t, "You have been logged out.")
}
