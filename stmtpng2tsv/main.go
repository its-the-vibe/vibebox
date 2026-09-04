package main

import (
	"context"
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	copilot "github.com/github/copilot-sdk/go"
	"github.com/joho/godotenv"
	"google.golang.org/genai"
)

//go:embed SKILL.md
var skillSpec string

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

type transactionExtractor interface {
	ExtractTransactions(ctx context.Context, imagePath, bank string) (string, error)
	Close() error
}

type transaction struct {
	Date        string
	Description string
	MoneyIn     string
	MoneyOut    string
	Balance     string
}

type agentResponse struct {
	StatementYear int              `json:"statement_year"`
	Transactions  []transactionRaw `json:"transactions"`
}

type transactionRaw struct {
	Date        any `json:"date"`
	Description any `json:"description"`
	MoneyIn     any `json:"money_in"`
	MoneyOut    any `json:"money_out"`
	Balance     any `json:"balance"`
}

func run(args []string, stdout, stderr io.Writer) int {
	_ = godotenv.Load()
	fs := flag.NewFlagSet("stmtpng2tsv", flag.ContinueOnError)
	fs.SetOutput(stderr)

	inputFlag := fs.String("input", "", "Path to bank statement PNG file")
	outputFlag := fs.String("output", "", "Path to output TSV file")
	bankFlag := fs.String("bank", "santander", "Bank statement format (santander or sumup)")
	backendFlag := fs.String("backend", defaultBackend(), "Extraction backend (copilot or gemini)")
	modelFlag := fs.String("model", "", "Model to use (overrides backend default)")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	inputPaths, err := resolveInputPaths(*inputFlag, fs.Args())
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		fs.Usage()
		return 2
	}

	bank, err := normalizeBankFormat(*bankFlag)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		fs.Usage()
		return 2
	}

	for _, inputPath := range inputPaths {
		if strings.ToLower(filepath.Ext(inputPath)) != ".png" {
			fmt.Fprintf(stderr, "error: input must be .png files\n")
			return 2
		}
	}
	sortPathsLogical(inputPaths)

	ctx := context.Background()

	backend := strings.ToLower(*backendFlag)
	model := *modelFlag
	if model == "" {
		model = defaultModel(backend)
	}

	extractor, err := newExtractor(backend, model)
	if err != nil {
		fmt.Fprintf(stderr, "error: could not initialize %s extractor: %v\n", backend, err)
		return 1
	}
	defer extractor.Close()

	allTxns := make([]transaction, 0)
	for _, inputPath := range inputPaths {
		agentText, extractErr := extractor.ExtractTransactions(ctx, inputPath, bank)
		if extractErr != nil {
			fmt.Fprintf(stderr, "error: transaction extraction failed for %s: %v\n", filepath.Base(inputPath), extractErr)
			return 1
		}

		parsed, parseErr := parseAgentResponse(agentText)
		if parseErr != nil {
			fmt.Fprintf(stderr, "error: unable to parse agent response for %s: %v\n", filepath.Base(inputPath), parseErr)
			return 1
		}

		yearHint := parsed.StatementYear
		if yearHint == 0 {
			yearHint = inferYearFromText(agentText)
		}

		txns, normalizeErr := normalizeTransactions(parsed.Transactions, yearHint)
		if normalizeErr != nil {
			fmt.Fprintf(stderr, "error: unable to normalize transactions for %s: %v\n", filepath.Base(inputPath), normalizeErr)
			return 1
		}
		allTxns = append(allTxns, txns...)
	}
	sortTransactionsByDate(allTxns)

	outputPath := *outputFlag
	if outputPath == "" {
		outputPath = inferOutputPath(inputPaths[0], allTxns)
	}

	if err := writeTSV(outputPath, allTxns); err != nil {
		fmt.Fprintf(stderr, "error: unable to write output file: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "wrote %d transactions to %s\n", len(allTxns), outputPath)
	return 0
}

func defaultBackend() string {
	if v := strings.TrimSpace(os.Getenv("STMTPNG2TSV_BACKEND")); v != "" {
		return strings.ToLower(v)
	}
	return "gemini"
}

func defaultModel(backend string) string {
	if v := strings.TrimSpace(os.Getenv("STMTPNG2TSV_MODEL")); v != "" {
		return v
	}
	if backend == "gemini" {
		return "gemini-flash-latest"
	}
	return "gpt-4.1"
}

func newExtractor(backend, model string) (transactionExtractor, error) {
	switch backend {
	case "copilot":
		return newCopilotExtractor(model)
	case "gemini":
		return newGeminiExtractor(model)
	default:
		return nil, fmt.Errorf("unsupported backend: %s", backend)
	}
}

func normalizeBankFormat(s string) (string, error) {
	bank := strings.ToLower(strings.TrimSpace(s))
	if bank == "" {
		bank = "santander"
	}
	switch bank {
	case "santander", "sumup":
		return bank, nil
	default:
		return "", fmt.Errorf("unsupported bank format: %s", s)
	}
}

func resolveInputPaths(inputFlag string, positional []string) ([]string, error) {
	if inputFlag != "" {
		if len(positional) > 0 {
			return nil, errors.New("provide either -input or positional PNG paths, not both")
		}
		return []string{inputFlag}, nil
	}

	if len(positional) == 0 {
		return nil, errors.New("missing PNG file path")
	}
	return append([]string(nil), positional...), nil
}

func sortPathsLogical(paths []string) {
	sort.SliceStable(paths, func(i, j int) bool {
		return logicalStringLess(paths[i], paths[j])
	})
}

func logicalStringLess(a, b string) bool {
	if a == b {
		return false
	}
	ra := []rune(a)
	rb := []rune(b)
	ia, ib := 0, 0
	for ia < len(ra) && ib < len(rb) {
		da := ia < len(ra) && ra[ia] >= '0' && ra[ia] <= '9'
		db := ib < len(rb) && rb[ib] >= '0' && rb[ib] <= '9'
		if da && db {
			sa, sb := ia, ib
			for ia < len(ra) && ra[ia] >= '0' && ra[ia] <= '9' {
				ia++
			}
			for ib < len(rb) && rb[ib] >= '0' && rb[ib] <= '9' {
				ib++
			}
			na := strings.TrimLeft(string(ra[sa:ia]), "0")
			nb := strings.TrimLeft(string(rb[sb:ib]), "0")
			if na == "" {
				na = "0"
			}
			if nb == "" {
				nb = "0"
			}
			if len(na) != len(nb) {
				return len(na) < len(nb)
			}
			if na != nb {
				return na < nb
			}
			continue
		}
		ca := strings.ToLower(string(ra[ia]))
		cb := strings.ToLower(string(rb[ib]))
		if ca != cb {
			return ca < cb
		}
		ia++
		ib++
	}
	return len(ra) < len(rb)
}

func sortTransactionsByDate(txns []transaction) {
	sort.SliceStable(txns, func(i, j int) bool {
		return txns[i].Date < txns[j].Date
	})
}

type copilotExtractor struct {
	client  *copilot.Client
	session *copilot.Session
}

func newCopilotExtractor(model string) (*copilotExtractor, error) {
	ctx := context.Background()
	client := copilot.NewClient(&copilot.ClientOptions{LogLevel: "error"})
	if err := client.Start(ctx); err != nil {
		return nil, err
	}

	session, err := client.CreateSession(ctx, &copilot.SessionConfig{
		Model:               model,
		OnPermissionRequest: copilot.PermissionHandler.ApproveAll,
	})
	if err != nil {
		_ = client.Stop()
		return nil, err
	}

	return &copilotExtractor{client: client, session: session}, nil
}

func (e *copilotExtractor) Close() error {
	if e.session != nil {
		_ = e.session.Disconnect()
	}
	if e.client != nil {
		return e.client.Stop()
	}
	return nil
}

func (e *copilotExtractor) ExtractTransactions(ctx context.Context, imagePath, bank string) (string, error) {
	var mu sync.Mutex
	assistantText := ""
	done := make(chan struct{})
	var once sync.Once

	unsubscribe := e.session.On(func(event copilot.SessionEvent) {
		switch d := event.Data.(type) {
		case *copilot.AssistantMessageData:
			mu.Lock()
			assistantText = d.Content
			mu.Unlock()
		case *copilot.SessionIdleData:
			once.Do(func() { close(done) })
		}
	})
	defer unsubscribe()

	_, err := e.session.Send(ctx, copilot.MessageOptions{
		Prompt: buildExtractionPrompt(bank),
		Attachments: []copilot.Attachment{
			&copilot.AttachmentFile{
				Path:        imagePath,
				DisplayName: filepath.Base(imagePath),
			},
		},
	})
	if err != nil {
		return "", err
	}

	select {
	case <-done:
	case <-ctx.Done():
		return "", ctx.Err()
	}

	mu.Lock()
	defer mu.Unlock()
	if strings.TrimSpace(assistantText) == "" {
		return "", errors.New("empty response from Copilot agent")
	}
	return assistantText, nil
}

type geminiExtractor struct {
	client    *genai.Client
	modelName string
}

func newGeminiExtractor(modelName string) (*geminiExtractor, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, errors.New("GEMINI_API_KEY environment variable not set")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return nil, err
	}

	return &geminiExtractor{client: client, modelName: modelName}, nil
}

