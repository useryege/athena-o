package main

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/joho/godotenv"
)

const (
	defaultEnvFile = ".env"

	smtpHost = "smtp.exmail.qq.com"
	smtpPort = 465

	fixedRecipient = "2687665142@qq.com"
	fromName       = "ATHENA Task Notification"

	maxAttempts    = 3
	attemptTimeout = 10 * time.Second
)

var retryDelays = [...]time.Duration{
	1 * time.Second,
	2 * time.Second,
}

type emailConfig struct {
	username string
	password string
	from     mail.Address
	to       mail.Address
	subject  string
	body     string
}

func main() {
	os.Exit(run(os.Stdout, os.Stderr))
}

func run(stdout, stderr io.Writer) int {
	if err := loadEnvironment(); err != nil {
		fmt.Fprintf(stderr, "error: load task notification environment: %v\n", err)
		return 1
	}

	config, err := readConfig()
	if err != nil {
		fmt.Fprintf(stderr, "error: configure task notification email: %v\n", err)
		return 1
	}
	messageID, err := newMessageID(config.from.Address)
	if err != nil {
		fmt.Fprintf(stderr, "error: create task notification message ID: %v\n", err)
		return 1
	}
	message := buildMessage(config, messageID, time.Now())

	attempt, err := sendWithRetry(config, message, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "error: send task notification email: %v\n", err)
		return 1
	}

	fmt.Fprintf(
		stdout,
		"SMTP accepted task notification email for %s on attempt %d/%d\n",
		config.to.Address,
		attempt,
		maxAttempts,
	)
	return 0
}

func loadEnvironment() error {
	configuredPath := strings.TrimSpace(os.Getenv("TASK_NOTIFICATION_ENV_FILE"))
	envFile := configuredPath
	if envFile == "" {
		envFile = defaultEnvFile
	}

	if err := godotenv.Load(envFile); err != nil {
		if configuredPath == "" && errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("environment file %q does not exist", envFile)
		}
		return fmt.Errorf("environment file %q is invalid or unreadable", envFile)
	}
	return nil
}

func readConfig() (emailConfig, error) {
	username, err := parseUsername(os.Getenv("ATHENA_TASK_NOTIFICATION_EMAIL_SMTP_USERNAME"))
	if err != nil {
		return emailConfig{}, err
	}
	password := os.Getenv("ATHENA_TASK_NOTIFICATION_EMAIL_SMTP_PASSWORD")
	if strings.TrimSpace(password) == "" {
		return emailConfig{}, fmt.Errorf("ATHENA_TASK_NOTIFICATION_EMAIL_SMTP_PASSWORD is required")
	}

	subject := os.Getenv("TASK_NOTIFICATION_SUBJECT")
	if strings.TrimSpace(subject) == "" {
		return emailConfig{}, fmt.Errorf("TASK_NOTIFICATION_SUBJECT is required")
	}
	if strings.ContainsAny(subject, "\r\n") {
		return emailConfig{}, fmt.Errorf("TASK_NOTIFICATION_SUBJECT must not contain a line break")
	}
	if !utf8.ValidString(subject) {
		return emailConfig{}, fmt.Errorf("TASK_NOTIFICATION_SUBJECT must be valid UTF-8")
	}

	body := os.Getenv("TASK_NOTIFICATION_BODY")
	if strings.TrimSpace(body) == "" {
		return emailConfig{}, fmt.Errorf("TASK_NOTIFICATION_BODY is required")
	}
	if !utf8.ValidString(body) {
		return emailConfig{}, fmt.Errorf("TASK_NOTIFICATION_BODY must be valid UTF-8")
	}

	return emailConfig{
		username: username,
		password: password,
		from: mail.Address{
			Name:    fromName,
			Address: username,
		},
		to: mail.Address{
			Address: fixedRecipient,
		},
		subject: subject,
		body:    body,
	}, nil
}

func parseUsername(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("ATHENA_TASK_NOTIFICATION_EMAIL_SMTP_USERNAME is required")
	}
	address, err := mail.ParseAddress(raw)
	if err != nil || address.Name != "" || address.Address != raw {
		return "", fmt.Errorf("ATHENA_TASK_NOTIFICATION_EMAIL_SMTP_USERNAME must be one complete email address")
	}
	if strings.IndexFunc(address.Address, func(value rune) bool { return value > 127 }) >= 0 {
		return "", fmt.Errorf("ATHENA_TASK_NOTIFICATION_EMAIL_SMTP_USERNAME must use an ASCII email address")
	}
	return address.Address, nil
}

