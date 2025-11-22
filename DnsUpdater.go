package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
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

func getRecordID(apiToken string, domain string) (string, error) {
	parentDomain := getParentDomain(domain)
	subdomain := getSubdomain(domain)

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

		req.Header.Set("Authorization", "Bearer "+apiToken)
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
			Links         struct {
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

func updateDNS(apiToken string, domainConfig DomainConfig, ip string) error {
	log.Printf("Updating DNS - Domain: %s, Record ID: %s, New IP: %s", domainConfig.Domain, domainConfig.RecordID, ip)
	record := DNSRecord{
		Type: "A",
		Name: getSubdomain(domainConfig.Domain),
		Data: ip,
		TTL:  3600,
	}

	jsonData, err := json.Marshal(record)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.digitalocean.com/v2/domains/%s/records/%s", getParentDomain(domainConfig.Domain), domainConfig.RecordID)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+apiToken)
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

type DomainConfig struct {
	Domain   string
	RecordID string
}

type Config struct {
	APIToken      string
	Domains       []DomainConfig
	UpdateMinutes int
}

func loadEnvFile(filename string) (Config, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return Config{}, fmt.Errorf("error reading .env file: %v", err)
	}

	config := Config{
		UpdateMinutes: 5, // Default to 5 minutes if not specified
		Domains:       []DomainConfig{},
	}

	var domainsStr string

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
		case "DO_DOMAINS":
			domainsStr = value
		case "UPDATE_MINUTES":
			if minutes, err := strconv.Atoi(value); err == nil {
				config.UpdateMinutes = minutes
			}
		}
	}

	if config.APIToken == "" || domainsStr == "" {
		return config, fmt.Errorf("missing required configuration in .env file")
	}

	// Parse comma-separated domains
	domainList := strings.Split(domainsStr, ",")
	for _, domain := range domainList {
		domain = strings.TrimSpace(domain)
		if domain != "" {
			config.Domains = append(config.Domains, DomainConfig{
				Domain:   domain,
				RecordID: "",
			})
		}
	}

	if len(config.Domains) == 0 {
		return config, fmt.Errorf("no domains specified in .env file")
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

	// Get the record IDs for all domains
	log.Printf("Fetching record IDs for %d domain(s)...", len(config.Domains))
	for i := range config.Domains {
		recordID, err := getRecordID(config.APIToken, config.Domains[i].Domain)
		if err != nil {
			log.Fatalf("Failed to get record ID for domain %s: %v", config.Domains[i].Domain, err)
		}
		config.Domains[i].RecordID = recordID
		log.Printf("Domain: %s, Record ID: %s", config.Domains[i].Domain, config.Domains[i].RecordID)
	}

	log.Printf("Starting DNS updater for %d domain(s)", len(config.Domains))
	log.Printf("Update interval: %d minutes", config.UpdateMinutes)

	for {
		ip, err := getCurrentIP()
		if err != nil {
			log.Printf("Error getting current IP: %v", err)
			time.Sleep(time.Duration(config.UpdateMinutes) * time.Minute)
			continue
		}

		log.Printf("Current IP: %s", ip)

		// Update all domains with the same IP
		for _, domainConfig := range config.Domains {
			err = updateDNS(config.APIToken, domainConfig, ip)
			if err != nil {
				log.Printf("Error updating DNS for %s: %v", domainConfig.Domain, err)
			} else {
				log.Printf("Successfully updated DNS record for %s to IP %s", domainConfig.Domain, ip)
			}
		}

		time.Sleep(time.Duration(config.UpdateMinutes) * time.Minute)
	}
}
