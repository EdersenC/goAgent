package search

import (
	"fmt"
	"github.com/EdersenC/goAgent"
	"slices"
	"strings"
	"time"
)

// Todo add more fields To trace infrostructure
type Trace struct {
	UserPrompt     string
	Reason         string
	Bundles        []*Bundle
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
	PageDigest []*PageDigest
}

type PageDigest struct {
	Results          []*Result
	Summary          *Summary
	SummarySpan      int
	AverageRelevancy float64
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

// ExtractInformation generates a summary of the Result's content using the provided chat agent and instructions.
// It chunks the content if it exceeds the maximum context size and processes each chunk to create a summary.
func (r *Result) ExtractInformation(chat *goAgent.Chat, instructions string, maxContext int) string {
	if r.getSummary() != "" {
		return r.getSummary()
	}
	pageInfo := r.FormatInfo()
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

func (r *Result) FormatInfo() string {
	pageInfo := fmt.Sprintf(
		"Title: %s\nURL: %s\nContent: %s\n\n**End of%s**\n\n",
		r.Title, r.URL, r.Content, r.Title,
	)
	return pageInfo
}

func getTotalContext[T any](items []T, getContextSize func(T) int) (totalContext int) {
	for _, item := range items {
		totalContext += getContextSize(item)
	}
	return
}

func (p *PageDigest) getTotalContextSize() int {
	return getTotalContext(p.Results, func(result *Result) int {
		return goAgent.Tokenize(result.Content)
	})
}

func (b *Bundle) getTotalContextSize() int {
	return getTotalContext(b.PageDigest, func(digest *PageDigest) int {
		return digest.getTotalContextSize()
	})
}

func (t *Trace) getTotalContextSize() int {
	return getTotalContext(t.Bundles, func(bundle *Bundle) int {
		return bundle.getTotalContextSize()
	})
}

// SortArrayBy sorts any slice of items by their context size in descending order.
// The getContextSize function should return the context size for each item.
func SortArrayBy[T any](items []T, function func(T) int) {
	slices.SortFunc(items, func(a, b T) int {
		aSize := function(a)
		bSize := function(b)
		if aSize > bSize {
			return -1
		} else if aSize < bSize {
			return 1
		}
		return 0
	})
}

func (p *PageDigest) SortByContextSize() {
	SortArrayBy(p.Results, func(d *Result) int {
		return goAgent.Tokenize(d.Content)
	})
}

func (b *Bundle) SortByContextSize() {
	SortArrayBy(b.PageDigest, func(d *PageDigest) int {
		return d.getTotalContextSize()
	})
}

func (b *Bundle) FormatContent() string {
	return ""
}

func (t *Trace) SortByContextSize() {
	SortArrayBy(t.Bundles, func(b *Bundle) int {
		return b.getTotalContextSize()
	})
}

func (t *Trace) MeetContextSize(agents []*goAgent.Agent, contextSize int) {
	for _, bundle := range t.Bundles {
		bundle.LowerContextSize(agents, contextSize)
	}
}

func (t *Trace) FormatResults() string {
	println(t.String())
	return t.String()
}

func (t *Trace) Summarize(chat *goAgent.Chat) (string, error) {
	if len(t.Bundles) < 0 {
		return "", fmt.Errorf("no Bundles to summarize")
	}
	var builder strings.Builder
	maxContext := chat.Agent.ContextPortion(75)
	t.MeetContextSize(t.SummaryAgents, maxContext)
	builder.WriteString(t.FormatResults())
	return builder.String(), nil
}

// this func will opperate on a bunch of pages
func (b *Bundle) LowerContextSize(agents []*goAgent.Agent, contextSize int) {
	pageContextSize := contextSize / len(b.PageDigest)
	totalContextSize := b.getTotalContextSize()
	//should punish lower relevancy and higher context size pages
	b.RankPages()
	for i := len(b.PageDigest) - 1; i > 0; i-- {
		page := b.PageDigest[i]
		if totalContextSize <= contextSize {
			break
		}
		if page.getTotalContextSize() <= pageContextSize {
			continue
		}
		page.LowerContextSize(agents, pageContextSize)
		totalContextSize -= page.getTotalContextSize()
	}
}

// here is where alot of the magic happens fr
func (p *PageDigest) LowerContextSize(agents []*goAgent.Agent, contextSize int) {
	resultContextSize := contextSize / len(p.Results)
	totalContextSize := p.getTotalContextSize()
	for i := len(p.Results) - 1; i > 0; i-- {
		result := p.Results[i]
		if totalContextSize <= contextSize {
			break
		}
		if goAgent.Tokenize(result.Content) <= resultContextSize {
			continue
		}
		result.Summarize(resultContextSize)
	}
}

func (r *Result) Summarize(maxContext int) string {
	chat := goAgent.NewChat(goAgent.SummaryAgent, goAgent.NewToolRegistry(searchExtraction.Clone()))
	goAgent.SummaryAgent.Tools = goAgent.NewToolRegistry(searchExtraction.Clone())
	instructions := "Summarize this Article and extract relevant info"
	return r.ExtractInformation(chat, instructions, maxContext)
}

var divider = "---"

func (t *Trace) String() string {
	var trace strings.Builder
	trace.WriteString(
		fmt.Sprintf(
			"# **UserPrompt: %s**\n ## Reason: %s\n",
			t.UserPrompt,
			t.Reason,
		),
	)
	for _, bundle := range t.Bundles {
		trace.WriteString(bundle.String())
	}
	return trace.String()
}

func (b *Bundle) String() string {
	var bundle strings.Builder
	bundleInfo := fmt.Sprintf("\n%s\n# **Search Results For: %s**\n", divider, b.Query)
	bundle.WriteString(bundleInfo)
	for i, digest := range b.PageDigest {
		bundle.WriteString(digest.String(i))
	}
	return bundle.String()
}

func (p *PageDigest) String(pageNumber int) string {
	var page strings.Builder
	pageInfo := fmt.Sprintf("\n%s\n ## Page:  %d\n", divider, pageNumber)
	page.WriteString(pageInfo)
	for _, result := range p.Results {
		page.WriteString(result.String())
	}
	page.WriteString(divider)
	return page.String()
}

func (r *Result) String() string {
	content := r.Content
	if r.Summary != nil && r.Summary.Content != "" {
		content = r.Summary.Content
	}
	return fmt.Sprintf(
		"\n%s\n### Title:\n%s\n#### Content:\n%s\n%s\n",
		divider, r.Title, content, divider,
	)
}

// Attach a new Bundle (appends to the bundle slice)
func (t *Trace) AttachBundle(b *Bundle) *Trace {
	t.Bundles = append(t.Bundles, b)
	return t
}

// GetBundle returns a bundle by 1-based index (page number)
func (t *Trace) GetBundle(page int) *Bundle {
	if page <= 0 || page > len(t.Bundles) {
		return nil
	}
	return t.Bundles[page-1]
}

// GetPageDigest from a specific page and index within that page
func (t *Trace) GetPageDigest(page, digestIndex int) *PageDigest {
	bundle := t.GetBundle(page)
	if bundle == nil || bundle.PageDigest == nil {
		return nil
	}
	pages := bundle.PageDigest
	if digestIndex <= 0 || digestIndex > len(pages) {
		return nil
	}
	return pages[digestIndex-1]
}

// GetPageResults returns the results from a specific digest on a page
// todo make 0 based
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
		Bundles:    make([]*Bundle, 0),
		cache:      make(map[string][]*Result),
	}
}

