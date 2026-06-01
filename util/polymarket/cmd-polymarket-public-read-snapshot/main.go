package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/useryege/athena/util/polymarket"
)

const (
	defaultDocsRoot = "util/polymarket/polymarket-docs/api-reference"
	defaultOutput   = "util/polymarket/request-response/latest"
)

var (
	reHeader    = regexp.MustCompile("(?m)^````yaml\\s+(\\S+)\\s+([A-Za-z]+)\\s+(\\S+)\\s*$")
	reFence     = regexp.MustCompile("(?m)^````\\s*$")
	rePathParam = regexp.MustCompile(`\{([^}]+)\}`)
)

type cliConfig struct {
	docsRoot string
	outRoot  string
	dryRun   bool
	timeout  time.Duration
}

type endpoint struct {
	Key             string
	DocSource       string
	SpecName        string
	Title           string
	Summary         string
	Method          string
	Path            string
	BaseURL         string
	RequiresAuth    bool
	PathParams      []param
	QueryParams     []param
	RequestBody     *requestBody
	InclusionReason string
	ExclusionReason string
}

type param struct {
	Name     string
	In       string
	Required bool
	Type     string
}

type requestBody struct {
	Required bool
	JSON     *schema
}

type schema struct {
	Type       string             `yaml:"type"`
	Format     string             `yaml:"format"`
	Ref        string             `yaml:"$ref"`
	Properties map[string]*schema `yaml:"properties"`
	Items      *schema            `yaml:"items"`
	Required   []string           `yaml:"required"`
	Enum       []any              `yaml:"enum"`
}

type openAPISpec struct {
	Servers    []server              `yaml:"servers"`
	Security   []map[string][]string `yaml:"security"`
	Paths      map[string]pathItem   `yaml:"paths"`
	Components components            `yaml:"components"`
}

type components struct {
	Parameters map[string]parameter `yaml:"parameters"`
	Schemas    map[string]schema    `yaml:"schemas"`
}

type server struct {
	URL string `yaml:"url"`
}

type pathItem struct {
	Parameters []parameter `yaml:"parameters"`
	Get        *operation  `yaml:"get"`
	Post       *operation  `yaml:"post"`
	Put        *operation  `yaml:"put"`
	Delete     *operation  `yaml:"delete"`
	Patch      *operation  `yaml:"patch"`
}

type operation struct {
	Summary     string                `yaml:"summary"`
	Description string                `yaml:"description"`
	Parameters  []parameter           `yaml:"parameters"`
	Security    []map[string][]string `yaml:"security"`
	RequestBody *rawRequestBody       `yaml:"requestBody"`
}

type rawRequestBody struct {
	Required bool                 `yaml:"required"`
	Content  map[string]mediaType `yaml:"content"`
}

type mediaType struct {
	Schema *schema `yaml:"schema"`
}

type parameter struct {
	Ref      string  `yaml:"$ref"`
	Name     string  `yaml:"name"`
	In       string  `yaml:"in"`
	Required bool    `yaml:"required"`
	Schema   *schema `yaml:"schema"`
}

type discovery struct {
	MarketID     int64
	MarketSlug   string
	EventID      int64
	EventSlug    string
	TagID        string
	TagSlug      string
	CommentID    string
	UserAddress  string
	SeriesID     int64
	ConditionID  string
	TokenID      string
	TodayUTCDate string
}

type requestLog struct {
	Method    string              `json:"method"`
	URL       string              `json:"url"`
	Headers   map[string]string   `json:"headers"`
	Query     map[string][]string `json:"query,omitempty"`
	Body      any                 `json:"body,omitempty"`
	StartedAt string              `json:"started_at"`
}

type responseLog struct {
	StatusCode int                 `json:"status_code"`
	Headers    map[string][]string `json:"headers"`
	Body       any                 `json:"body,omitempty"`
	RawBody    string              `json:"raw_body,omitempty"`
	LatencyMS  int64               `json:"latency_ms"`
	EndedAt    string              `json:"ended_at"`
}

type metaLog struct {
	EndpointKey string `json:"endpoint_key"`
	DocSource   string `json:"doc_source"`
	Result      string `json:"result"`
	Error       string `json:"error,omitempty"`
}

