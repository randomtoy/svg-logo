package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/randomtoy/svg-logo/internal/config"
	"github.com/randomtoy/svg-logo/internal/downloader"
)

type Flags struct {
	ConfigPath string
	Parallel   int
	Strict     bool
}

func main() {
	flags := parseFlags()

	cfg, err := config.Load(flags.ConfigPath)
	if err != nil {
		panic(err)
	}
	if cfg.Items == nil {
		panic("no items in config")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	type result struct {
		idx     int
		path    string
		status  string
		updated bool
		err     error
	}

	ch := make(chan int, len(cfg.Items))
	var wg sync.WaitGroup
	results := make([]result, len(cfg.Items))

	sem := make(chan struct{}, flags.Parallel)

	for i := range cfg.Items {
		ch <- i
	}
	close(ch)

	for i := 0; i < flags.Parallel; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range ch {
				item := cfg.Items[idx]
				sem <- struct{}{}
				updated, status, err := downloader.Download(ctx, item.Path, item.URL)
				results[idx] = result{
					idx:     idx,
					path:    item.Path,
					status:  status,
					updated: updated,
					err:     err,
				}
				<-sem
			}
		}()
	}
	wg.Wait()

	var failures int
	var updatedCount int

	fmt.Println()
	fmt.Println("===== Download Summary =====")

	for _, r := range results {
		if r.err != nil {
			failures++
			fmt.Printf("[FAIL]   %s\n", r.path)
			fmt.Printf("         Error:  %v\n", r.err)
			fmt.Printf("         Status: %s\n\n", r.status)
		} else {
			if r.updated {
				updatedCount++
				fmt.Printf("[OK]     %s\n", r.path)
				fmt.Printf("         Updated (%s)\n\n", r.status)
			} else {
				fmt.Printf("[SKIP]   %s\n", r.path)
				fmt.Printf("         Not modified (%s)\n\n", r.status)
			}
		}
	}
	fmt.Printf("===== Completed: %d updated, %d failed =====\n", updatedCount, failures)

	if failures > 0 && flags.Strict {
		os.Exit(1)
	}
}

func parseFlags() *Flags {
	cfgPath := flag.String("config", "config.yaml", "Path to configuration file")
	parallel := flag.Int("parallel", runtime.NumCPU(), "parallel downloads")
	strict := flag.Bool("strict", false, "fail if any download fails")
	flag.Parse()
	return &Flags{
		ConfigPath: *cfgPath,
		Parallel:   *parallel,
		Strict:     *strict,
	}

}
