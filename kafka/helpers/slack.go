package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

// SlackMessage represents a Slack message structure
type SlackMessage struct {
	Text        string            `json:"text"`
	Username    string            `json:"username,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	Channel     string            `json:"channel,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
}

// SlackAttachment represents a Slack message attachment
type SlackAttachment struct {
	Color     string       `json:"color,omitempty"`
	Title     string       `json:"title,omitempty"`
	Text      string       `json:"text,omitempty"`
	Fields    []SlackField `json:"fields,omitempty"`
	Timestamp int64        `json:"ts,omitempty"`
	Footer    string       `json:"footer,omitempty"`
}

// SlackField represents a field in Slack attachment
type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// SlackHelper contains methods for Slack operations
type SlackHelper struct {
	webhookURL string
	// alertEnabled bool
}

// NewSlackHelper creates a new Slack helper instance
func NewSlackHelper(webhookURL string) *SlackHelper {
	return &SlackHelper{
		webhookURL: webhookURL,
	}
}

// IsSlackEnabled checks if Slack alerts are enabled
func (s *SlackHelper) IsSlackEnabled() bool {
	_ = godotenv.Load("./../../.env")
	enabled := os.Getenv("ALERT_ENABLED")
	return enabled == "true" || enabled == "1"
}

// GetSlackColor returns Slack color based on severity
func (s *SlackHelper) GetSlackColor(severity string) string {
	switch severity {
	case "critical":
		return "danger" // Red
	case "warning":
		return "warning" // Yellow
	case "info":
		return "good" // Green
	default:
		return "#808080" // Gray
	}
}

func (s *SlackHelper) BuildSlackMessage(alert map[string]interface{}) SlackMessage {
	severity := alert["severity"].(string)
	topic := alert["topic"].(string)
	errorMsg := alert["error"].(string)
	timestamp := alert["timestamp"].(time.Time)

	color := s.GetSlackColor(severity)
	mainText := fmt.Sprintf("🚨 Kafka Failure Alert - %s", strings.ToUpper(severity))

	fields := []SlackField{
		{
			Title: "Service",
			Value: GetEnvOrDefault("APP_NAME", "unknown-service"),
			Short: true,
		},
		{
			Title: "Environment",
			Value: GetEnvOrDefault("ENVIRONMENT", "unknown"),
			Short: true,
		},
		{
			Title: "Topic",
			Value: topic,
			Short: true,
		},
		{
			Title: "Severity",
			Value: strings.ToUpper(severity),
			Short: true,
		},
		{
			Title: "Kafka Hosts",
			Value: fmt.Sprintf("%v", alert["kafka_hosts"]),
			Short: false,
		},
		{
			Title: "Timestamp",
			Value: timestamp.Format("2006-01-02 15:04:05 MST"),
			Short: true,
		},
		{
			Title: "Alert Type",
			Value: fmt.Sprintf("%s", alert["alert_type"]),
			Short: true,
		},
	}

	if eventRaw, ok := alert["event"]; ok {
		if event, ok := eventRaw.(map[string]interface{}); ok {
			if eventName, ok := event["event_name"].(string); ok {
				fields = append(fields, SlackField{
					Title: "Event Name",
					Value: eventName,
					Short: true,
				})
			}
			if source, ok := event["source"].(string); ok {
				fields = append(fields, SlackField{
					Title: "Event Source",
					Value: source,
					Short: true,
				})
			}
			if data, ok := event["data"]; ok {
				if marshaled, err := json.MarshalIndent(data, "", "  "); err == nil {
					fields = append(fields, SlackField{
						Title: "Event Data",
						Value: fmt.Sprintf("```\n%s\n```", marshaled),
						Short: false,
					})
				}
			}
		}
	}

	// -- Attachment block
	attachment := SlackAttachment{
		Color:     color,
		Title:     fmt.Sprintf("Failed to publish to topic: %s", topic),
		Text:      fmt.Sprintf("```%s```", errorMsg),
		Timestamp: timestamp.Unix(),
		Footer:    "Kafka Alert System",
		Fields:    fields,
	}

	if severity == "critical" {
		attachment.Text += "\n\n⚠️ This is a critical alert requiring immediate attention!"
	}

	// -- Final message
	return SlackMessage{
		Text:        mainText,
		Username:    GetEnvOrDefault("SLACK_USERNAME", "Kafka Alert Bot"),
		IconEmoji:   GetEnvOrDefault("SLACK_ICON_EMOJI", ":warning:"),
		Channel:     GetEnvOrDefault("SLACK_CHANNEL", "#kafka-alerts"),
		Attachments: []SlackAttachment{attachment},
	}
}