func (e *geminiExtractor) Close() error {
	return nil
}

func (e *geminiExtractor) ExtractTransactions(ctx context.Context, imagePath, bank string) (string, error) {
	imgData, err := os.ReadFile(imagePath)
	if err != nil {
		return "", err
	}

	prompt := buildExtractionPrompt(bank)

	contents := []*genai.Content{
		{
			Parts: []*genai.Part{
				{Text: prompt},
				{InlineData: &genai.Blob{Data: imgData, MIMEType: "image/png"}},
			},
		},
	}

	resp, err := e.client.Models.GenerateContent(ctx, e.modelName, contents, &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
	})
	if err != nil {
		return "", err
	}

	res := resp.Text()
	if strings.TrimSpace(res) == "" {
		return "", errors.New("empty response from Gemini")
	}
	return res, nil
}

func buildExtractionPrompt(bank string) string {
	rules := strings.TrimSpace(`Rules:
- Extract only rows from "Your transactions" or "My transactions".
- Keep one transaction per item.
- Keep money fields as plain decimal strings, no currency symbols.
- If amount is missing for Money In or Money Out, use an empty string.
- Prefer statement year from context when the row date omits year.`)
	if bank == "sumup" {
		rules = strings.TrimSpace(`Rules:
- Find the transaction header row with columns: Date, Reference, Type, Amount, Description.
- Extract all transaction rows below that header.
- Map Date to date.
- Map Amount: positive values to money_in; negative values to money_out (absolute value).
- Build description from Reference, Type, and Description joined with " - " and omit empty fields.
- Set balance to an empty string when it is not present in the statement.
- Keep money fields as plain decimal strings, no currency symbols.
- Prefer statement year from context when the row date omits year.`)
	}

	return strings.TrimSpace(`Use this extraction spec:

` + skillSpec + `

Use the attached PNG directly for text extraction (do not require external OCR input).

Return JSON only (no markdown), with this exact schema:
{
  "statement_year": 2026,
  "transactions": [
    {
      "date": "2026-05-13",
      "description": "MAINTAINING THE ACCOUNT - MONTHLY FEE",
      "money_in": "",
      "money_out": "3.00",
      "balance": "737.26"
    }
  ]
}

` + rules + `
`)
}

