package search

import (
	"fmt"
	"github.com/EdersenC/goAgent"
	"strings"
	"time"
)

// Todo add more fields To trace infrostructure
type Trace struct {
	UserPrompt     string
	Reason         string
	Bundle         []*Bundle
	cache          map[string][]*Result
	Duration       int64
	Chat           *goAgent.Chat
	SummaryAgents  []*goAgent.Agent
	EmbeddingAgent *goAgent.Agent
}

// NewTrace creates a new Trace instance with the provided user prompt and reason.
func (t *Trace) FormatDuration() string {
	if t.Duration <= 0 {
		return "0ms"
	}
	return formatDuration(t.Duration)
}

// formatDuration formats the given duration in milliseconds into a human-readable string.
func formatDuration(duration int64) string {
	return (time.Duration(duration) * time.Millisecond).String()
}

type Bundle struct {
	Query      string
	PageDigest *[]PageDigest
}

type PageDigest struct {
	Results     []*Result
	Summary     *Summary
	SummarySpan int
	ranked      []*Result
}

type Summary struct {
	Content  string
	Duration int64
}

// newSummary creates a new Summary instance with the provided content and duration.
func newSummary(summary string, duration int64) *Summary {
	return &Summary{
		Content:  summary,
		Duration: duration,
	}
}

// NewSummary creates a new Summary for the Result with the provided content and duration.
func (r *Result) NewSummary(summary string, duration int64) {
	r.Summary = newSummary(summary, duration)
}

type Result struct {
	Title           string
	URL             string
	Snippet         string
	Content         string
	EmbeddedContent []*goAgent.EmbeddedContent
	Summary         *Summary
	Score           float64
}

// FormatDuration formats the duration of the Result's summary into a human-readable string.
func (r *Result) FormatDuration() string {
	if r.Summary.Duration <= 0 {
		return "0ms"
	}
	return formatDuration(r.Summary.Duration)
}

// getSummary retrieves the summary content from the Result.
func (r *Result) getSummary() string {
	if r.Summary == nil {
		return ""
	}
	return r.Summary.Content
}

// Summarize generates a summary of the Result's content using the provided chat agent and instructions.
// It chunks the content if it exceeds the maximum context size and processes each chunk to create a summary.
func (r *Result) Summarize(chat *goAgent.Chat, instructions string, maxContext int) string {
	if r.getSummary() != "" {
		return r.getSummary()
	}
	pageInfo := fmt.Sprintf(
		"Title: %s\nURL: %s\nContent: %s\n\n**End of%s**\n\n",
		r.Title, r.URL, r.Content, r.Title,
	)
	task := fmt.Sprintf("\n\n%s\n\n%s", instructions, pageInfo)
	chunks := goAgent.ChunkByTokens(task, maxContext)
	if len(chunks) == 0 {
		return "No content to summarize"
	}
	startTime := time.Now()
	processedChunks := ProcessChunks(chunks, chat, instructions, maxContext)
	var summary strings.Builder
	summary.WriteString(strings.Join(processedChunks, "\n\n"))

	for goAgent.Tokenize(chat.Agent.SystemPrompt+summary.String()) > maxContext {
		fmt.Println("Summary too long, chunking again")
		summary.Reset()
		summary.WriteString(strings.Join(ProcessChunks(processedChunks, chat, instructions, maxContext), "\n\n"))
	}
	r.NewSummary(summary.String(), time.Since(startTime).Milliseconds())
	fmt.Println("\n\nSummary duration:", r.FormatDuration())

	return r.getSummary()
}

// Attach a new Bundle (appends to the bundle slice)
func (t *Trace) AttachBundle(b *Bundle) *Trace {
	t.Bundle = append(t.Bundle, b)
	return t
}

// GetBundle returns a bundle by 1-based index (page number)
func (t *Trace) GetBundle(page int) *Bundle {
	if page <= 0 || page > len(t.Bundle) {
		return nil
	}
	return t.Bundle[page-1]
}

// GetPageDigest from a specific page and index within that page
func (t *Trace) GetPageDigest(page, digestIndex int) *PageDigest {
	bundle := t.GetBundle(page)
	if bundle == nil || bundle.PageDigest == nil {
		return nil
	}
	pages := *bundle.PageDigest
	if digestIndex <= 0 || digestIndex > len(pages) {
		return nil
	}
	return &pages[digestIndex-1]
}

// GetPageResults returns the results from a specific digest on a page
func (t *Trace) GetPageResults(page, digestIndex int) []*Result {
	digest := t.GetPageDigest(page, digestIndex)
	if digest == nil {
		return nil
	}
	return digest.Results
}

// GetPageSummary returns the summary for a specific digest on a page
func (t *Trace) GetPageSummary(page, digestIndex int) string {
	digest := t.GetPageDigest(page, digestIndex)
	if digest == nil {
		return ""
	}
	return digest.Summary.Content
}

// Attach to cache (optional helper)
func (t *Trace) AttachResultToCache(source string, r *Result) *Trace {
	if t.cache == nil {
		t.cache = make(map[string][]*Result)
	}
	t.cache[source] = append(t.cache[source], r)
	return t
}

// Create a new Result
func NewResult(title, url, snippet, content, summary string, score float64) *Result {
	return &Result{
		Title:   title,
		URL:     url,
		Snippet: snippet,
		Content: content,
		Summary: &Summary{
			Content: summary,
		},
		Score: score,
	}
}

// NewTrace creates a new Trace instance with the provided prompt and reason.
func NewTrace(prompt, reason string) *Trace {
	return &Trace{
		UserPrompt: prompt,
		Reason:     reason,
		Bundle:     make([]*Bundle, 0),
		cache:      make(map[string][]*Result),
	}
}

// NewPageDigest creates a new PageDigest with the provided results, summary, and ranked results.
func NewPageDigest(results []*Result, summary string, ranked []*Result) *PageDigest {
	return &PageDigest{
		Results: results,
		Summary: &Summary{
			Content: summary,
		},
		ranked: ranked,
	}
}

// NewBundle creates a new Bundle with the provided query and page digests.
func NewBundle(query string, digests ...*PageDigest) *Bundle {
	p := make([]PageDigest, 0, len(digests))
	for _, d := range digests {
		if d != nil {
			p = append(p, *d)
		}
	}
	return &Bundle{
		Query:      query,
		PageDigest: &p,
	}
}

// todo make more fields for history and mabe another holder stuct
type Engine interface {
	Search(query string, page int) ([]*Result, error)
}

var cache = make(map[string][]*Result)