// NewPageDigest creates a new PageDigest with the provided results, summary, and ranked results.
func NewPageDigest(results []*Result) (digest *PageDigest) {
	digest = &PageDigest{
		Results: results,
		Summary: &Summary{
			Content: "",
		},
	}

	digest.ComputeAverageRelevancy()
	return
}
func (p *PageDigest) getRankedResults(query string, minimumThreshHold float64) ([]*Result, error) {
	return rankByRelevance(p.Results, query, minimumThreshHold)
}
func (p *PageDigest) ComputeAverageRelevancy() {
	for _, result := range p.Results {
		p.AverageRelevancy += result.Score
	}
	p.AverageRelevancy = p.AverageRelevancy / float64(len(p.Results))
}

func (b *Bundle) getAverageRelevancy() (average float64) {
	for _, digest := range b.PageDigest {
		average += digest.AverageRelevancy
	}
	average = average / float64(len(b.PageDigest))
	return
}

func (b *Bundle) RankPages() {
	slices.SortFunc(b.PageDigest, func(a, b *PageDigest) int {
		aSize := a.AverageRelevancy
		bSize := b.AverageRelevancy
		if aSize > bSize {
			return -1
		} else if aSize < bSize {
			return 1
		}
		return 0
	})
}

// NewBundle creates a new Bundle with the provided query and page digests.
func NewBundle(query string, digests ...*PageDigest) *Bundle {
	p := make([]*PageDigest, 0, len(digests))
	for i := range digests {
		if digests[i] != nil {
			p = append(p, digests[i])
		}
	}
	return &Bundle{
		Query:      query,
		PageDigest: p,
	}
}

func (b *Bundle) GetRankedResults(minimumThreshHold float64) ([]*Result, error) {
	allRankedResults := make([]*Result, 0)
	pages := b.PageDigest
	for i := range pages {
		rankedResult, err := pages[i].getRankedResults(b.Query, minimumThreshHold)
		if err != nil {
			return nil, err
		}
		allRankedResults = append(allRankedResults, rankedResult...)
	}

	return allRankedResults, nil
}

// todo make more fields for history and mabe another holder stuct
type Engine interface {
	Search(query string, page int) ([]*Result, error)
}

var cache = make(map[string]*Bundle)
