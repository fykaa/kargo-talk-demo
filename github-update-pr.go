// proves Update API ≠ Merge Queue
package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

func main() {
	prNumber := flag.Int("pr", 0, "PR number")
	token := flag.String("token", "", "GitHub token")
	owner := flag.String("owner", "", "repo owner")
	repo := flag.String("repo", "", "repo name")
	flag.Parse()

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: *token})
	client := github.NewClient(oauth2.NewClient(ctx, ts))

	// Update PR (common operations: title, body, state, etc.)
	updatedPR, resp, err := client.PullRequests.Edit(ctx, *owner, *repo, *prNumber, &github.PullRequest{
		Title: github.String("Updated title - testing merge queue"),
		State: github.String("open"),
	})

	if err != nil {
		log.Fatalf("Update failed: %v (status: %d)", err, resp.StatusCode)
	}

	fmt.Printf("=== UPDATE PR RESULT ===\n")
	fmt.Printf("HTTP STATUS: %d ← ALWAYS 200, NEVER 202!\n", resp.StatusCode)
	fmt.Printf("State: %q\n", updatedPR.GetState())
	fmt.Printf("Auto-merge: %v\n", updatedPR.GetAutoMerge())
	// NO merge queue status here 
}
