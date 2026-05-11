package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("stmtdate", flag.ContinueOnError)
	fs.SetOutput(stderr)

	fileFlag := fs.String("file", "", "Path to the PDF file")
	rename := fs.Bool("rename", false, "Rename file by appending statement month")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	filePath, err := resolveFilePath(*fileFlag, fs.Args())
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		fs.Usage()
		return 2
	}

	statementMonth, err := statementMonthFromPDF(filePath)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, statementMonth)

	if *rename {
		newPath, err := renameFileWithMonth(filePath, statementMonth)
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "renamed: %s\n", newPath)
	}

	return 0
}

func resolveFilePath(fileFlag string, positional []string) (string, error) {
	if fileFlag != "" {
		if len(positional) > 0 {
			return "", errors.New("provide either -file or a positional PDF path, not both")
		}
		return fileFlag, nil
	}

	if len(positional) == 0 {
		return "", errors.New("missing PDF file path")
	}

	if len(positional) > 1 {
		return "", errors.New("too many positional arguments")
	}

	return positional[0], nil
}

func statementMonthFromPDF(filePath string) (string, error) {
	creationDate, err := creationDateFromPDF(filePath)
	if err != nil {
		return "", err
	}

	return statementMonthFromCreationDate(creationDate)
}

func creationDateFromPDF(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	info, err := api.PDFInfo(f, filepath.Base(filePath), nil, false, nil)
	if err != nil {
		return "", err
	}

	if strings.TrimSpace(info.CreationDate) == "" {
		return "", errors.New("creationDate not found in PDF metadata")
	}

	return info.CreationDate, nil
}

func statementMonthFromCreationDate(creationDate string) (string, error) {
	parsed, err := parsePDFCreationDate(creationDate)
	if err != nil {
		return "", err
	}

	return parsed.AddDate(0, -1, 0).Format("2006-01"), nil
}

func parsePDFCreationDate(value string) (time.Time, error) {
	layouts := []string{
		"D:20060102150405-07'00'",
		"D:20060102150405Z07'00'",
		"D:20060102150405-0700",
		"D:20060102150405Z0700",
		"D:20060102150405",
		"20060102150405-07'00'",
		"20060102150405Z07'00'",
		"20060102150405-0700",
		"20060102150405Z0700",
		"20060102150405",
	}

	for _, layout := range layouts {
		t, err := time.Parse(layout, value)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unsupported creationDate format: %q", value)
}

func renameFileWithMonth(filePath, statementMonth string) (string, error) {
	newPath := renamedPath(filePath, statementMonth)
	if _, err := os.Stat(newPath); err == nil {
		return "", fmt.Errorf("target file already exists: %s", newPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	if err := os.Rename(filePath, newPath); err != nil {
		return "", err
	}

	return newPath, nil
}

func renamedPath(filePath, statementMonth string) string {
	ext := filepath.Ext(filePath)
	base := strings.TrimSuffix(filePath, ext)
	return base + "-" + statementMonth + ext
}
