package watcher

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"

	appbuyer "github.com/sieryo/invoice-extractor/internal/app/buyer"
	"github.com/sieryo/invoice-extractor/internal/infra/storage"
)

type CSVWatcher struct {
	registry *appbuyer.Registry
	store    *storage.BuyerCSVStore
	csvPath  string
}

func NewCSVWatcher(
	registry *appbuyer.Registry,
	store *storage.BuyerCSVStore,
	csvPath string,
) *CSVWatcher {
	return &CSVWatcher{
		registry: registry,
		store:    store,
		csvPath:  csvPath,
	}
}

func (w *CSVWatcher) Run(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	dir := filepath.Dir(w.csvPath)

	if err := watcher.Add(dir); err != nil {
		return err
	}

	if err := w.registry.ReloadFromCSV(w.store, w.csvPath); err != nil {
		fmt.Printf("Error from reload from csv: %s", err.Error())
	}

	debounce := time.Now()

	for {
		select {
		case evt := <-watcher.Events:
			if evt.Name == w.csvPath &&
				(evt.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) != 0) {

				// debounce biar gak reload berkali-kali
				if time.Since(debounce) < 300*time.Millisecond {
					continue
				}
				debounce = time.Now()

				if err := retryReload(func() error {
					return w.registry.ReloadFromCSV(w.store, w.csvPath)
				}, 3, 100*time.Millisecond); err != nil {
					slog.Error("failed to reload buyer csv", "err", err)
				}

			}

		case err := <-watcher.Errors:
			slog.Error("csv watcher error", "err", err)

		case <-ctx.Done():
			slog.Info("csv watcher stopped")
			return nil
		}
	}
}

func retryReload(
	fn func() error,
	attempts int,
	delay time.Duration,
) error {
	var err error
	for range attempts {
		if err = fn(); err == nil {
			return nil
		}
		time.Sleep(delay)
	}
	return err
}
