package external

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// RedditClient defines the interface for Reddit API operations
type RedditClient interface {
	SearchPostGameThread(team1, team2 string, gameDate time.Time) ([]RedditPost, error)
	GetThreadComments(threadURL string) ([]RedditComment, error)
}

// HTTPRedditClient implements RedditClient using HTTP requests
type HTTPRedditClient struct {
	client    *http.Client
	userAgent string
}

// NewHTTPRedditClient creates a new HTTP-based Reddit client
func NewHTTPRedditClient(userAgent string) *HTTPRedditClient {
	if userAgent == "" {
		userAgent = "nfl-app/1.0 by /u/nfl-app-user"
	}
	return &HTTPRedditClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		userAgent: userAgent,
	}
}

// RedditPost represents a Reddit post/thread
type RedditPost struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Author    string    `json:"author"`
	URL       string    `json:"url"`
	Permalink string    `json:"permalink"`
	Created   float64   `json:"created_utc"`
	Score     int       `json:"score"`
	NumComments int     `json:"num_comments"`
}

// RedditComment represents a top-level comment from a Reddit thread
type RedditComment struct {
	ID      string  `json:"id"`
	Author  string  `json:"author"`
	Body    string  `json:"body"`
	Score   int     `json:"score"`
	Created float64 `json:"created_utc"`
}

// RedditSearchResponse represents the response from Reddit search API
type RedditSearchResponse struct {
	Kind string `json:"kind"`
	Data struct {
		Children []struct {
			Kind string `json:"kind"`
			Data RedditPost `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

// RedditThreadResponse represents the response from Reddit thread API
type RedditThreadResponse []struct {
	Kind string `json:"kind"`
	Data struct {
		Children []struct {
			Kind string `json:"kind"`
			Data json.RawMessage `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

// SearchPostGameThread searches for post-game threads on /r/nfl
// gameDate is used to determine the appropriate search time window
func (c *HTTPRedditClient) SearchPostGameThread(team1, team2 string, gameDate time.Time) ([]RedditPost, error) {
	// Build search query using title filter for better accuracy
	// Format: title:"Post Game Thread" team1 team2
	query := fmt.Sprintf(`title:"Post Game Thread" %s %s`, team1, team2)

	// Determine time window based on game date
	daysSinceGame := int(time.Since(gameDate).Hours() / 24)
	var timeWindow string
	switch {
	case daysSinceGame <= 7:
		timeWindow = "week"
	case daysSinceGame <= 30:
		timeWindow = "month"
	case daysSinceGame <= 365:
		timeWindow = "year"
	default:
		timeWindow = "all"
	}

	// Build URL with query parameters
	baseURL := "https://old.reddit.com/r/nfl/search.json"
	params := url.Values{}
	params.Add("q", query)
	params.Add("restrict_sr", "on")
	params.Add("sort", "new")
	params.Add("t", timeWindow)
	params.Add("limit", "10")

	searchURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// Make request with backoff
	var resp *http.Response
	var err error
	maxRetries := 3

	for i := 0; i < maxRetries; i++ {
		resp, err = c.makeRequest(searchURL)
		if err != nil {
			return nil, fmt.Errorf("request failed: %w", err)
		}

		// Check for rate limiting
		if resp.StatusCode == 429 {
			resp.Body.Close()
			waitTime := time.Duration(2<<uint(i)) * time.Second // Exponential backoff: 2s, 4s, 8s
			time.Sleep(waitTime)
			continue
		}

		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
		}

		break
	}

	if resp == nil || resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed after %d retries", maxRetries)
	}

	defer resp.Body.Close()

	var searchResp RedditSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract posts
	posts := make([]RedditPost, 0, len(searchResp.Data.Children))
	for _, child := range searchResp.Data.Children {
		posts = append(posts, child.Data)
	}

	return posts, nil
}

// GetThreadComments fetches top-level comments from a Reddit thread
func (c *HTTPRedditClient) GetThreadComments(threadURL string) ([]RedditComment, error) {
	// Convert regular Reddit URL to JSON API URL
	jsonURL := c.convertToJSONURL(threadURL)

	// Add query parameters for sorting and limiting
	if !strings.Contains(jsonURL, "?") {
		jsonURL += "?"
	} else {
		jsonURL += "&"
	}
	jsonURL += "sort=top&limit=100"

	// Make request with backoff
	var resp *http.Response
	var err error
	maxRetries := 3

	for i := 0; i < maxRetries; i++ {
		resp, err = c.makeRequest(jsonURL)
		if err != nil {
			return nil, fmt.Errorf("request failed: %w", err)
		}

		// Check for rate limiting
		if resp.StatusCode == 429 {
			resp.Body.Close()
			waitTime := time.Duration(2<<uint(i)) * time.Second // Exponential backoff
			time.Sleep(waitTime)
			continue
		}

		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
		}

		break
	}

	if resp == nil || resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed after %d retries", maxRetries)
	}

	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var threadResp RedditThreadResponse
	if err := json.Unmarshal(body, &threadResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// The response contains two elements: [0] is the thread, [1] is the comments
	if len(threadResp) < 2 {
		return nil, fmt.Errorf("unexpected response format: expected 2 elements, got %d", len(threadResp))
	}

	// Extract top-level comments only (exclude replies)
	comments := make([]RedditComment, 0)
	for _, child := range threadResp[1].Data.Children {
		if child.Kind != "t1" { // t1 is a comment, "more" indicates there are more comments
			continue
		}

		var comment RedditComment
		if err := json.Unmarshal(child.Data, &comment); err != nil {
			// Skip comments that fail to parse
			continue
		}

		// Only include top-level comments (not replies)
		// We can check if it's top-level by seeing if it has a parent_id starting with t3_ (link/post)
		var fullData struct {
			ParentID string `json:"parent_id"`
		}
		if err := json.Unmarshal(child.Data, &fullData); err == nil {
			if strings.HasPrefix(fullData.ParentID, "t3_") {
				comments = append(comments, comment)
			}
		}
	}

	return comments, nil
}

// makeRequest creates and executes an HTTP request with proper headers
func (c *HTTPRedditClient) makeRequest(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Set User-Agent header (required by Reddit API)
	req.Header.Set("User-Agent", c.userAgent)

	return c.client.Do(req)
}

// convertToJSONURL converts a regular Reddit URL to its JSON API equivalent
func (c *HTTPRedditClient) convertToJSONURL(threadURL string) string {
	// Handle different URL formats
	threadURL = strings.TrimSpace(threadURL)

	// If it already ends with .json, return as-is
	if strings.HasSuffix(threadURL, ".json") {
		return threadURL
	}

	// Remove trailing slash if present
	threadURL = strings.TrimSuffix(threadURL, "/")

	// Convert www.reddit.com or reddit.com to old.reddit.com
	// Handle www.reddit.com first, then plain reddit.com (but not if it's already old.reddit.com)
	if strings.Contains(threadURL, "www.reddit.com") {
		threadURL = strings.Replace(threadURL, "www.reddit.com", "old.reddit.com", 1)
	} else if strings.Contains(threadURL, "reddit.com") && !strings.Contains(threadURL, "old.reddit.com") {
		threadURL = strings.Replace(threadURL, "reddit.com", "old.reddit.com", 1)
	}

	// Add .json extension
	return threadURL + ".json"
}
