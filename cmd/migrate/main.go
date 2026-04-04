package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gauravprasad/clawcontrol/internal/app"
	"github.com/gauravprasad/clawcontrol/internal/platform/database"
)

func main() {
	if err := app.LoadDotEnv(".env"); err != nil {
		log.Fatal(err)
	}
	cfg := app.LoadConfig()
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	db, err := database.Open(context.Background(), database.Config{
		URL:             cfg.DatabaseURL,
		MaxOpenConns:    2,
		MaxIdleConns:    1,
		ConnMaxLifetime: cfg.DBConnMaxLifetime,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	entries, err := os.ReadDir("migrations")
	if err != nil {
		log.Fatal(err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) == ".sql" {
			files = append(files, filepath.Join("migrations", entry.Name()))
		}
	}
	sort.Strings(files)

	for _, path := range files {
		sqlBytes, err := fs.ReadFile(os.DirFS("."), path)
		if err != nil {
			log.Fatal(err)
		}
		stmt := strings.TrimSpace(string(sqlBytes))
		if stmt == "" {
			continue
		}
		if _, err := db.ExecContext(context.Background(), stmt); err != nil {
			log.Fatalf("apply %s: %v", path, err)
		}
		fmt.Printf("applied %s\n", path)
	}
}