func debugEnvironmentVariables() {
	webhookURL := os.Getenv("SLACK_WEBHOOK_URL")

	logrus.WithFields(logrus.Fields{
		"webhook_length": len(webhookURL),
		"webhook_prefix": webhookURL[:min(len(webhookURL), 50)],   // First 50 chars
		"webhook_suffix": webhookURL[max(0, len(webhookURL)-20):], // Last 20 chars
		"has_newline":    strings.Contains(webhookURL, "\n"),
		"has_carriage":   strings.Contains(webhookURL, "\r"),
		"has_tab":        strings.Contains(webhookURL, "\t"),
		"has_space":      strings.Contains(webhookURL, " "),
		"environment":    os.Getenv("ENVIRONMENT"),
		"alert_enabled":  os.Getenv("ALERT_ENABLED"),
		"slack_channel":  os.Getenv("SLACK_CHANNEL"),
	}).Info("Environment variable analysis")

	
	for i, char := range webhookURL {
		if char < 32 || char > 126 {
			logrus.WithFields(logrus.Fields{
				"position":  i,
				"char_code": int(char),
				"char_hex":  fmt.Sprintf("0x%02X", char),
			}).Warn("Found non-printable character in webhook URL")
		}
	}
}

func cleanWebhookURL(rawURL string) string {
	// Remove common problematic characters
	cleaned := strings.TrimSpace(rawURL)
	cleaned = strings.ReplaceAll(cleaned, "\n", "")
	cleaned = strings.ReplaceAll(cleaned, "\r", "")
	cleaned = strings.ReplaceAll(cleaned, "\t", "")

	if cleaned != rawURL {
		logrus.WithFields(logrus.Fields{
			"original_length": len(rawURL),
			"cleaned_length":  len(cleaned),
		}).Warn("Webhook URL contained whitespace/control characters - cleaned")
	}

	return cleaned
}