func parseAgentResponse(response string) (agentResponse, error) {
	payload := extractJSONPayload(response)
	if payload == "" {
		return agentResponse{}, errors.New("no JSON object found")
	}

	var parsed agentResponse
	if err := json.Unmarshal([]byte(payload), &parsed); err != nil {
		return agentResponse{}, err
	}

	if parsed.StatementYear == 0 {
		parsed.StatementYear = inferYearFromText(payload)
	}
	return parsed, nil
}

func extractJSONPayload(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
	}

	start := strings.IndexByte(s, '{')
	end := strings.LastIndexByte(s, '}')
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return strings.TrimSpace(s[start : end+1])
}

func normalizeTransactions(raw []transactionRaw, yearHint int) ([]transaction, error) {
	txns := make([]transaction, 0, len(raw))
	for i, r := range raw {
		isoDate, err := normalizeDate(stringify(r.Date), yearHint)
		if err != nil {
			return nil, fmt.Errorf("row %d: invalid date %q: %w", i+1, stringify(r.Date), err)
		}

		txns = append(txns, transaction{
			Date:        isoDate,
			Description: cleanDescription(stringify(r.Description)),
			MoneyIn:     normalizeAmount(stringify(r.MoneyIn)),
			MoneyOut:    normalizeAmount(stringify(r.MoneyOut)),
			Balance:     normalizeAmount(stringify(r.Balance)),
		})
	}
	return txns, nil
}

