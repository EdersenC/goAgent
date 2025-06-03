package tools

import (
	"encoding/json"
	"fmt"
	"github.com/EdersenC/goAgent"
	"github.com/EdersenC/goAgent/api/search"
	"github.com/PuerkitoBio/goquery"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var SearchTool = &goAgent.Tool{}
var ResponseTool = &goAgent.Tool{}

// todo refactor so that we add more fields to engine interface
func init() {
	goAgent.InitTool(SearchTool, "search.json", initSearch)
	goAgent.InitTool(ResponseTool, "respond.json", PrintResponse)
}

func PrintResponse(response map[string]interface{}, chat *goAgent.Chat) (map[string]interface{}, error) {
	arguments, ok := response["arguments"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid arguments format")
	}
	message, ok := arguments["message"].(string)
	if !ok {
		return nil, fmt.Errorf("message not found in arguments")
	}
	fmt.Println("Message:", message)
	return response, nil
}

type DuckDuckGo struct {
}

func (d DuckDuckGo) Trace() *search.Trace {

	panic("implement me")
}

func (d DuckDuckGo) Search(query string, page int) ([]*search.Result, error) {
	offset := (page - 1) * 10
	data := url.Values{
		"q": {query},
		"s": {fmt.Sprintf("%d", offset)},
	}

	req, _ := http.NewRequest("POST", "https://html.duckduckgo.com/html/", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	client := &http.Client{}
	time.Sleep(1 * time.Second)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {

		}
	}(resp.Body)

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var results []*search.Result
	doc.Find(".result").Each(func(i int, s *goquery.Selection) {
		title := s.Find(".result__title a").Text()
		href, _ := s.Find(".result__title a").Attr("href")
		snippet := s.Find(".result__snippet").Text()

		results = append(results, &search.Result{
			Title:   title,
			URL:     href,
			Snippet: snippet,
		})
	})

	return results, nil
}

var Relevancy = 50.0

func initSearch(request map[string]interface{}, chat *goAgent.Chat) (map[string]interface{}, error) {
	arguments, err := extractArguments(request)
	if err != nil {
		return nil, err
	}

	prompt, ok := request["prompt"].(string)
	if !ok || prompt == "" {
		return nil, fmt.Errorf("prompt is required and must be a string")
	}

	reason, ok := arguments["reason"].(string)
	if !ok {
		reason = ""
	}

	queries, err := normalizeQueries(arguments["queries"])
	if err != nil {
		return nil, err
	}
	fmt.Println("Total Queries:", len(queries))

	pageNumber, err := parsePageNumber(arguments["page"])
	if err != nil {
		return nil, err
	}

	engine := DuckDuckGo{}

	traceChat := goAgent.NewChat(chat.Agent, goAgent.NewToolRegistry())
	trace := executeQueries(engine, traceChat, queries, prompt, reason, pageNumber)
	instruction := "Search results Completed."

	summarySection := fmt.Sprintf("**Search Summary**:\n%s", trace.Summarize(chat))

	fullMessage := instruction + "\n\n" + summarySection

	result, err := chat.SendMessage("user", fullMessage, false)
	if err != nil {
		return nil, fmt.Errorf("failed to send assistant message: %w", err)
	}
	result.PrintThoughts()
	result.PrintContent()
	return map[string]interface{}{"summary": trace}, nil
}

func extractArguments(args map[string]interface{}) (map[string]interface{}, error) {
	arguments, ok := args["arguments"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid query parameter")
	}
	return arguments, nil
}

func parsePageNumber(raw interface{}) (int, error) {
	pageStr, ok := raw.(string)
	if !ok || pageStr == "" {
		return 1, nil // default to page 1
	}
	pageNumber, err := strconv.Atoi(pageStr)
	if err != nil {
		return 0, fmt.Errorf("invalid page parameter")
	}
	return pageNumber, nil
}

func executeQueries(engine search.Engine, chat *goAgent.Chat, queries []string, prompt, reason string, pageNumber int) *search.Trace {
	if chat == nil {
		chat = goAgent.NewChat(goAgent.PlannerAgent, goAgent.NewToolRegistry())
	}
	tracer := search.NewTrace(prompt, reason)
	tracer.Chat = chat
	for _, query := range queries {
		err := search.RunQuery(engine, query, tracer, pageNumber, Relevancy)
		if err != nil {
			continue
		}

	}
	return tracer
}

func normalizeQueries(raw interface{}) ([]string, error) {
	switch v := raw.(type) {

	case []interface{}:
		queries := make([]string, len(v))
		for i, val := range v {
			queries[i] = fmt.Sprintf("%v", val)
		}
		return queries, nil

	case []string:
		return v, nil

	case string:
		trimmed := strings.TrimSpace(v)

		// Fix common format error: single-quoted list string
		if strings.HasPrefix(trimmed, "['") && strings.HasSuffix(trimmed, "']") {
			trimmed = strings.ReplaceAll(trimmed, `'`, `"`) // convert to valid JSON
		}

		// First pass: parse into []string
		var parsed []string
		if err := json.Unmarshal([]byte(trimmed), &parsed); err == nil {
			return parsed, nil
		}

		// Second pass: see if it's a stringifies list (i.e. a JSON string containing JSON)
		var embedded string
		if err := json.Unmarshal([]byte(trimmed), &embedded); err == nil {
			var nestedParsed []string
			if err = json.Unmarshal([]byte(embedded), &nestedParsed); err == nil {
				return nestedParsed, nil
			}
		}

		return nil, fmt.Errorf("invalid query format or unsupported string structure")

	default:
		return nil, fmt.Errorf("unexpected queries type: %T", raw)
	}
}