func (s *SlackHelper) SendToSlack(webhookURL string, message SlackMessage) error {
	// ✅ VALIDASI WEBHOOK URL DETAIL
	logrus.WithFields(logrus.Fields{
		"webhook_raw": webhookURL,
		"webhook_len": len(webhookURL),
		"environment": os.Getenv("ENVIRONMENT"),
	}).Info("Webhook URL validation")

	if !strings.HasPrefix(webhookURL, "https://hooks.slack.com/services/") {
		return fmt.Errorf("invalid Slack webhook URL format: %s", MaskWebhookURL(webhookURL))
	}

	parsedURL, err := url.Parse(webhookURL)
	if err != nil {
		return fmt.Errorf("failed to parse webhook URL: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"host":   parsedURL.Host,
		"path":   parsedURL.Path,
		"scheme": parsedURL.Scheme,
	}).Info("Parsed webhook URL")

	// ✅ Marshal SlackMessage to JSON
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack message: %w", err)
	}

	payloadSize := len(jsonData)

	logrus.WithFields(logrus.Fields{
		"webhook_url":  MaskWebhookURL(webhookURL),
		"payload_size": payloadSize,
		"environment":  os.Getenv("ENVIRONMENT"),
		"channel":      message.Channel,
	}).Info("Sending Slack message")

	// ✅ Create HTTP request
	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Kafka-Alert-Bot/1.0")

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			IdleConnTimeout:       10 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
		},
	}

	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start)

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error":       err.Error(),
			"duration_ms": duration.Milliseconds(),
		}).Error("HTTP request failed")
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	responseStr := string(body)

	logrus.WithFields(logrus.Fields{
		"status_code":    resp.StatusCode,
		"response_body":  responseStr,
		"content_length": resp.ContentLength,
		"duration_ms":    duration.Milliseconds(),
		"response_headers": map[string]string{
			"content-type": resp.Header.Get("Content-Type"),
			"server":       resp.Header.Get("Server"),
			"date":         resp.Header.Get("Date"),
		},
	}).Info("Slack webhook response details")

	if strings.HasPrefix(responseStr, "<!DOCTYPE html") || strings.Contains(responseStr, "<html") {
		logrus.WithFields(logrus.Fields{
			"response_preview": responseStr[:min(len(responseStr), 200)],
		}).Error("🚨 Received HTML response instead of JSON - Wrong endpoint or URL issue!")
		return fmt.Errorf("webhook returned HTML page instead of JSON response - check URL validity")
	}

	if resp.StatusCode != http.StatusOK {
		logrus.WithFields(logrus.Fields{
			"status_code": resp.StatusCode,
			"response":    responseStr,
		}).Error("Slack webhook returned non-200 status")
		return fmt.Errorf("slack webhook returned status %d: %s", resp.StatusCode, responseStr)
	}

	if responseStr != "ok" {
		logrus.WithField("unexpected_response", responseStr).Warn("Unexpected Slack response format")
	}

	logrus.WithFields(logrus.Fields{
		"status_code": resp.StatusCode,
		"duration_ms": duration.Milliseconds(),
		"response":    responseStr,
	}).Info("✅ Slack message sent successfully")

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// SendSlackAlert sends alert to Slack (main function)
func (s *SlackHelper) SendSlackAlert(alert map[string]interface{}) {
	// ✅ Debug environment di awal
	debugEnvironmentVariables()

	if !s.IsSlackEnabled() {
		logrus.Warn("Slack is not enabled")
		return
	}

	webhookURL := s.webhookURL
	if webhookURL == "" {
		webhookURL = os.Getenv("SLACK_WEBHOOK_URL")
	}

	if webhookURL == "" {
		logrus.Error("SLACK_WEBHOOK_URL not configured, skipping Slack alert")
		return
	}

	// ✅ CLEAN WEBHOOK URL
	webhookURL = cleanWebhookURL(webhookURL)

	// Check if alerts enabled
	alertEnabled := os.Getenv("ALERT_ENABLED")
	if alertEnabled != "true" && alertEnabled != "1" {
		logrus.WithField("alert_enabled", alertEnabled).Warn("Slack alerts disabled, skipping")
		return
	}

	logrus.WithField("topic", alert["topic"]).Info("Preparing to send Slack alert")

	// Build and send message
	message := s.BuildSlackMessage(alert)
	if err := s.SendToSlack(webhookURL, message); err != nil {
		logrus.WithError(err).Error("Failed to send Slack alert")
	} else {
		logrus.WithFields(logrus.Fields{
			"topic":    alert["topic"],
			"severity": alert["severity"],
		}).Info("✅ Successfully sent Slack alert")
	}
}

// CreateRecoveryMessage creates a recovery message for Slack
func CreateRecoveryMessage(topic string, kafkaHosts []string) SlackMessage {
	return SlackMessage{
		Text:      "✅ *Kafka Recovery Alert*",
		Username:  os.Getenv("SLACK_USERNAME"),
		IconEmoji: ":white_check_mark:",
		Channel:   os.Getenv("SLACK_CHANNEL"),
		Attachments: []SlackAttachment{
			{
				Color: "good",
				Title: "Kafka Connection Restored",
				Text:  fmt.Sprintf("Topic %s is now accessible", topic),
				Fields: []SlackField{
					{
						Title: "Service",
						Value: GetEnvOrDefault("APP_NAME", "unknown-service"),
						Short: true,
					},
					{
						Title: "Environment",
						Value: GetEnvOrDefault("ENVIRONMENT", "unknown"),
						Short: true,
					},
					{
						Title: "Kafka Hosts",
						Value: fmt.Sprintf("%v", kafkaHosts),
						Short: false,
					},
				},
				Footer:    "Kafka Alert System",
				Timestamp: time.Now().Unix(),
			},
		},
	}
}
