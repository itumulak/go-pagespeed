package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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
	Page       WordPressPage
	Result     *PageSpeedResult
	Error      error
	RetryCount int
}

func fetchWordPressPages(wpURL string) ([]WordPressPage, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(wpURL)
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

func checkPageSpeedWithRetry(apiKey, pageURL string, maxRetries int, results chan<- PageResult, wg *sync.WaitGroup) {
	defer wg.Done()

	encodedURL := url.QueryEscape(pageURL)

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Use longer timeout for slower servers (EFS can cause delays)
		apiURL := fmt.Sprintf("https://www.googleapis.com/pagespeedonline/v5/runPagespeed?url=%s&key=%s&category=performance&category=accessibility&category=best-practices&category=seo", encodedURL, apiKey)

		client := &http.Client{Timeout: 60 * time.Second} // Increased timeout for slow EFS
		resp, err := client.Get(apiURL)

		if err != nil {
			if attempt < maxRetries {
				fmt.Printf("   ⏳ Attempt %d/%d failed for: %s (retrying in %d seconds)\n",
					attempt, maxRetries, pageURL, attempt*2)
				time.Sleep(time.Duration(attempt*2) * time.Second) // Exponential backoff
				continue
			}
			results <- PageResult{
				Page:       WordPressPage{Link: pageURL},
				Error:      fmt.Errorf("failed after %d attempts: %v", maxRetries, err),
				RetryCount: attempt,
			}
			return
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			if attempt < maxRetries {
				fmt.Printf("   ⏳ Attempt %d/%d failed to read response for: %s (retrying)\n",
					attempt, maxRetries, pageURL)
				time.Sleep(time.Duration(attempt*2) * time.Second)
				continue
			}
			results <- PageResult{
				Page:       WordPressPage{Link: pageURL},
				Error:      fmt.Errorf("failed to read response after %d attempts: %v", maxRetries, err),
				RetryCount: attempt,
			}
			return
		}

		var result PageSpeedResult
		err = json.Unmarshal(body, &result)

		// Check if the response contains an error from Google's API
		var errorResponse map[string]interface{}
		json.Unmarshal(body, &errorResponse)
		if errorResp, ok := errorResponse["error"]; ok {
			err = fmt.Errorf("Google API error: %v", errorResp)

			if attempt < maxRetries {
				fmt.Printf("   ⏳ Attempt %d/%d for %s: %v (retrying)\n",
					attempt, maxRetries, pageURL, err)

				// For certain errors, wait longer
				waitTime := time.Duration(attempt*3) * time.Second
				if strings.Contains(err.Error(), "TIMED_OUT") || strings.Contains(err.Error(), "FAILED_DOCUMENT_REQUEST") {
					waitTime = time.Duration(attempt*5) * time.Second
					fmt.Printf("   🔄 Detected timeout, waiting longer: %d seconds\n", waitTime/time.Second)
				}

				time.Sleep(waitTime)
				continue
			}

			results <- PageResult{
				Page:       WordPressPage{Link: pageURL},
				Error:      err,
				RetryCount: attempt,
			}
			return
		}

		// Success!
		results <- PageResult{
			Page:       WordPressPage{Link: pageURL},
			Result:     &result,
			Error:      nil,
			RetryCount: attempt,
		}
		return
	}
}

func displayResult(result PageResult) {
	// Find the actual page title if available
	title := result.Page.Title.Rendered
	if title == "" && result.Page.Link != "" {
		title = "Unknown Page"
	}

	fmt.Printf("📄 %s\n", title)
	fmt.Printf("   🔗 %s\n", result.Page.Link)

	if result.Error != nil {
		fmt.Printf("   ❌ Error after %d attempts: %v\n", result.RetryCount, result.Error)
	} else if result.Result != nil {
		perf := result.Result.LighthouseResult.Categories.Performance.Score
		access := result.Result.LighthouseResult.Categories.Accessibility.Score
		bp := result.Result.LighthouseResult.Categories.BestPractices.Score
		seo := result.Result.LighthouseResult.Categories.SEO.Score

		// Get Core Web Vitals
		fid := result.Result.LoadingExperience.Metrics.FirstInputDelay.Percentile
		fcp := result.Result.LoadingExperience.Metrics.FirstContentfulPaint.Percentile

		fmt.Printf("   🎯 Performance Score: %.0f/100\n", perf*100)
		fmt.Printf("   ♿ Accessibility: %.0f/100\n", access*100)
		fmt.Printf("   ✨ Best Practices: %.0f/100\n", bp*100)
		fmt.Printf("   🔍 SEO: %.0f/100\n", seo*100)

		if fid > 0 || fcp > 0 {
			fmt.Printf("   📊 Core Web Vitals:\n")
			if fid > 0 {
				fmt.Printf("      • First Input Delay: %.0f ms\n", fid)
			}
			if fcp > 0 {
				fmt.Printf("      • First Contentful Paint: %.0f ms\n", fcp)
			}
		}

		if result.RetryCount > 1 {
			fmt.Printf("   ✅ Succeeded after %d attempts\n", result.RetryCount)
		}
	}
	fmt.Println()
}

func getScoreEmoji(score float64) string {
	if score >= 0.9 {
		return "🟢"
	} else if score >= 0.5 {
		return "🟠"
	}
	return "🔴"
}

func main() {
	apiKey := "AIzaSyBub-bWKfnlrAWOhoPYazPclJsKZBBjguQ"
	wpURL := "http://staging.avocadova.com/wp-json/wp/v2/pages"
	maxRetries := 3

	fmt.Println("🚀 WordPress PageSpeed Analyzer")
	fmt.Println("================================")
	fmt.Println("Fetching WordPress pages...")

	pages, err := fetchWordPressPages(wpURL)
	if err != nil {
		fmt.Printf("❌ Error fetching WordPress pages: %v\n", err)
		return
	}

	fmt.Printf("✅ Found %d pages\n", len(pages))
	fmt.Println("Analyzing with PageSpeed Insights (up to 3 retries per page)...")

	// Create channels and wait group
	results := make(chan PageResult, len(pages))
	var wg sync.WaitGroup

	// Track start time
	startTime := time.Now()

	// Launch goroutines for each page
	for _, page := range pages {
		wg.Add(1)
		go checkPageSpeedWithRetry(apiKey, page.Link, maxRetries, results, &wg)
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results as they come in and display immediately
	displayedCount := 0
	for result := range results {
		// Find the full page data to get title
		for _, page := range pages {
			if page.Link == result.Page.Link {
				result.Page = page
				break
			}
		}
		displayResult(result)
		displayedCount++
	}

	// Summary
	elapsed := time.Since(startTime)
	fmt.Println("================================")
	fmt.Printf("📊 Summary: Completed %d/%d pages in %s\n", displayedCount, len(pages), elapsed)
}
