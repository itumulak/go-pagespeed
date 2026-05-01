package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type WordPressPage struct {
	ID    int `json:"id"`
	Title struct {
		Rendered string `json:"rendered"`
	} `json:"title"`
	Link string `json:"link"`
	Slug string `json:"slug"`
}

type PageSpeedResult struct {
	LighthouseResult struct {
		Categories struct {
			Performance struct {
				Score float64 `json:"score"`
			} `json:"performance"`
			Accessibility struct {
				Score float64 `json:"score"`
			} `json:"accessibility"`
			BestPractices struct {
				Score float64 `json:"best-practices"`
			} `json:"best-practices"`
			SEO struct {
				Score float64 `json:"seo"`
			} `json:"seo"`
		} `json:"categories"`
	} `json:"lighthouseResult"`
	LoadingExperience struct {
		Metrics struct {
			FirstInputDelay struct {
				Percentile float64 `json:"percentile"`
			} `json:"FIRST_INPUT_DELAY_MS"`
			FirstContentfulPaint struct {
				Percentile float64 `json:"percentile"`
			} `json:"FIRST_CONTENTFUL_PAINT_MS"`
		} `json:"metrics"`
	} `json:"loadingExperience"`
}

type PageResult struct {
	Page   WordPressPage
	Result *PageSpeedResult
	Error  error
}

func fetchWordPressPages(wpURL string) ([]WordPressPage, error) {
	resp, err := http.Get(wpURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var pages []WordPressPage
	err = json.Unmarshal(body, &pages)
	return pages, err
}

func checkPageSpeed(apiKey, pageURL string, wg *sync.WaitGroup, results chan<- PageResult) {
	defer wg.Done()

	encodedURL := url.QueryEscape(pageURL)
	apiURL := fmt.Sprintf("https://www.googleapis.com/pagespeedonline/v5/runPagespeed?url=%s&key=%s", encodedURL, apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		results <- PageResult{Error: err}
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		results <- PageResult{Error: err}
		return
	}

	var result PageSpeedResult
	err = json.Unmarshal(body, &result)

	// We need to match the page with its result
	// For simplicity, we'll create a temporary struct
	results <- PageResult{Result: &result, Error: err}
}

func main() {
	apiKey := "AIzaSyBub-bWKfnlrAWOhoPYazPclJsKZBBjguQ"
	wpURL := "http://staging.avocadova.com/wp-json/wp/v2/pages"

	fmt.Println("Fetching WordPress page...")
	pages, err := fetchWordPressPages(wpURL)
	if err != nil {
		fmt.Printf("Error fetching WordPress pages: %v\n", err)
		return
	}

	fmt.Printf("Found %d pages. Analyzing with PageSpeed...\n\n", len(pages))

	var wg sync.WaitGroup
	results := make(chan PageResult, len(pages))

	for _, page := range pages {
		wg.Add(1)
		go checkPageSpeed(apiKey, page.Link, &wg, results)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	resultsList := make([]PageResult, 0, len(pages))
	for res := range results {
		resultsList = append(resultsList, res)
	}

	// Display results
	for i, page := range pages {
		fmt.Printf("%d. %s\n", i+1, page.Title.Rendered)
		fmt.Printf("   URL: %s\n", page.Link)
		if i < len(resultsList) && resultsList[i].Result != nil {
			perf := resultsList[i].Result.LighthouseResult.Categories.Performance.Score
			fmt.Printf("   Performance Score: %.0f/100\n\n", perf*100)
		} else {
			fmt.Printf("   Error analyzing page\n\n")
		}
	}
}