type indexRecord struct {
	EndpointKey string `json:"endpoint_key"`
	Method      string `json:"method"`
	URL         string `json:"url"`
	StatusCode  int    `json:"status_code,omitempty"`
	Result      string `json:"result"`
	Error       string `json:"error,omitempty"`
	DocSource   string `json:"doc_source"`
}

type summary struct {
	StartedAt       string         `json:"started_at"`
	EndedAt         string         `json:"ended_at"`
	TotalCandidates int            `json:"total_candidates"`
	Included        int            `json:"included"`
	Excluded        map[string]int `json:"excluded"`
	Passed          int            `json:"passed"`
	Failed          int            `json:"failed"`
	Failures        []string       `json:"failures"`
	DurationMS      int64          `json:"duration_ms"`
}

func main() {
	cfg := cliConfig{}
	flag.StringVar(&cfg.docsRoot, "docs-root", defaultDocsRoot, "Path to api-reference markdown docs")
	flag.StringVar(&cfg.outRoot, "out", defaultOutput, "Output snapshot directory")
	flag.BoolVar(&cfg.dryRun, "dry-run", false, "Parse and classify endpoints without sending requests")
	flag.DurationVar(&cfg.timeout, "timeout", 20*time.Second, "Per-request timeout")
	flag.Parse()

	start := time.Now().UTC()

	endpoints, excluded, totalCandidates, err := collectEndpoints(cfg.docsRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "collect endpoints failed: %v\n", err)
		os.Exit(1)
	}

	if err := ensureUniqueKeys(endpoints); err != nil {
		fmt.Fprintf(os.Stderr, "endpoint key conflict: %v\n", err)
		os.Exit(1)
	}

	if cfg.dryRun {
		fmt.Printf("dry-run\n")
		fmt.Printf("total candidates: %d\n", totalCandidates)
		fmt.Printf("included: %d\n", len(endpoints))
		fmt.Printf("excluded:\n")
		for _, k := range sortedMapKeys(excluded) {
			fmt.Printf("  %s: %d\n", k, excluded[k])
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	hc := &http.Client{Timeout: cfg.timeout}
	d, err := discoverSamples(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "discovery failed: %v\n", err)
		os.Exit(1)
	}

	if err := os.RemoveAll(cfg.outRoot); err != nil {
		fmt.Fprintf(os.Stderr, "clear output dir failed: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(cfg.outRoot, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "create output dir failed: %v\n", err)
		os.Exit(1)
	}

	summary := summary{
		StartedAt:       start.Format(time.RFC3339Nano),
		TotalCandidates: totalCandidates,
		Included:        len(endpoints),
		Excluded:        excluded,
	}

	indexFile, err := os.Create(filepath.Join(cfg.outRoot, "index.ndjson"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "create index.ndjson failed: %v\n", err)
		os.Exit(1)
	}
	defer indexFile.Close()

	for _, ep := range endpoints {
		rec := runEndpoint(ctx, hc, cfg.outRoot, ep, d)
		if rec.Result == "pass" {
			summary.Passed++
		} else {
			summary.Failed++
			summary.Failures = append(summary.Failures, rec.EndpointKey+": "+rec.Error)
		}
		b, _ := json.Marshal(rec)
		_, _ = indexFile.Write(append(b, '\n'))
	}

	end := time.Now().UTC()
	summary.EndedAt = end.Format(time.RFC3339Nano)
	summary.DurationMS = end.Sub(start).Milliseconds()

	if err := writeJSON(filepath.Join(cfg.outRoot, "summary.json"), summary); err != nil {
		fmt.Fprintf(os.Stderr, "write summary failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("snapshot finished: included=%d pass=%d fail=%d\n", len(endpoints), summary.Passed, summary.Failed)
	fmt.Printf("output: %s\n", cfg.outRoot)

	if summary.Failed > 0 {
		os.Exit(1)
	}
}

func collectEndpoints(docsRoot string) ([]endpoint, map[string]int, int, error) {
	files := make([]string, 0, 128)
	err := filepath.WalkDir(docsRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, nil, 0, err
	}
	sort.Strings(files)

	excluded := map[string]int{}
	in := make([]endpoint, 0, len(files))
	totalCandidates := 0
	seen := map[string]struct{}{}

	for _, file := range files {
		ep, ok, reason, err := parseEndpointFromFile(file)
		if err != nil {
			excluded["parse_error"]++
			continue
		}
		if !ok {
			if reason == "" {
				reason = "no_openapi_endpoint"
			}
			excluded[reason]++
			continue
		}
		totalCandidates++
		okInclude, reason := isPublicReadEndpoint(ep)
		if !okInclude {
			excluded[reason]++
			continue
		}
		if _, exists := seen[ep.Key]; exists {
			excluded["duplicate_key"]++
			continue
		}
		seen[ep.Key] = struct{}{}
		in = append(in, ep)
	}

	sort.Slice(in, func(i, j int) bool { return in[i].Key < in[j].Key })
	return in, excluded, totalCandidates, nil
}

func parseEndpointFromFile(path string) (endpoint, bool, string, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return endpoint{}, false, "", err
	}
	content := string(payload)

	m := reHeader.FindStringSubmatchIndex(content)
	if m == nil {
		return endpoint{}, false, "no_openapi_header", nil
	}
	specName := strings.TrimSpace(content[m[2]:m[3]])
	method := strings.ToUpper(strings.TrimSpace(content[m[4]:m[5]]))
	pathValue := strings.TrimSpace(content[m[6]:m[7]])

	blockStart := m[1]
	fence := reFence.FindStringIndex(content[blockStart:])
	if fence == nil {
		return endpoint{}, false, "openapi_block_unclosed", nil
	}
	yamlBlock := content[blockStart : blockStart+fence[0]]

	var spec openAPISpec
	if err := yaml.Unmarshal([]byte(yamlBlock), &spec); err != nil {
		return endpoint{}, false, "", fmt.Errorf("yaml unmarshal for %s: %w", path, err)
	}

	op, pathParams, ok := pickOperation(spec.Paths[pathValue], method)
	if !ok || op == nil {
		return endpoint{}, false, "operation_not_found", nil
	}

	params, err := resolveParameters(spec.Components, append(pathParams, op.Parameters...))
	if err != nil {
		return endpoint{}, false, "", fmt.Errorf("resolve parameters %s: %w", path, err)
	}

	title := extractTitle(content)
	summaryText := strings.TrimSpace(op.Summary)
	if summaryText == "" {
		summaryText = title
	}

	baseURL := ""
	if len(spec.Servers) > 0 {
		baseURL = strings.TrimSpace(spec.Servers[0].URL)
	}
	if baseURL == "" {
		baseURL = fallbackBaseURL(specName, path)
	}

	ep := endpoint{
		DocSource:    path,
		SpecName:     specName,
		Title:        title,
		Summary:      summaryText,
		Method:       method,
		Path:         pathValue,
		BaseURL:      baseURL,
		RequiresAuth: requiresAuth(spec.Security, op.Security, params),
		RequestBody:  toRequestBody(op.RequestBody),
	}

	for _, p := range params {
		pp := param{Name: p.Name, In: p.In, Required: p.Required}
		if p.Schema != nil {
			pp.Type = strings.TrimSpace(p.Schema.Type)
		}
		switch p.In {
		case "path":
			ep.PathParams = append(ep.PathParams, pp)
		case "query":
			ep.QueryParams = append(ep.QueryParams, pp)
		}
	}

	ep.Key = endpointKey(ep)
	return ep, true, "", nil
}

func pickOperation(item pathItem, method string) (*operation, []parameter, bool) {
	switch method {
	case http.MethodGet:
		return item.Get, item.Parameters, item.Get != nil
	case http.MethodPost:
		return item.Post, item.Parameters, item.Post != nil
	case http.MethodPut:
		return item.Put, item.Parameters, item.Put != nil
	case http.MethodDelete:
		return item.Delete, item.Parameters, item.Delete != nil
	case http.MethodPatch:
		return item.Patch, item.Parameters, item.Patch != nil
	default:
		return nil, nil, false
	}
}

func resolveParameters(components components, params []parameter) ([]parameter, error) {
	out := make([]parameter, 0, len(params))
	for _, p := range params {
		if strings.TrimSpace(p.Ref) == "" {
			out = append(out, p)
			continue
		}
		name := strings.TrimPrefix(strings.TrimSpace(p.Ref), "#/components/parameters/")
		resolved, ok := components.Parameters[name]
		if !ok {
			return nil, fmt.Errorf("missing parameter ref %q", p.Ref)
		}
		out = append(out, resolved)
	}
	return out, nil
}

func toRequestBody(in *rawRequestBody) *requestBody {
	if in == nil {
		return nil
	}
	out := &requestBody{Required: in.Required}
	if mt, ok := in.Content["application/json"]; ok {
		out.JSON = mt.Schema
	}
	return out
}

func requiresAuth(globalSec, opSec []map[string][]string, params []parameter) bool {
	sec := globalSec
	if opSec != nil {
		sec = opSec
	}
	if len(sec) > 0 {
		for _, req := range sec {
			if len(req) > 0 {
				return true
			}
		}
	}
	for _, p := range params {
		if strings.ToLower(p.In) != "header" {
			continue
		}
		n := strings.ToUpper(strings.TrimSpace(p.Name))
		if strings.HasPrefix(n, "POLY_") || strings.HasPrefix(n, "RELAYER_") || n == "AUTHORIZATION" || strings.Contains(n, "API_KEY") {
			return true
		}
	}
	return false
}

func isPublicReadEndpoint(ep endpoint) (bool, string) {
	p := strings.ToLower(ep.DocSource)
	if strings.Contains(p, "/wss/") {
		return false, "wss_excluded"
	}
	if strings.Contains(p, "/relayer/") || strings.Contains(p, "/relayer-api-keys/") {
		return false, "relayer_excluded"
	}
	if strings.Contains(p, "/trade/") {
		return false, "trade_excluded"
	}

	if ep.RequiresAuth {
		return false, "auth_required"
	}

	method := strings.ToUpper(ep.Method)
	if method == http.MethodGet {
		return true, ""
	}
	if method != http.MethodPost {
		return false, "non_read_method"
	}

	t := strings.ToLower(strings.TrimSpace(ep.Title + " " + ep.Summary))
	if strings.HasPrefix(t, "get ") || strings.HasPrefix(t, "list ") || strings.HasPrefix(t, "search ") || strings.HasPrefix(t, "check ") {
		return true, ""
	}
	if strings.Contains(strings.ToLower(ep.Path), "prices") || strings.Contains(strings.ToLower(ep.Path), "books") || strings.Contains(strings.ToLower(ep.Path), "midpoints") || strings.Contains(strings.ToLower(ep.Path), "spreads") || strings.Contains(strings.ToLower(ep.Path), "quote") {
		return true, ""
	}
	return false, "post_not_read_semantic"
}

func endpointKey(ep endpoint) string {
	host := "unknown-host"
	if u, err := url.Parse(ep.BaseURL); err == nil && u.Host != "" {
		host = u.Host
	}
	k := strings.ToLower(ep.Method) + "_" + host + "_" + strings.TrimPrefix(ep.Path, "/")
	k = strings.ReplaceAll(k, "/", "_")
	k = strings.ReplaceAll(k, "{", "")
	k = strings.ReplaceAll(k, "}", "")
	k = strings.ReplaceAll(k, "-", "_")
	return k
}

func extractTitle(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return ""
}

func fallbackBaseURL(specName, docPath string) string {
	s := strings.ToLower(specName)
	switch {
	case strings.Contains(s, "gamma"):
		return polymarket.DefaultGammaBaseURL
	case strings.Contains(s, "data"):
		return polymarket.DefaultDataBaseURL
	case strings.Contains(s, "clob"):
		return polymarket.DefaultCLOBBaseURL
	case strings.Contains(s, "bridge"):
		return "https://bridge.polymarket.com"
	case strings.Contains(s, "relayer"):
		return "https://relayer-v2.polymarket.com"
	default:
		p := strings.ToLower(docPath)
		switch {
		case strings.Contains(p, "/gamma/") || strings.Contains(p, "/events/") || strings.Contains(p, "/markets/") || strings.Contains(p, "/tags/") || strings.Contains(p, "/series/") || strings.Contains(p, "/profiles/"):
			return polymarket.DefaultGammaBaseURL
		case strings.Contains(p, "/core/") || strings.Contains(p, "/misc/") || strings.Contains(p, "/builders/"):
			return polymarket.DefaultDataBaseURL
		case strings.Contains(p, "/market-data/") || strings.Contains(p, "/trade/") || strings.Contains(p, "/rewards/") || strings.Contains(p, "/rebates/"):
			return polymarket.DefaultCLOBBaseURL
		case strings.Contains(p, "/bridge/"):
			return "https://bridge.polymarket.com"
		case strings.Contains(p, "/relayer"):
			return "https://relayer-v2.polymarket.com"
		}
	}
	return polymarket.DefaultGammaBaseURL
}

func ensureUniqueKeys(endpoints []endpoint) error {
	seen := map[string]struct{}{}
	for _, ep := range endpoints {
		if _, ok := seen[ep.Key]; ok {
			return fmt.Errorf("duplicate key %q", ep.Key)
		}
		seen[ep.Key] = struct{}{}
	}
	return nil
}

func discoverSamples(ctx context.Context) (discovery, error) {
	client, err := polymarket.NewGammaClient(polymarket.GammaConfig{})
	if err != nil {
		return discovery{}, fmt.Errorf("new gamma client: %w", err)
	}
	one := 1
	out := discovery{TodayUTCDate: time.Now().UTC().Format("2006-01-02")}

	markets, err := client.ListMarkets(ctx, polymarket.ListMarketsOptions{ListOptions: polymarket.ListOptions{Limit: &one}})
	if err != nil {
		return out, fmt.Errorf("discover markets: %w", err)
	}
	if len(markets) == 0 {
		return out, errors.New("discover markets: empty")
	}
	out.MarketID, err = strconv.ParseInt(strings.TrimSpace(markets[0].ID), 10, 64)
	if err != nil {
		return out, fmt.Errorf("parse market id: %w", err)
	}
	if markets[0].Slug != nil {
		out.MarketSlug = strings.TrimSpace(*markets[0].Slug)
	}
	if markets[0].ConditionID != nil {
		out.ConditionID = strings.TrimSpace(*markets[0].ConditionID)
	}
	if markets[0].ClobTokenIDs != nil {
		var tokens []string
		if err := json.Unmarshal([]byte(*markets[0].ClobTokenIDs), &tokens); err == nil && len(tokens) > 0 {
			out.TokenID = strings.TrimSpace(tokens[0])
		}
	}

	events, err := client.ListEvents(ctx, polymarket.ListEventsOptions{ListOptions: polymarket.ListOptions{Limit: &one}})
	if err != nil {
		return out, fmt.Errorf("discover events: %w", err)
	}
	if len(events) == 0 {
		return out, errors.New("discover events: empty")
	}
	out.EventID, err = strconv.ParseInt(strings.TrimSpace(events[0].ID), 10, 64)
	if err != nil {
		return out, fmt.Errorf("parse event id: %w", err)
	}
	if events[0].Slug != nil {
		out.EventSlug = strings.TrimSpace(*events[0].Slug)
	}

	tags, err := client.ListTags(ctx, polymarket.ListTagsOptions{ListOptions: polymarket.ListOptions{Limit: &one}})
	if err != nil {
		return out, fmt.Errorf("discover tags: %w", err)
	}
	if len(tags) == 0 {
		return out, errors.New("discover tags: empty")
	}
	out.TagID = strings.TrimSpace(tags[0].ID)
	if tags[0].Slug != nil {
		out.TagSlug = strings.TrimSpace(*tags[0].Slug)
	}

	comments, err := client.ListComments(ctx, polymarket.ListCommentsOptions{
		ListOptions:      polymarket.ListOptions{Limit: &one},
		ParentEntityType: "Event",
		ParentEntityID:   &out.EventID,
	})
	if err == nil && len(comments) > 0 {
		out.CommentID = strings.TrimSpace(comments[0].ID)
		if comments[0].UserAddress != nil {
			out.UserAddress = strings.TrimSpace(*comments[0].UserAddress)
		}
	}

	seriesRows, err := client.ListSeries(ctx, polymarket.ListSeriesOptions{ListOptions: polymarket.ListOptions{Limit: &one}})
	if err == nil && len(seriesRows) > 0 {
		out.SeriesID, _ = strconv.ParseInt(strings.TrimSpace(seriesRows[0].ID), 10, 64)
	}

	if out.UserAddress == "" {
		out.UserAddress = "0x0000000000000000000000000000000000000000"
	}
	if out.TokenID == "" || out.ConditionID == "" {
		return out, errors.New("discover token/condition sample missing")
	}
	return out, nil
}

func runEndpoint(ctx context.Context, hc *http.Client, outRoot string, ep endpoint, d discovery) indexRecord {
	folder := filepath.Join(outRoot, ep.Key)
	_ = os.MkdirAll(folder, 0o755)

	requestStarted := time.Now().UTC()
	meta := metaLog{EndpointKey: ep.Key, DocSource: ep.DocSource, Result: "fail"}

	urlValue, queryMap, bodyValue, err := buildEndpointRequest(ep, d)
	if err != nil {
		meta.Error = "missing sample: " + err.Error()
		_ = writeJSON(filepath.Join(folder, "meta.json"), meta)
		_ = writeJSON(filepath.Join(folder, "request.json"), requestLog{Method: ep.Method, URL: "", Headers: map[string]string{}, StartedAt: requestStarted.Format(time.RFC3339Nano)})
		_ = writeJSON(filepath.Join(folder, "response.json"), responseLog{EndedAt: time.Now().UTC().Format(time.RFC3339Nano)})
		return indexRecord{EndpointKey: ep.Key, Method: ep.Method, URL: "", Result: "fail", Error: meta.Error, DocSource: ep.DocSource}
	}

	reqLog := requestLog{
		Method:    ep.Method,
		URL:       urlValue,
		Headers:   map[string]string{"Accept": "application/json"},
		Query:     queryMap,
		Body:      bodyValue,
		StartedAt: requestStarted.Format(time.RFC3339Nano),
	}
	_ = writeJSON(filepath.Join(folder, "request.json"), reqLog)

	var bodyReader io.Reader = http.NoBody
	if bodyValue != nil {
		buf, err := json.Marshal(bodyValue)
		if err != nil {
			meta.Error = "marshal request body: " + err.Error()
			_ = writeJSON(filepath.Join(folder, "meta.json"), meta)
			return indexRecord{EndpointKey: ep.Key, Method: ep.Method, URL: urlValue, Result: "fail", Error: meta.Error, DocSource: ep.DocSource}
		}
		bodyReader = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, ep.Method, urlValue, bodyReader)
	if err != nil {
		meta.Error = "new request: " + err.Error()
		_ = writeJSON(filepath.Join(folder, "meta.json"), meta)
		return indexRecord{EndpointKey: ep.Key, Method: ep.Method, URL: urlValue, Result: "fail", Error: meta.Error, DocSource: ep.DocSource}
	}
	req.Header.Set("Accept", "application/json")
	if bodyValue != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	respStart := time.Now()
	resp, err := hc.Do(req)
	if err != nil {
		meta.Error = "http request: " + err.Error()
		_ = writeJSON(filepath.Join(folder, "meta.json"), meta)
		_ = writeJSON(filepath.Join(folder, "response.json"), responseLog{EndedAt: time.Now().UTC().Format(time.RFC3339Nano), LatencyMS: time.Since(respStart).Milliseconds(), RawBody: ""})
		return indexRecord{EndpointKey: ep.Key, Method: ep.Method, URL: urlValue, Result: "fail", Error: meta.Error, DocSource: ep.DocSource}
	}
	defer resp.Body.Close()

	rawBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if readErr != nil {
		meta.Error = "read response: " + readErr.Error()
	}

	respLog := responseLog{
		StatusCode: resp.StatusCode,
		Headers:    map[string][]string(resp.Header),
		LatencyMS:  time.Since(respStart).Milliseconds(),
		EndedAt:    time.Now().UTC().Format(time.RFC3339Nano),
	}
	trimmed := strings.TrimSpace(string(rawBody))
	if trimmed != "" {
		var asJSON any
		if err := json.Unmarshal(rawBody, &asJSON); err == nil {
			respLog.Body = asJSON
		} else {
			respLog.RawBody = trimmed
		}
	}
	_ = writeJSON(filepath.Join(folder, "response.json"), respLog)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 && meta.Error == "" {
		meta.Result = "pass"
		_ = writeJSON(filepath.Join(folder, "meta.json"), meta)
		return indexRecord{EndpointKey: ep.Key, Method: ep.Method, URL: urlValue, StatusCode: resp.StatusCode, Result: "pass", DocSource: ep.DocSource}
	}

	if meta.Error == "" {
		if respLog.RawBody != "" {
			meta.Error = fmt.Sprintf("http %d: %s", resp.StatusCode, respLog.RawBody)
		} else {
			meta.Error = fmt.Sprintf("http %d", resp.StatusCode)
		}
	}
	_ = writeJSON(filepath.Join(folder, "meta.json"), meta)
	return indexRecord{EndpointKey: ep.Key, Method: ep.Method, URL: urlValue, StatusCode: resp.StatusCode, Result: "fail", Error: meta.Error, DocSource: ep.DocSource}
}

func buildEndpointRequest(ep endpoint, d discovery) (string, map[string][]string, any, error) {
	base := strings.TrimRight(strings.TrimSpace(ep.BaseURL), "/")
	if base == "" {
		return "", nil, nil, errors.New("missing base url")
	}

	pathValue := ep.Path
	for _, m := range rePathParam.FindAllStringSubmatch(ep.Path, -1) {
		name := m[1]
		value, err := sampleValue(name, "path", ep, d)
		if err != nil {
			return "", nil, nil, fmt.Errorf("path param %s: %w", name, err)
		}
		pathValue = strings.ReplaceAll(pathValue, "{"+name+"}", url.PathEscape(value))
	}

	u, err := url.Parse(base + pathValue)
	if err != nil {
		return "", nil, nil, err
	}

	q := make(url.Values)
	for _, p := range ep.QueryParams {
		vals, err := sampleQueryValues(p, ep, d)
		if err != nil {
			if p.Required {
				return "", nil, nil, fmt.Errorf("query param %s: %w", p.Name, err)
			}
			continue
		}
		for _, v := range vals {
			q.Add(p.Name, v)
		}
	}
	u.RawQuery = q.Encode()

	body, err := sampleBody(ep, d)
	if err != nil {
		return "", nil, nil, err
	}
	return u.String(), q, body, nil
}

func sampleQueryValues(p param, ep endpoint, d discovery) ([]string, error) {
	v, err := sampleValue(p.Name, "query", ep, d)
	if err == nil {
		if strings.Contains(v, ",") {
			return strings.Split(v, ","), nil
		}
		return []string{v}, nil
	}
	if !p.Required {
		return nil, err
	}
	return nil, err
}

func sampleBody(ep endpoint, d discovery) (any, error) {
	if ep.Method != http.MethodPost {
		return nil, nil
	}
	path := strings.ToLower(ep.Path)
	switch path {
	case "/books", "/midpoints", "/last-trades-prices", "/spreads":
		return []map[string]string{{"token_id": d.TokenID}}, nil
	case "/prices":
		return []map[string]string{{"token_id": d.TokenID, "side": "BUY"}}, nil
	case "/batch-prices-history":
		return map[string]any{"markets": []string{d.TokenID}, "interval": "1d"}, nil
	case "/quote":
		return map[string]any{
			"fromAmountBaseUnit": "1000000",
			"fromChainId":        "137",
			"fromTokenAddress":   "0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359",
			"recipientAddress":   d.UserAddress,
			"toChainId":          "137",
			"toTokenAddress":     "0xC011a7E12a19f7B1f670d46F03B03f3342E82DFB",
		}, nil
	default:
		if ep.RequestBody == nil {
			return nil, nil
		}
		return nil, fmt.Errorf("no automatic body template for %s", ep.Path)
	}
}

func sampleValue(name, in string, ep endpoint, d discovery) (string, error) {
	n := strings.ToLower(strings.TrimSpace(name))
	p := strings.ToLower(ep.Path)

	switch n {
	case "q":
		return "btc", nil
	case "league":
		return "nfl", nil
	case "side":
		return "BUY", nil
	case "sides":
		return "BUY", nil
	case "interval":
		return "1d", nil
	case "date":
		return d.TodayUTCDate, nil
	case "maker_address", "address", "user", "user_address", "recipientaddress":
		if d.UserAddress == "" {
			return "", errors.New("sample user address missing")
		}
		return d.UserAddress, nil
	case "token_id", "asset_id":
		if d.TokenID == "" {
			return "", errors.New("sample token id missing")
		}
		return d.TokenID, nil
	case "token_ids":
		if d.TokenID == "" {
			return "", errors.New("sample token id missing")
		}
		return d.TokenID, nil
	case "market":
		if strings.Contains(p, "prices-history") || strings.Contains(p, "holders") || strings.Contains(p, "positions") || strings.Contains(p, "oi") || strings.Contains(p, "value") {
			if d.ConditionID == "" {
				return "", errors.New("sample condition id missing")
			}
			return d.ConditionID, nil
		}
		if d.TokenID == "" {
			return "", errors.New("sample token id missing")
		}
		return d.TokenID, nil
	case "condition_id", "conditionid":
		if d.ConditionID == "" {
			return "", errors.New("sample condition id missing")
		}
		return d.ConditionID, nil
	case "series_id":
		if d.SeriesID == 0 {
			return "", errors.New("sample series id missing")
		}
		return strconv.FormatInt(d.SeriesID, 10), nil
	case "tag_id":
		if d.TagID == "" {
			return "", errors.New("sample tag id missing")
		}
		return d.TagID, nil
	case "tag_slug":
		if d.TagSlug == "" {
			return "", errors.New("sample tag slug missing")
		}
		return d.TagSlug, nil
	case "parent_entity_type":
		return "Event", nil
	case "parent_entity_id":
		if d.EventID == 0 {
			return "", errors.New("sample event id missing")
		}
		return strconv.FormatInt(d.EventID, 10), nil
	case "slug":
		switch {
		case strings.Contains(p, "/events"):
			if d.EventSlug == "" {
				return "", errors.New("sample event slug missing")
			}
			return d.EventSlug, nil
		case strings.Contains(p, "/markets"):
			if d.MarketSlug == "" {
				return "", errors.New("sample market slug missing")
			}
			return d.MarketSlug, nil
		case strings.Contains(p, "/tags"):
			if d.TagSlug == "" {
				return "", errors.New("sample tag slug missing")
			}
			return d.TagSlug, nil
		}
	case "id":
		switch {
		case strings.Contains(p, "/comments"):
			if d.CommentID == "" {
				return "", errors.New("sample comment id missing")
			}
			return d.CommentID, nil
		case strings.Contains(p, "/events") || strings.Contains(p, "live-volume"):
			if d.EventID == 0 {
				return "", errors.New("sample event id missing")
			}
			return strconv.FormatInt(d.EventID, 10), nil
		case strings.Contains(p, "/series"):
			if d.SeriesID == 0 {
				return "", errors.New("sample series id missing")
			}
			return strconv.FormatInt(d.SeriesID, 10), nil
		case strings.Contains(p, "/tags"):
			if d.TagID == "" {
				return "", errors.New("sample tag id missing")
			}
			return d.TagID, nil
		case strings.Contains(p, "/markets"):
			if d.MarketID == 0 {
				return "", errors.New("sample market id missing")
			}
			return strconv.FormatInt(d.MarketID, 10), nil
		}
	case "fromchainid", "tochainid":
		return "137", nil
	case "fromtokenaddress":
		return "0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359", nil
	case "totokenaddress":
		return "0xC011a7E12a19f7B1f670d46F03B03f3342E82DFB", nil
	case "fromamountbaseunit":
		return "1000000", nil
	case "after_cursor", "next_cursor":
		return "", errors.New("cursor not auto-generated for first page")
	}

	if in == "query" && n == "limit" {
		return "1", nil
	}

	return "", fmt.Errorf("no sample rule")
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}

func sortedMapKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
