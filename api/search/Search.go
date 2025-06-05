package search

import (
	"fmt"
	"github.com/EdersenC/goAgent"
	"github.com/PuerkitoBio/goquery"
	"io"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// ScrapeContentInto fetches the content of the Result's URL,
// extracts text from <p> tags, and embeds it using the EmbeddingAgent.
// It returns an error if the URL is not HTTPS or if scraping fails.
func (r *Result) ScrapeContentInto() error {
	if !strings.HasPrefix(r.URL, "https://") {
		return fmt.Errorf("skipping non-HTTPS URL: %s", r.URL)
	}

	req, _ := http.NewRequest("GET", r.URL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {

		}
	}(resp.Body)

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return err
	}

	text := ""
	doc.Find("p").Each(func(i int, s *goquery.Selection) {
		text += s.Text() + "\n"
	})

	r.Content = strings.TrimSpace(text)
	if len(r.Content) > 0 {
		embedding, err := goAgent.EmbeddingAgent.Embed(r.Content)
		if err != nil {
			return fmt.Errorf("embedding error: %w", err)
		}
		r.EmbeddedContent = embedding
	} else {
		return fmt.Errorf("no content found for URL: %s", r.URL)
	}
	return nil
}

// cosineSimilarity calculates the cosine similarity between two vectors.
// It returns a value between -1 and 1, where 1 means identical vectors,
func cosineSimilarity(a, b []float64) float64 {
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// AverageComboScore computes the average cosine similarity score
// between all pairs of embeddings from two slices: embeddings and match.
func AverageComboScore(embeddings, match []*goAgent.EmbeddedContent) float64 {
	totalScore := 0.0
	totalComparisons := 0

	if len(embeddings) == 0 || len(match) == 0 {
		return 0.0
	}

	for _, embA := range embeddings {
		vecA := embA.Embedding
		if len(vecA) == 0 {
			continue
		}

		for _, embB := range match {
			vecB := embB.Embedding
			if len(vecB) == 0 {
				continue
			}

			totalScore += cosineSimilarity(vecA, vecB)
			totalComparisons++
		}
	}

	if totalComparisons == 0 {
		return 0.0
	}

	return totalScore / float64(totalComparisons)
}

// rankByRelevance ranks search results based on their relevance to a query.
// It uses the EmbeddingAgent to embed the query and compares it against
// the embedded content of each result using cosine similarity.
func rankByRelevance(results []*Result, query string, minimumThreshHold float64) ([]*Result, error) {
	minimumThreshHold = minimumThreshHold / 100.0 // Convert to a 0-1 scale
	embedding, err := goAgent.EmbeddingAgent.Embed(query)
	rankedResults := make([]*Result, 0)
	if err != nil {
		return nil, fmt.Errorf("embedding error: %w", err)
	}

	for i := range results {
		if results[i].EmbeddedContent == nil {
			continue
		}
		results[i].Score = AverageComboScore(embedding, results[i].EmbeddedContent)
		if results[i].Score < minimumThreshHold {
			continue
		}
		fmt.Println("Ranking Results for:", results[i].Title, "Score:", results[i].Score)
		rankedResults = append(rankedResults, results[i])
	}

	sort.Slice(rankedResults, func(i, j int) bool {
		return rankedResults[i].Score > rankedResults[j].Score
	})

	return rankedResults, nil
}

var SummaryPrompt string
var SearchExtractorPrompt string

var SummaryTool = &goAgent.Tool{}
var searchExtraction = &goAgent.Tool{}

func init() {
	goAgent.InitTool(SummaryTool, "summarize.json", nil)
	goAgent.InitTool(searchExtraction, "searchExtraction.json", ReviewExtraction)
	SummaryPrompt = SummaryTool.AsPrompt(-1)
	SearchExtractorPrompt = searchExtraction.AsPrompt(-1)
}

// handlePage processes a single page of search results.
// It checks the cache for existing results, performs a search if not found,
// scrapes the content, ranks the results by relevance, and summarizes them.
func handlePage(engine Engine, tracer *Trace, query string, page int, minimumRelevancy float64) ([]*Result, error) {
	if cachedResults, found := cache[query]; found {
		fmt.Printf("\n\nUsing cached results for query: %s, page: %d\n\n", query, page)
		return cachedResults, nil
	}

	results, err := engine.Search(query, page)
	if err != nil {
		return nil, fmt.Errorf("search error: %w", err)
	}
	fmt.Println("Results for query:", query, "Page:", page, "Results:", len(results))

	if err = scrapeAll(results); err != nil {
		return nil, fmt.Errorf("scraping error: %w", err)
	}

	rankedResults, err := rankByRelevance(results, query, minimumRelevancy)
	if err != nil {
		return nil, fmt.Errorf("ranking error: %w", err)
	}
	if len(rankedResults) == 0 {
		return rankedResults, nil // No results meet the relevancy threshold
	}
	fmt.Println("Ranked results for query:", query, "Page:", page, "Results:", len(rankedResults))
	tracer.AttachBundle(NewBundle(query, NewPageDigest(results, "", rankedResults)))

	agentTools := tracer.Chat.Agent.SwapRegistry(goAgent.NewToolRegistry(searchExtraction)) // Use only the SearchExtraction tool for this Chat
	ToolRegistry := tracer.Chat.ToolRegistry.Swap(goAgent.NewToolRegistry(searchExtraction))
	message := " IF not Relevant say 'No results found' and exit.\n\n"
	message += "**YOU MUST USE USE TOOLS Provided**"
	newExtraction := searchExtraction.Clone()
	newExtraction.AddConstraints(message)
	tracer.SummaryAgents = make([]*goAgent.Agent, 0)
	tracer.SummaryAgents = append(tracer.SummaryAgents,
		goAgent.SummaryAgent,
		goAgent.SummaryAgent.WithPort("11436"),
	)

	var wg sync.WaitGroup
	jobs := make(chan *Result, len(rankedResults))

	for w := 0; w < len(tracer.SummaryAgents); w++ {
		go func(workerID int) {
			chat := goAgent.NewChat(tracer.SummaryAgents[workerID], goAgent.NewToolRegistry(newExtraction))
			for result := range jobs { // pull jobs from the channel
				fmt.Printf("Worker %d summarizing: %s URL: %s\n", workerID, result.Title, result.URL)
				result.Summarize(
					chat,
					message,
					chat.Agent.ContextPortion(75),
				)
				wg.Done()
			}
		}(w)
	}

	for i := range rankedResults {
		wg.Add(1)
		jobs <- rankedResults[i]
	}

	close(jobs)
	wg.Wait()

	tracer.Chat.Agent.SwapRegistry(agentTools) // Restore original tools after summarization
	tracer.Chat.ToolRegistry.Swap(ToolRegistry)
	return rankedResults, nil
}

// RunQuery executes a search query using the specified engine and tracer.
// It handles pagination, caches results, and summarizes the findings.
func RunQuery(engine Engine, query string, tracer *Trace, pages int, minimumRelevancy float64) error {
	allRankedResults := make([]*Result, 0)
	start := time.Now()

	tracer.Chat.Agent = goAgent.SummaryAgent

	for page := 1; page <= pages; page++ {
		pageResults, err := handlePage(engine, tracer, query, page, minimumRelevancy)
		if err != nil {
			fmt.Println("Error handling page:", err)
			continue
		}
		allRankedResults = append(allRankedResults, pageResults...)
		cache[query] = pageResults
	}
	if len(allRankedResults) == 0 {
		return fmt.Errorf("no results found for query: %s", query)
	}
	tracer.Duration = time.Since(start).Milliseconds()
	return nil
}

func (t *Trace) Summarize(chat *goAgent.Chat) string {
	return ""
}

// scrapeAll iterates over a slice of Result pointers and scrapes their content.
// It calls ScrapeContentInto for each Result and logs any errors encountered.
func scrapeAll(results []*Result) error {
	for _, result := range results {
		if err := result.ScrapeContentInto(); err != nil {
			fmt.Println("Error scraping content:", err)
		}
	}
	return nil
}

// printResult formats and prints the details of a Result object.
func printResult(res *Result) {
	fmt.Println("-------------------------------------------")
	fmt.Printf("Title: %s\nURL: %s\nSnippet: %s\nContent: %s\nScore: %.4f\n\n",
		res.Title, res.URL, res.Snippet, res.getSummary(), res.Score)
	fmt.Println("-------------------------------------------")
}
