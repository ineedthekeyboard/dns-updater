package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type DNSRecord struct {
	Data struct {
		Type string `json:"type"`
		Name string `json:"name"`
		Data string `json:"data"`
		TTL  int    `json:"ttl"`
	} `json:"data"`
}

type Config struct {
	APIToken string
	Domain   string
	RecordID string
}

func loadEnvFile(filename string) (Config, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return Config{}, fmt.Errorf("error reading .env file: %v", err)
	}

	config := Config{}
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		// Remove quotes if present
		value = strings.Trim(value, `"'`)

		switch key {
		case "DO_API_TOKEN":
			config.APIToken = value
		case "DO_DOMAIN":
			config.Domain = value
		case "DO_RECORD_ID":
			config.RecordID = value
		}
	}

	if config.APIToken == "" || config.Domain == "" || config.RecordID == "" {
		return config, fmt.Errorf("missing required configuration in .env file")
	}

	return config, nil
}

func getCurrentIP() (string, error) {
	resp, err := http.Get("https://api.ipify.org")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	ip, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(ip), nil
}

func updateDNS(config Config, ip string) error {
	record := DNSRecord{}
	record.Data.Type = "A"
	record.Data.Name = "@"
	record.Data.Data = ip
	record.Data.TTL = 3600

	jsonData, err := json.Marshal(record)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.digitalocean.com/v2/domains/%s/records/%s", config.Domain, config.RecordID)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+config.APIToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func main() {
	config, err := loadEnvFile(".env")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Starting DNS updater for domain: %s", config.Domain)

	for {
		ip, err := getCurrentIP()
		if err != nil {
			log.Printf("Error getting current IP: %v", err)
			time.Sleep(5 * time.Minute)
			continue
		}

		err = updateDNS(config, ip)
		if err != nil {
			log.Printf("Error updating DNS: %v", err)
		} else {
			log.Printf("Successfully updated DNS record for %s to IP %s", config.Domain, ip)
		}

		time.Sleep(5 * time.Minute)
	}
}
