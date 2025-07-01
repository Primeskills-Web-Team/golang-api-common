package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
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
	webhookURL   string
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

// BuildSlackMessage builds formatted Slack message from alert data
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

    // Tambahkan event details jika ada
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

            // Data JSON dari event
            if data, ok := event["data"]; ok {
                if marshaled, err := json.MarshalIndent(data, "", "  "); err == nil {
                    fields = append(fields, SlackField{
                        Title: "Event Data",
                        Value: fmt.Sprintf("```json\n%s\n```", marshaled),
                        Short: false,
                    })
                }
            }
        }
    }

    attachment := SlackAttachment{
        Color:     color,
        Title:     fmt.Sprintf("Failed to publish to topic: %s", topic),
        Text:      fmt.Sprintf("%s", errorMsg),
        Timestamp: timestamp.Unix(),
        Footer:    "Kafka Alert System",
        Fields:    fields,
    }

    if severity == "critical" {
        attachment.Text += "\n\n⚠️ This is a critical alert requiring immediate attention!"
    }

    return SlackMessage{
        Text:        mainText,
        Username:    GetEnvOrDefault("SLACK_USERNAME", "Kafka Alert Bot"),
        IconEmoji:   GetEnvOrDefault("SLACK_ICON_EMOJI", ":warning:"),
        Channel:     GetEnvOrDefault("SLACK_CHANNEL", "#kafka-alerts"),
        Attachments: []SlackAttachment{attachment},
    }
}


// SendToSlack sends message to Slack webhook with enhanced error handling
func (s *SlackHelper) SendToSlack(webhookURL string, message SlackMessage) error {
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack message: %w", err)
	}

	fmt.Printf("Sending Slack message to: %s, size: %d bytes\n",
		MaskWebhookURL(webhookURL), len(jsonData))

	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Kafka-Alert-Bot/1.0")

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack webhook returned status %d: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("Slack message sent successfully, status: %d\n", resp.StatusCode)
	return nil
}

// SendSlackAlert sends alert to Slack (main function)
func (s *SlackHelper) SendSlackAlert(alert map[string]interface{}) {
	if !s.IsSlackEnabled() {
		return
	}

	// Get webhook URL priority: 1. From constructor, 2. From environment
	webhookURL := s.webhookURL
	if webhookURL == "" {
		webhookURL = os.Getenv("SLACK_WEBHOOK_URL")
	}

	if webhookURL == "" {
		fmt.Println("SLACK_WEBHOOK_URL not configured, skipping Slack alert")
		return
	}

	// Check if alerts enabled
	alertEnabled := os.Getenv("ALERT_ENABLED")
	if alertEnabled != "true" && alertEnabled != "1" {
		fmt.Println("Slack alerts disabled, skipping")
		return
	}

	fmt.Printf("Preparing to send Slack alert for topic: %v\n", alert["topic"])

	// Build and send message
	message := s.BuildSlackMessage(alert)
	if err := s.SendToSlack(webhookURL, message); err != nil {
		fmt.Printf("Failed to send Slack alert: %v\n", err)
	} else {
		fmt.Printf("✅ Successfully sent Slack alert for topic: %v, severity: %v\n",
			alert["topic"], alert["severity"])
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
