package config

import (
	"os"
	"testing"
	"time"

	"github.com/Primeskills-Web-Team/golang-api-common/kafka/helpers"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func setupSlackTest(t *testing.T) *KafkaConfig {
	// ✅ Multiple path attempts untuk load .env
	envPaths := []string{
		".env",                // Current directory
		"../.env",             // Parent directory
		"../../.env",          // 2 levels up
		"../../../.env",       // 3 levels up
		"../../../../.env",    // 4 levels up
		"../../../../../.env", // 5 levels up (root project)
	}

	loaded := false
	for _, path := range envPaths {
		if err := godotenv.Load(path); err == nil {
			t.Logf("✅ Successfully loaded .env from: %s", path)
			loaded = true
			break
		}
	}

	if !loaded {
		t.Log("⚠️ Could not load .env file from any path, using system environment")

		// ✅ Manual setup untuk testing jika .env tidak ditemukan
		if os.Getenv("KAFKA_HOST") == "" {
			os.Setenv("KAFKA_HOST", "34.126.85.201:9092")
		}
		if os.Getenv("SLACK_WEBHOOK_URL") == "" {
			os.Setenv("SLACK_WEBHOOK_URL", "https://hooks.slack.com/services/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")
		}
		if os.Getenv("ALERT_ENABLED") == "" {
			os.Setenv("ALERT_ENABLED", "true")
		}
		if os.Getenv("APP_NAME") == "" {
			os.Setenv("APP_NAME", "ejourney-authorization")
		}
		if os.Getenv("ENVIRONMENT") == "" {
			os.Setenv("ENVIRONMENT", "test")
		}
	}

	// ✅ Verify environment variables loaded
	t.Logf("KAFKA_HOST: %s", os.Getenv("KAFKA_HOST"))
	t.Logf("SLACK_WEBHOOK_URL: %s", maskWebhookURL(os.Getenv("SLACK_WEBHOOK_URL")))
	t.Logf("ALERT_ENABLED: %s", os.Getenv("ALERT_ENABLED"))

	// Create Kafka config
	kafkaHost := os.Getenv("KAFKA_HOST")
	if kafkaHost == "" {
		kafkaHost = "localhost:9092"
	}

	return NewKafkaConfig("", "", []string{kafkaHost}, "")
}

// ✅ Helper function untuk mask webhook URL di log
// func maskWebhookURL(url string) string {
//     if len(url) > 20 {
//         return url[:20] + "***MASKED***"
//     }
//     return "***MASKED***"
// }

func TestSlackAlert(t *testing.T) {
	config := setupSlackTest(t)

	// ✅ Check if Slack is configured
	slackWebhook := os.Getenv("SLACK_WEBHOOK_URL")
	if slackWebhook == "" {
		t.Skip("SLACK_WEBHOOK_URL not configured, skipping Slack test")
	}

	t.Run("send test alert", func(t *testing.T) {
		err := config.sendToSlack(slackWebhook, helpers.SlackMessage{
			Text: "Test Slack alert",
		})
		assert.NoError(t, err)

		t.Log("✅ Test Slack alert sent successfully")
		t.Log("📱 Check your Slack channel for the test message")

		// ✅ Wait a bit untuk async operation
		time.Sleep(2 * time.Second)
	})

	t.Run("send failure alert", func(t *testing.T) {
		// ✅ Simulate failure alert
		testAlert := map[string]interface{}{
			"alert_type":  "kafka_publish_failed",
			"topic":       "test-failure-topic",
			"error":       "connection refused: kafka server unavailable",
			"timestamp":   time.Now(),
			"severity":    "critical",
			"service":     "test-service",
			"environment": "test",
			"kafka_hosts": []string{"localhost:9092"},
		}

		config.sendSlackAlert(testAlert)

		t.Log("✅ Failure alert sent to Slack")
		t.Log("📱 Check your Slack channel for the failure alert")

		// ✅ Wait untuk async operation
		time.Sleep(2 * time.Second)
	})

	t.Run("send recovery alert", func(t *testing.T) {
		config.SendRecoveryAlert("test-recovery-topic")

		t.Log("✅ Recovery alert sent to Slack")
		t.Log("📱 Check your Slack channel for the recovery alert")

		// ✅ Wait untuk async operation
		time.Sleep(2 * time.Second)
	})
}

func TestSlackMessageBuilding(t *testing.T) {
	config := setupSlackTest(t)

	t.Run("build slack message", func(t *testing.T) {
		alert := map[string]interface{}{
			"alert_type":  "kafka_publish_failed",
			"topic":       "test-topic",
			"error":       "connection timeout",
			"timestamp":   time.Now(),
			"severity":    "warning",
			"service":     "test-service",
			"environment": "test",
			"kafka_hosts": []string{"localhost:9092"},
		}

		message := config.slackHelper.BuildSlackMessage(alert)

		assert.NotEmpty(t, message.Text)
		assert.Contains(t, message.Text, "Kafka Failure Alert")
		assert.Equal(t, 1, len(message.Attachments))
		assert.Equal(t, "warning", message.Attachments[0].Color)

		t.Logf("Built message text: %s", message.Text)
		t.Logf("Attachment color: %s", message.Attachments[0].Color)
		t.Logf("Fields count: %d", len(message.Attachments[0].Fields))
	})

	t.Run("get slack colors", func(t *testing.T) {
		assert.Equal(t, "danger", config.slackHelper.GetSlackColor("critical"))
		assert.Equal(t, "warning", config.slackHelper.GetSlackColor("warning"))
		assert.Equal(t, "good", config.slackHelper.GetSlackColor("info"))
		assert.Equal(t, "#808080", config.slackHelper.GetSlackColor("unknown"))

		t.Log("✅ All color mappings working correctly")
	})
}

// ✅ Test untuk verify semua environment variables
func TestEnvironmentVariables(t *testing.T) {
	setupSlackTest(t)

	requiredEnvVars := map[string]string{
		"KAFKA_HOST":        os.Getenv("KAFKA_HOST"),
		"SLACK_WEBHOOK_URL": os.Getenv("SLACK_WEBHOOK_URL"),
		"ALERT_ENABLED":     os.Getenv("ALERT_ENABLED"),
		"APP_NAME":          os.Getenv("APP_NAME"),
		"ENVIRONMENT":       os.Getenv("ENVIRONMENT"),
	}

	t.Log("📋 Environment Variables Status:")
	for key, value := range requiredEnvVars {
		if value != "" {
			if key == "SLACK_WEBHOOK_URL" {
				t.Logf("✅ %s: %s", key, maskWebhookURL(value))
			} else {
				t.Logf("✅ %s: %s", key, value)
			}
		} else {
			t.Logf("❌ %s: NOT SET", key)
		}
	}
}

// ✅ Integration test dengan real Slack (optional)
func TestRealSlackIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real Slack integration test in short mode")
	}

	config := setupSlackTest(t)

	slackWebhook := os.Getenv("SLACK_WEBHOOK_URL")
	if slackWebhook == "" {
		t.Skip("SLACK_WEBHOOK_URL not configured, skipping real integration test")
	}

	t.Run("real slack test", func(t *testing.T) {
		t.Log("🚀 Sending REAL Slack alert...")

		// ✅ Send test alert
		err := config.sendToSlack(slackWebhook, helpers.SlackMessage{
			Text: "Test Slack alert",
		})
		assert.NoError(t, err)

		// ✅ Wait untuk delivery
		time.Sleep(3 * time.Second)

		t.Log("✅ Real Slack alert sent!")
		t.Log("📱 Check your Slack channel: #kafka-alerts")
		t.Log("💡 If you don't see the message, check:")
		t.Log("   - Webhook URL is correct")
		t.Log("   - Bot has permission to post")
		t.Log("   - Channel exists")
	})
}