func stringify(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(t)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case int:
		return strconv.Itoa(t)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func cleanDescription(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}

func normalizeAmount(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || s == "-" {
		return ""
	}
	s = strings.ReplaceAll(s, "£", "")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.TrimPrefix(s, "+")
	return s
}

func normalizeDate(raw string, statementYear int) (string, error) {
	raw = strings.TrimSpace(strings.ReplaceAll(raw, ",", ""))
	raw = strings.Join(strings.Fields(raw), " ")
	if raw == "" {
		return "", errors.New("empty date")
	}

	candidateFormats := []string{
		"2006-01-02",
		"02/01/2006",
		"2/1/2006",
		"02-01-2006",
		"2-1-2006",
		"2 Jan 2006",
		"02 Jan 2006",
		"2 January 2006",
		"02 January 2006",
	}

	for _, format := range candidateFormats {
		if parsed, err := time.Parse(format, raw); err == nil {
			return parsed.Format("2006-01-02"), nil
		}
	}

	if statementYear == 0 {
		statementYear = time.Now().Year()
	}

	noYearFormats := []string{
		"2 Jan",
		"02 Jan",
		"2 January",
		"02 January",
		"2/1",
		"02/01",
		"2-1",
		"02-01",
	}

	for _, format := range noYearFormats {
		if parsed, err := time.Parse(format+" 2006", raw+" "+strconv.Itoa(statementYear)); err == nil {
			return parsed.Format("2006-01-02"), nil
		}
	}

	return "", fmt.Errorf("unsupported date format: %s", raw)
}

var yearRegex = regexp.MustCompile(`\b(20\d{2})\b`)

func inferYearFromText(s string) int {
	matches := yearRegex.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return 0
	}

	counts := map[int]int{}
	for _, m := range matches {
		y, _ := strconv.Atoi(m[1])
		counts[y]++
	}

	type yearCount struct {
		year  int
		count int
	}

	all := make([]yearCount, 0, len(counts))
	for y, c := range counts {
		all = append(all, yearCount{year: y, count: c})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].count == all[j].count {
			return all[i].year > all[j].year
		}
		return all[i].count > all[j].count
	})
	return all[0].year
}

func inferOutputPath(inputPath string, txns []transaction) string {
	base := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	month := time.Now().Format("2006-01")
	if len(txns) > 0 {
		if parsed, err := time.Parse("2006-01-02", txns[0].Date); err == nil {
			month = parsed.Format("2006-01")
		}
	}
	return filepath.Join(filepath.Dir(inputPath), fmt.Sprintf("%s-%s.tsv", base, month))
}

func writeTSV(path string, txns []transaction) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	writer.Comma = '|'
	defer writer.Flush()

	if err := writer.Write([]string{"Date", "Description", "Money In", "Money Out", "Balance"}); err != nil {
		return err
	}

	for _, tx := range txns {
		if err := writer.Write([]string{tx.Date, tx.Description, tx.MoneyIn, tx.MoneyOut, tx.Balance}); err != nil {
			return err
		}
	}

	if err := writer.Error(); err != nil {
		return err
	}
	return nil
}
