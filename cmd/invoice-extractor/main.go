package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"github.com/sieryo/invoice-extractor/internal/invoice/template/seamakeup"
	"github.com/sieryo/invoice-extractor/internal/pdftool"
)

func main() {
	ctx := context.Background()

	pdfPath := filepath.Join("assets", "pdf", "contoh_beda.pdf")

	text, err := pdftool.ExtractText(
		ctx,
		pdfPath,
		pdftool.DefaultOptions(),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== PDF TEXT ===")

	fmt.Println(text)

	template := seamakeup.SeaMakeupTemplate{}

	if template.Match(text) {
		normalized, err := template.Normalize(text)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("=== NORMALIZED ===")
		fmt.Println(normalized)
	}

	// lines := strings.Split(
	// 	strings.ReplaceAll(text, "\r\n", "\n"),
	// 	"\n",
	// )

	// for i, line := range lines {
	// 	line = strings.TrimSpace(line)
	// 	if line == "" {
	// 		continue
	// 	}
	// 	fmt.Printf("[%d] %s\n", i, line)
	// }
}
