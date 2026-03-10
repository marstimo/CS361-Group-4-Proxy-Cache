package cmd

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-fuego/fuego"
	"github.com/marstimo/CS361-Group-4-Proxy-Cache/service/cache"
	"github.com/spf13/cobra"
)

var (
	port   int
	apiKey string
)

func authMiddleware(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			k := r.Header.Get("X-API-Key")
			if k == "" {
				k = r.URL.Query().Get("api_key")
			}
			if k != key {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func copyHeaders(w http.ResponseWriter, headers http.Header) {
	for k, vals := range headers {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
}

func fetchOrigin(targetURL string) ([]byte, http.Header, string, error) {
	resp, err := http.Get(targetURL)
	if err != nil {
		return nil, nil, "", fmt.Errorf("Failed to fetch origin: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, "", fmt.Errorf("Failed to read origin response")
	}

	cc := resp.Header.Get("Cache-Control")
	return body, resp.Header, cc, nil
}

func newServer(c *cache.Cache, key string, opts ...func(*fuego.Server)) *fuego.Server {
	s := fuego.NewServer(opts...)
	fuego.Use(s, authMiddleware(key))

	fuego.Get(s, "/proxy", func(ctx fuego.ContextNoBody) (any, error) {
		w := ctx.Response()
		r := ctx.Request()

		targetURL := r.URL.Query().Get("url")
		if targetURL == "" {
			http.Error(w, "Missing url parameter", http.StatusBadRequest)
			return nil, nil
		}

		if entry, ok := c.Get(targetURL); ok {
			copyHeaders(w, entry.Headers)
			remaining := time.Until(entry.Expiry).Seconds()
			w.Header().Set("X-Proxy-Cache", "HIT")
			w.Header().Set("X-Cache-TTL-Remaining", strconv.Itoa(int(remaining)))
			w.WriteHeader(http.StatusOK)
			w.Write(entry.Body)
			return nil, nil
		}

		body, headers, cc, err := fetchOrigin(targetURL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return nil, nil
		}

		maxAge, shouldStore := cache.ParseCacheControl(cc)
		if shouldStore {
			c.Set(targetURL, &cache.Entry{
				Body:    body,
				Headers: headers.Clone(),
				Expiry:  time.Now().Add(time.Duration(maxAge) * time.Second),
			})
		}

		copyHeaders(w, headers)
		w.Header().Set("X-Proxy-Cache", "MISS")
		w.WriteHeader(http.StatusOK)
		w.Write(body)
		return nil, nil
	})

	fuego.Delete(s, "/cache", func(ctx fuego.ContextNoBody) (any, error) {
		w := ctx.Response()
		r := ctx.Request()

		targetURL := r.URL.Query().Get("url")
		if targetURL == "" {
			http.Error(w, "Missing url parameter", http.StatusBadRequest)
			return nil, nil
		}

		if c.Delete(targetURL) {
			w.WriteHeader(http.StatusNoContent)
		} else {
			http.Error(w, "Not found", http.StatusNotFound)
		}
		return nil, nil
	})

	return s
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the proxy cache server",
	RunE: func(cmd *cobra.Command, args []string) error {
		s := newServer(cache.New(), apiKey, fuego.WithAddr(fmt.Sprintf(":%d", port)))
		log.Printf("Proxy cache listening on :%d", port)
		return s.Run()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().IntVarP(&port, "port", "p", 8080, "port to listen on")
	serveCmd.Flags().StringVarP(&apiKey, "api-key", "k", "default-api-key", "API key for authentication")
}
