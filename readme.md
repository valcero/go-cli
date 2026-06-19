# Containerized CLI Login System with Optional 2FA

A secure, interactive command-line login system built in Go. It supports user registration, authentication, optional TOTP-based 2FA (Google Authenticator compatible), and session management. The entire stack is containerized using Docker and backed by PostgreSQL.

## Features
- **Authentication**: Bcrypt password hashing, session management with timeouts.
- **Account Security**: Lockout mechanism (15-minute lock after 3 failed attempts).
- **Two-Factor Authentication (2FA)**: Time-Based One-Time Password (TOTP) support.
- **Interactive CLI**: Auto-completes commands via `<TAB>`, tracks command history (via up/down arrows), and provides masked password input using `golang.org/x/term`.
- **Database Persistence**: Containerized PostgreSQL database with volume mounts to ensure data survives restarts.

## Prerequisites
- Docker and Docker Compose installed.

## Setup Instructions

1. **Clone the repository and enter the directory**:
   ```bash
   git clone <repository_url>
   cd go-cli
   ```

2. **Start the Database**:
   Spin up the PostgreSQL container in the background. The `schema.sql` will automatically initialize the required tables on the first run.
   ```bash
   docker-compose up -d
   ```

3. **Build the CLI Application**:
   Build the multi-stage Docker image for the Go application.
   ```bash
   docker build -t mycli .
   ```

4. **Run the CLI Application**:
   Because this is an interactive CLI, you must run it interactively (`-it`) attached to the host network (or passing the proper DB_HOST) so it can reach the database container.
   ```bash
   # On Linux / macOS:
   docker run -it --rm --network host -e DB_HOST=localhost mycli

   # On Windows / Docker Desktop:
   docker run -it --rm -e DB_HOST=host.docker.internal mycli
   ```

## Usage/Commands

### Before Login
- `register` - Create a new user account.
- `login` - Authenticate using your username and password (prompts for TOTP if enabled).
- `help` - Show available commands.
- `exit` - Quit the application.

### After Login
- `whoami` - Show current user details including registration date, MFA status, session expiration, and last login time.
- `enable-2fa` - Setup Google Authenticator 2FA. Generates a secret key for you to add manually.
- `disable-2fa` - Remove 2FA requirement from your account.
- `logout` - End your current session securely.
- `help` - Show available commands.
- `exit` - Quit the application.

## Technologies Used
- **Go 1.22+**: standard library + `x/term` + `x/crypto/bcrypt` + `pquerna/otp`
- **PostgreSQL**: For persistent data storage.
- **Docker**: Multi-stage application builds and orchestration.