func newMessageID(sender string) (string, error) {
	at := strings.LastIndexByte(sender, '@')
	if at <= 0 || at == len(sender)-1 {
		return "", fmt.Errorf("sender address has no domain")
	}
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("read secure randomness: %w", err)
	}
	return fmt.Sprintf("<%s@%s>", hex.EncodeToString(randomBytes), sender[at+1:]), nil
}

func buildMessage(config emailConfig, messageID string, now time.Time) []byte {
	encodedSubject := mime.QEncoding.Encode("UTF-8", config.subject)
	encodedBody := base64.StdEncoding.EncodeToString([]byte(config.body))

	var message strings.Builder
	fmt.Fprintf(&message, "From: %s\r\n", config.from.String())
	fmt.Fprintf(&message, "To: %s\r\n", config.to.String())
	fmt.Fprintf(&message, "Subject: %s\r\n", encodedSubject)
	fmt.Fprintf(&message, "Date: %s\r\n", now.Format(time.RFC1123Z))
	fmt.Fprintf(&message, "Message-ID: %s\r\n", messageID)
	message.WriteString("MIME-Version: 1.0\r\n")
	message.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	message.WriteString("Content-Transfer-Encoding: base64\r\n")
	message.WriteString("\r\n")
	for len(encodedBody) > 76 {
		message.WriteString(encodedBody[:76])
		message.WriteString("\r\n")
		encodedBody = encodedBody[76:]
	}
	message.WriteString(encodedBody)
	message.WriteString("\r\n")
	return []byte(message.String())
}

func sendWithRetry(config emailConfig, message []byte, stderr io.Writer) (int, error) {
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := sendAttempt(config, message); err == nil {
			return attempt, nil
		} else {
			lastErr = err
		}

		if attempt == maxAttempts {
			break
		}
		delay := retryDelays[attempt-1]
		fmt.Fprintf(
			stderr,
			"task notification email attempt %d/%d failed: %s; retrying in %s\n",
			attempt,
			maxAttempts,
			compactError(lastErr, config.username, config.password),
			delay,
		)
		time.Sleep(delay)
	}
	return 0, fmt.Errorf(
		"all %d attempts failed: %s",
		maxAttempts,
		compactError(lastErr, config.username, config.password),
	)
}

func sendAttempt(config emailConfig, message []byte) error {
	deadline := time.Now().Add(attemptTimeout)
	address := net.JoinHostPort(smtpHost, strconv.Itoa(smtpPort))
	connection, err := tls.DialWithDialer(
		&net.Dialer{Deadline: deadline},
		"tcp",
		address,
		&tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: smtpHost,
		},
	)
	if err != nil {
		return fmt.Errorf("connect to %s: %w", address, err)
	}
	defer connection.Close()
	if err := connection.SetDeadline(deadline); err != nil {
		return fmt.Errorf("set SMTP deadline: %w", err)
	}

	client, err := smtp.NewClient(connection, smtpHost)
	if err != nil {
		return fmt.Errorf("create SMTP client: %w", err)
	}
	defer client.Close()

	auth := smtp.PlainAuth("", config.username, config.password, smtpHost)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("authenticate SMTP account: %w", err)
	}
	if err := client.Mail(config.from.Address); err != nil {
		return fmt.Errorf("set envelope sender: %w", err)
	}
	if err := client.Rcpt(config.to.Address); err != nil {
		return fmt.Errorf("set envelope recipient: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("start SMTP message: %w", err)
	}
	if _, err := writer.Write(message); err != nil {
		return fmt.Errorf("write SMTP message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("finish SMTP message: %w", err)
	}

	// A successful DATA close means the server accepted the message. QUIT is
	// best-effort so a disconnect here cannot trigger a duplicate retry.
	_ = client.Quit()
	return nil
}

func compactError(err error, sensitiveValues ...string) string {
	if err == nil {
		return "unknown error"
	}
	message := strings.NewReplacer("\r", " ", "\n", " ", "\t", " ").Replace(err.Error())
	for _, sensitiveValue := range sensitiveValues {
		if sensitiveValue != "" {
			message = strings.ReplaceAll(message, sensitiveValue, "[redacted]")
		}
	}
	return strings.Join(strings.Fields(message), " ")
}
