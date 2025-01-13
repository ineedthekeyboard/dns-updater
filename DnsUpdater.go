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
	Type string `json:"type"`
	Name string `json:"name"`
	Data string `json:"data"`
	TTL  int    `json:"ttl"`
	ID   int    `json:"id"`
}

type DNSRecordsResponse struct {
	DomainRecords []DNSRecord `json:"domain_records"`
}

func getParentDomain(domain string) string {
	parts := strings.Split(domain, ".")
	if len(parts) > 2 {
		return strings.Join(parts[len(parts)-2:], ".")
	}
	return domain
}

func getSubdomain(domain string) string {
	parts := strings.Split(domain, ".")
	if len(parts) > 2 {
		return strings.Join(parts[:len(parts)-2], ".")
	}
	return "@"
}

func getRecordID(config Config) (string, error) {
	parentDomain := getParentDomain(config.Domain)
	subdomain := getSubdomain(config.Domain)
	
	log.Printf("Looking for subdomain '%s' in parent domain '%s'", subdomain, parentDomain)
	
	baseURL := fmt.Sprintf("https://api.digitalocean.com/v2/domains/%s/records", parentDomain)
	currentURL := baseURL
	
	var allRecords []DNSRecord
	
	for currentURL != "" {
			log.Printf("Fetching DNS records from: %s", currentURL)
			
			req, err := http.NewRequest("GET", currentURL, nil)
			if err != nil {
					return "", fmt.Errorf("failed to create request: %v", err)
			}

			req.Header.Set("Authorization", "Bearer "+config.APIToken)
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
					return "", fmt.Errorf("failed to send request: %v", err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
					return "", fmt.Errorf("failed to read response body: %v", err)
			}
			
			if resp.StatusCode != http.StatusOK {
					log.Printf("Request failed. Domain: %s, Status: %d, Response: %s", parentDomain, resp.StatusCode, string(body))
					return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
			}

			// Define structures to match the exact JSON response
			type PageResponse struct {
					DomainRecords []DNSRecord `json:"domain_records"`
					Links struct {
							Pages struct {
									Last string `json:"last"`
									Next string `json:"next"`
							} `json:"pages"`
					} `json:"links"`
					Meta struct {
							Total int `json:"total"`
					} `json:"meta"`
			}

			var pageResponse PageResponse
			if err := json.Unmarshal(body, &pageResponse); err != nil {
					return "", fmt.Errorf("failed to parse response: %v", err)
			}

			log.Printf("Fetched page with %d records", len(pageResponse.DomainRecords))
			
			// Append the records from this page
			allRecords = append(allRecords, pageResponse.DomainRecords...)
			
			// Update URL for next page or exit if no more pages
			currentURL = pageResponse.Links.Pages.Next
			if currentURL == "" {
					log.Printf("No more pages to fetch")
			}
	}

	log.Printf("Total records fetched: %d", len(allRecords))

	// Look for the A record matching the subdomain
	for _, record := range allRecords {
			// log.Printf("Checking record - Type: %s, Name: %s", record.Type, record.Name)
			if record.Type == "A" && record.Name == subdomain {
					log.Printf("Found matching record ID: %d", record.ID)
					return fmt.Sprintf("%d", record.ID), nil
			}
	}

	return "", fmt.Errorf("no matching A record found for subdomain %s in domain %s", subdomain, parentDomain)
}

func updateDNS(config Config, ip string) error {
	log.Printf("Updating DNS - Domain: %s, Record ID: %s, New IP: %s", config.Domain, config.RecordID, ip)
	record := DNSRecord{
		Type: "A",
		Name: getSubdomain(config.Domain),
		Data: ip,
		TTL:  3600,
	}

	jsonData, err := json.Marshal(record)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.digitalocean.com/v2/domains/%s/records/%s", getParentDomain(config.Domain), config.RecordID)
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
		value = strings.Trim(value, `"' `)

		switch key {
		case "DO_API_TOKEN":
			config.APIToken = value
		case "DO_DOMAIN":
			config.Domain = value
		}
	}

	if config.APIToken == "" || config.Domain == "" {
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

func main() {
	config, err := loadEnvFile(".env")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Get the record ID for the domain
	recordID, err := getRecordID(config)
	if err != nil {
		log.Fatalf("Failed to get record ID: %v", err)
	}
	config.RecordID = recordID

	log.Printf("Starting DNS updater for domain: %s with record ID: %s", config.Domain, config.RecordID)

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
