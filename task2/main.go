package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"mime"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"
)

type Movie struct {
	Title    string
	Year     int
	Director string
}

type movieResponse struct {
	Title    *string `json:"title"`
	Year     *int    `json:"year"`
	Director *string `json:"director"`
}

func fetchMovie(id int, ctx context.Context, timeout time.Duration) (Movie, error) {
	url := fmt.Sprintf("https://homeworksite.site/%d/info.0.json", id)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)

	if err != nil {
		return Movie{}, err
	}

	client := http.Client{
		Timeout: timeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return Movie{}, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return Movie{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return Movie{}, fmt.Errorf("invalid Content-Type: %w", err)
	}

	if mediaType != "application/json" {
		return Movie{}, fmt.Errorf("unexpected Content-Type: %s", contentType)
	}

	var data movieResponse

	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return Movie{}, fmt.Errorf("failed to decode JSON: %w", err)
	}

	if data.Title == nil || data.Year == nil || data.Director == nil {
		return Movie{}, fmt.Errorf("response is missing required fields")
	}

	movie := Movie{
		Title:    *data.Title,
		Year:     *data.Year,
		Director: *data.Director,
	}

	return movie, nil
}

func worker(jobs <-chan int, ctx context.Context, timeout time.Duration) {
	for {
		select {
		case <-ctx.Done():
			return
		case id, ok := <-jobs:
			if !ok {
				return
			}

			movie, err := fetchMovie(id, ctx, timeout)

			if err != nil {
				if ctx.Err() != nil {
					return
				}

				fmt.Printf("%d - error : %v\n", id, err)
				continue
			}

			fmt.Printf("%d - %s - %d - %s\n",
				id,
				movie.Title,
				movie.Year,
				movie.Director)
		}
	}
}

func main() {
	from := flag.Int("from", 0, "first movie id")
	to := flag.Int("to", 0, "last movie id")
	workers := flag.Int("workers", 10, "workers count")
	timeout := flag.Duration("timeout", 5*time.Second, "HTTP request timeout")

	flag.Parse()

	provided := map[string]bool{}

	flag.Visit(func(f *flag.Flag) {
		provided[f.Name] = true
	})

	if !provided["from"] || !provided["to"] {
		_, _ = fmt.Fprintln(os.Stderr, "error: --from and --to are required")
		os.Exit(1)
	}

	if *from > *to {
		_, _ = fmt.Fprintln(os.Stderr, "error: --from must be less than or equal to --to")
		os.Exit(1)
	}

	if *workers <= 0 {
		_, _ = fmt.Fprintln(os.Stderr, "error: --workers must be greater than 0")
		os.Exit(1)
	}

	if *timeout <= 0 {
		_, _ = fmt.Fprintln(os.Stderr, "error: --timeout must be greater than 0")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	var wg sync.WaitGroup
	wg.Add(*workers)

	jobs := make(chan int, *workers)

	for i := 0; i < *workers; i++ {
		go func() {
			defer wg.Done()
			worker(jobs, ctx, *timeout)
		}()
	}

sendLoop:
	for id := *from; id <= *to; id++ {
		select {
		case <-ctx.Done():
			break sendLoop
		case jobs <- id:
		}
	}

	close(jobs)
	wg.Wait()
}
