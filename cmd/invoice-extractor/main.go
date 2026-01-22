package main

import (
	"log"

	"github.com/sieryo/invoice-extractor/internal/database"
)

func main() {
	db := database.NewSQLite("data/app.db")
	defer db.Close()

	if err := database.RunMigration(db, "migrations/001_init.sql"); err != nil {
		log.Fatal(err)
	}

	// app := app.New(db)
}

// func main() {

// 	db := database.NewSQLite("data/app.db")
// 	defer db.Close()

// 	if err := database.RunMigration(db, "migrations/001_init.sql"); err != nil {
// 		log.Fatal(err)
// 	}

// 	log.Println("SQLite connected")
// 	ctx := context.Background()

// 	pdfPath := filepath.Join("assets", "pdf", "TEST.pdf")

// 	text, err := pdftool.ExtractText(
// 		ctx,
// 		pdfPath,
// 		pdftool.DefaultOptions(),
// 	)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	fmt.Println("=== PDF TEXT ===")

// 	fmt.Println(text)

// 	// template := seamakeup.SeaMakeupTemplate{}

// 	// if template.Match(text) {
// 	// 	normalized, err := template.Normalize(text)
// 	// 	if err != nil {
// 	// 		log.Fatal(err)
// 	// 	}
// 	// 	fmt.Println("=== NORMALIZED ===")
// 	// 	fmt.Println(normalized)
// 	// }

// 	lines := strings.Split(
// 		strings.ReplaceAll(text, "\r\n", "\n"),
// 		"\n",
// 	)

// 	for _, line := range lines {
// 		line = strings.TrimSpace(line)
// 		if line == "" {
// 			continue
// 		}
// 		fmt.Printf("%s\n", line)
// 	}
// }
