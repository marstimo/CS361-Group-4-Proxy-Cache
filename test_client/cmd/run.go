package cmd

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

var (
	proxyAddr string
	apiKey    string
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run example requests against the proxy cache",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := &http.Client{Timeout: 10 * time.Second}

		testURL := "https://jsonplaceholder.typicode.com/posts/1"

		// Request without API key
		fmt.Println("=== Test 1: No API key ===")
		resp, err := doGet(client, proxyAddr+"/proxy?url="+testURL, "")
		if err != nil {
			return fmt.Errorf("test 1 failed: %w", err)
		}
		printResponse(resp)

		// Test 2: First request
		fmt.Println("\n=== Test 2: First request ===")
		resp, err = doGet(client, proxyAddr+"/proxy?url="+testURL, apiKey)
		if err != nil {
			return fmt.Errorf("test 2 failed: %w", err)
		}
		printResponse(resp)

		// Test 3: Second request
		fmt.Println("\n=== Test 3: Second request ===")
		resp, err = doGet(client, proxyAddr+"/proxy?url="+testURL, apiKey)
		if err != nil {
			return fmt.Errorf("test 3 failed: %w", err)
		}
		printResponse(resp)

		// Test 4: Delete cached entry
		fmt.Println("\n=== Test 4: Delete cache entry ===")
		resp, err = doDelete(client, proxyAddr+"/cache?url="+testURL, apiKey)
		if err != nil {
			return fmt.Errorf("test 4 failed: %w", err)
		}
		printResponse(resp)

		// Test 5: Delete non-existent entry
		fmt.Println("\n=== Test 5: Delete non-existent entry ===")
		resp, err = doDelete(client, proxyAddr+"/cache?url=https://example.com/nonexistent", apiKey)
		if err != nil {
			return fmt.Errorf("test 5 failed: %w", err)
		}
		printResponse(resp)

		// Test 6: Request after purge
		fmt.Println("\n=== Test 6: Request after purge ===")
		resp, err = doGet(client, proxyAddr+"/proxy?url="+testURL, apiKey)
		if err != nil {
			return fmt.Errorf("test 6 failed: %w", err)
		}
		printResponse(resp)

		return nil
	},
}

func doGet(client *http.Client, url, key string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if key != "" {
		req.Header.Set("X-API-Key", key)
	}
	return client.Do(req)
}

func doDelete(client *http.Client, url, key string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return nil, err
	}
	if key != "" {
		req.Header.Set("X-API-Key", key)
	}
	return client.Do(req)
}

func printResponse(resp *http.Response) {
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	fmt.Printf("Status: %d\n", resp.StatusCode)
	if v := resp.Header.Get("X-Proxy-Cache"); v != "" {
		fmt.Printf("X-Proxy-Cache: %s\n", v)
	}
	if v := resp.Header.Get("X-Cache-TTL-Remaining"); v != "" {
		fmt.Printf("X-Cache-TTL-Remaining: %s\n", v)
	}
	if len(body) > 200 {
		fmt.Printf("Body: %s...\n", body[:200])
	} else {
		fmt.Printf("Body: %s\n", body)
	}
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVar(&proxyAddr, "proxy", "http://localhost:8080", "proxy cache address")
	runCmd.Flags().StringVarP(&apiKey, "api-key", "k", "default-api-key", "API key for authentication")
}
