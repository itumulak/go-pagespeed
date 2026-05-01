package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type WordPressPage struct {
	Link string `json:"link"`
}

func main() {
	// Fetch pages from WordPress API
	url := "http://localhost:8000/wp-json/wp/v2/pages"

	resp, err := http.Get(url)

	if err != nil {
		fmt.Printf("Error fetching URL: %v\n", err)
		return
	}

	defer resp.Body.Close()

	// Check if status is OK
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("HTTP Error: %s\n", resp.Status)
		return
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	// Parse JSON
	var pages []WordPressPage

	err = json.Unmarshal(body, &pages)
	if err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		return
	}

	// Just display links only (simpler version)
	fmt.Println("=== LINKS ONLY ===")
	for _, page := range pages {
		fmt.Printf("%s\n", page.Link)
	}
}
