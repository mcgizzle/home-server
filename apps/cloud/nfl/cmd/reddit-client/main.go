package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/mcgizzle/home-server/apps/cloud/internal/external"
)

func main() {
	var (
		search    = flag.String("search", "", "Search for post-game thread (e.g., 'rams seahawks')")
		thread    = flag.String("thread", "", "Fetch comments from a thread URL")
		export    = flag.String("export", "", "Export comments as JSON to specified file path")
		dateStr   = flag.String("date", "", "Game date for search (YYYY-MM-DD), defaults to today")
		userAgent = flag.String("user-agent", "nfl-app/1.0 by /u/nfl-app-user", "User-Agent header for Reddit API")
		pretty    = flag.Bool("pretty", true, "Pretty print JSON output")
		help      = flag.Bool("help", false, "Show help")
	)
	flag.Parse()

	if *help {
		printHelp()
		return
	}

	// Create Reddit client
	client := external.NewHTTPRedditClient(*userAgent)

	// Parse date if provided
	gameDate := time.Now()
	if *dateStr != "" {
		parsed, err := time.Parse("2006-01-02", *dateStr)
		if err != nil {
			log.Fatalf("Invalid date format (use YYYY-MM-DD): %v", err)
		}
		gameDate = parsed
	}

	// Determine which command to run
	if *search != "" {
		searchPostGameThread(client, *search, gameDate, *pretty)
	} else if *thread != "" {
		fetchThreadComments(client, *thread, *export, *pretty)
	} else {
		fmt.Println("Error: must specify either -search or -thread")
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("Reddit Client - Fetch NFL post-game threads and comments from /r/nfl")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  reddit-client [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -search <teams>        Search for post-game thread (e.g., 'rams seahawks')")
	fmt.Println("  -thread <url>          Fetch comments from a thread URL")
	fmt.Println("  -export <path>         Export comments as JSON to file (use with -thread)")
	fmt.Println("  -user-agent <string>   Custom User-Agent header (default: 'nfl-app/1.0 by /u/nfl-app-user')")
	fmt.Println("  -pretty                Pretty print JSON output (default: true)")
	fmt.Println("  -help                  Show this help")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Search for Rams vs Seahawks post-game thread")
	fmt.Println("  reddit-client -search \"rams seahawks\"")
	fmt.Println()
	fmt.Println("  # Fetch comments from a specific thread")
	fmt.Println("  reddit-client -thread \"https://www.reddit.com/r/nfl/comments/abc123/post_game_thread/\"")
	fmt.Println()
	fmt.Println("  # Fetch and export comments to JSON file")
	fmt.Println("  reddit-client -thread \"https://www.reddit.com/r/nfl/comments/abc123/\" -export comments.json")
}

func printJSON(data interface{}, pretty bool) {
	var output []byte
	var err error

	if pretty {
		output, err = json.MarshalIndent(data, "", "  ")
	} else {
		output, err = json.Marshal(data)
	}

	if err != nil {
		log.Printf("Error marshaling JSON: %v", err)
		return
	}

	fmt.Println(string(output))
}

func searchPostGameThread(client *external.HTTPRedditClient, query string, gameDate time.Time, pretty bool) {
	fmt.Printf("Searching for post-game thread: %s (date: %s)\n", query, gameDate.Format("2006-01-02"))

	// Split query into teams (assuming space-separated)
	parts := strings.Fields(query)
	if len(parts) < 2 {
		fmt.Println("Warning: search query should contain at least two team names")
	}

	team1 := ""
	team2 := ""
	if len(parts) >= 2 {
		team1 = parts[0]
		team2 = strings.Join(parts[1:], " ")
	} else if len(parts) == 1 {
		team1 = parts[0]
	}

	posts, err := client.SearchPostGameThread(team1, team2, gameDate)
	if err != nil {
		log.Fatalf("Error searching for post-game thread: %v", err)
	}

	fmt.Printf("\nFound %d post(s):\n\n", len(posts))

	for i, post := range posts {
		fmt.Printf("=== Post %d ===\n", i+1)
		fmt.Printf("Title: %s\n", post.Title)
		fmt.Printf("Author: %s\n", post.Author)
		fmt.Printf("Score: %d\n", post.Score)
		fmt.Printf("Comments: %d\n", post.NumComments)
		fmt.Printf("Created: %s\n", formatTimestamp(post.Created))
		fmt.Printf("URL: https://www.reddit.com%s\n", post.Permalink)
		fmt.Println()
	}

	if len(posts) > 0 {
		fmt.Println("To fetch comments from a thread, use:")
		fmt.Printf("  reddit-client -thread \"https://www.reddit.com%s\"\n", posts[0].Permalink)
		fmt.Println()
	}

	fmt.Println("Raw JSON:")
	printJSON(posts, pretty)
}

func fetchThreadComments(client *external.HTTPRedditClient, threadURL string, exportPath string, pretty bool) {
	fmt.Printf("Fetching comments from: %s\n", threadURL)

	comments, err := client.GetThreadComments(threadURL)
	if err != nil {
		log.Fatalf("Error fetching thread comments: %v", err)
	}

	fmt.Printf("\nFound %d top-level comment(s):\n\n", len(comments))

	// Display first 5 comments as preview
	previewCount := 5
	if len(comments) < previewCount {
		previewCount = len(comments)
	}

	for i := 0; i < previewCount; i++ {
		comment := comments[i]
		fmt.Printf("=== Comment %d ===\n", i+1)
		fmt.Printf("Author: %s\n", comment.Author)
		fmt.Printf("Score: %d\n", comment.Score)
		fmt.Printf("Created: %s\n", formatTimestamp(comment.Created))
		fmt.Printf("Body: %s\n", truncateString(comment.Body, 200))
		fmt.Println()
	}

	if len(comments) > previewCount {
		fmt.Printf("... and %d more comment(s)\n\n", len(comments)-previewCount)
	}

	// Export if path is specified
	if exportPath != "" {
		if err := exportComments(comments, exportPath, pretty); err != nil {
			log.Fatalf("Error exporting comments: %v", err)
		}
		fmt.Printf("Comments exported to: %s\n", exportPath)
	} else {
		fmt.Println("Full JSON output:")
		printJSON(comments, pretty)
	}
}

func exportComments(comments []external.RedditComment, path string, pretty bool) error {
	var data []byte
	var err error

	if pretty {
		data, err = json.MarshalIndent(comments, "", "  ")
	} else {
		data, err = json.Marshal(comments)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal comments: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func formatTimestamp(unixTime float64) string {
	t := time.Unix(int64(unixTime), 0)
	return t.Format("2006-01-02 15:04:05 MST")
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
