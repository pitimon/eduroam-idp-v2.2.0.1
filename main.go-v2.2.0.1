/*
Program: eduroam-idp (Identity Provider Accept Analysis)
Version: 2.2.0.1
Description: This program aggregates Access-Accept events for users from a specified domain
             using the Quickwit search engine's aggregation capabilities. It collects data 
             over a specified time range, processes the results, and outputs the aggregated 
             data to a JSON file.

Usage: ./eduroam-idp <domain> [days|Ny|DD-MM-YYYY]
      <domain>: The domain to search for (e.g., 'example.ac.th' or 'etlr1' or 'etlr2')
      [days]: Optional. The number of days (1-3650) to look back from the current date.
      [Ny]: Optional. The number of years (1y-10y) to look back from the current date.
      [DD-MM-YYYY]: Optional. A specific date to process data for.

Features:
- Efficient data aggregation using Quickwit's aggregation queries
- Optimized concurrent processing with worker pools
- Flexible time range specification: number of days or specific date
- Real-time progress reporting with accurate hit counts
- Streamlined output format focusing on essential information
- Enhanced performance through code optimization

Changes in version 2.2.0.1:
- Added context.Context for better cancellation and timeout management
- Defined constants for commonly used values
- Made the number of workers configurable via environment variable
- Improved HTTP connection management with proper timeouts and settings
- Enhanced error handling and more detailed error messages
- Improved logging with log levels
- Added better function documentation in GoDoc format
- Added graceful shutdown with signal handling

Author: [P.Itarun]
Date: [February 25, 2025]
License: [License Information if applicable]
*/

package main

import (
    "bufio"
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "os/signal"
    "sort"
    "strconv"
    "strings"
    "sync"
    "syscall"
    "time"
    "sync/atomic"
)

const (
    // DefaultNumWorkers defines the default number of concurrent workers
    DefaultNumWorkers = 10
    
    // MaxDaysRange defines the maximum number of days supported
    MaxDaysRange = 3650
    
    // MaxYearsRange defines the maximum number of years supported
    MaxYearsRange = 10
    
    // ResultChanBuffer defines the buffer size for the result channel
    ResultChanBuffer = 10000
    
    // DefaultHTTPTimeout defines the default timeout for HTTP requests
    DefaultHTTPTimeout = 30 * time.Second
    
    // PropertiesFile is the filename for authentication properties
    PropertiesFile = "qw-auth.properties"
    
    // DateFormat defines the format for date output
    DateFormat = "2006-01-02"
    
    // DateTimeFormat defines the format for date and time output
    DateTimeFormat = "2006-01-02 15:04:05"
    
    // SpecificDateFormat defines the format for input specific dates
    SpecificDateFormat = "02-01-2006"
)

var (
    // ErrMissingConfiguration indicates missing required configuration
    ErrMissingConfiguration = errors.New("missing required configuration")
    
    // ErrInvalidDateRange indicates an invalid date range was specified
    ErrInvalidDateRange = errors.New("invalid date range")
    
    // ErrNoAggregationsInResponse indicates missing aggregations in the response
    ErrNoAggregationsInResponse = errors.New("no aggregations in response")
)

// Properties represents the authentication properties for Quickwit API
type Properties struct {
    QWUser string
    QWPass string
    QWURL  string
}

// LogEntry represents a single log entry from Quickwit search results
type LogEntry struct {
    Username        string    `json:"username"`
    ServiceProvider string    `json:"service_provider"`
    Timestamp       time.Time `json:"timestamp"`
}

// UserStats contains statistics for a user
type UserStats struct {
    Providers map[string]bool
}

// ProviderStats contains statistics for a service provider
type ProviderStats struct {
    Users map[string]bool
}

// Result holds the aggregated results
type Result struct {
    Users     map[string]*UserStats
    Providers map[string]*ProviderStats
}

// SimplifiedOutputData represents the output JSON structure
type SimplifiedOutputData struct {
    QueryInfo struct {
        Domain    string `json:"domain"`
        Days      int    `json:"days"`
        StartDate string `json:"start_date"`
        EndDate   string `json:"end_date"`
    } `json:"query_info"`
    Description   string `json:"description"`
    Summary       struct {
        TotalUsers     int `json:"total_users"`
        TotalProviders int `json:"total_providers"`
    } `json:"summary"`
    ProviderStats []struct {
        Provider  string   `json:"provider"`
        UserCount int      `json:"user_count"`
        Users     []string `json:"users"`
    } `json:"provider_stats"`
    UserStats []struct {
        Username  string   `json:"username"`
        Providers []string `json:"providers"`
    } `json:"user_stats"`
}

// Job represents a single day's query job
type Job struct {
    StartTimestamp int64
    EndTimestamp   int64
}

// QueryStats tracks the statistics of queries
type QueryStats struct {
    ProcessedDays atomic.Int32
    TotalHits     atomic.Int64
}

// HTTPClient is a wrapper around the standard http.Client with authentication
type HTTPClient struct {
    client *http.Client
    props  Properties
}

// NewHTTPClient creates a new HTTP client with the given properties
func NewHTTPClient(props Properties) *HTTPClient {
    transport := &http.Transport{
        MaxIdleConnsPerHost: 20,
        IdleConnTimeout:     90 * time.Second,
        DisableCompression:  false,
    }
    
    client := &http.Client{
        Timeout:   DefaultHTTPTimeout,
        Transport: transport,
    }
    
    return &HTTPClient{
        client: client,
        props:  props,
    }
}

// SendQuickwitRequest handles HTTP communication with Quickwit
func (c *HTTPClient) SendQuickwitRequest(ctx context.Context, query map[string]interface{}) (map[string]interface{}, error) {
    jsonQuery, err := json.Marshal(query)
    if err != nil {
        return nil, fmt.Errorf("error marshaling query: %w", err)
    }
    
    // Debug output if needed
    if os.Getenv("DEBUG") != "" {
        log.Printf("Query: %s", string(jsonQuery))
    }

    req, err := http.NewRequestWithContext(ctx, "POST", c.props.QWURL+"/api/v1/nro-logs/search", strings.NewReader(string(jsonQuery)))
    if err != nil {
        return nil, fmt.Errorf("error creating request: %w", err)
    }

    req.SetBasicAuth(c.props.QWUser, c.props.QWPass)
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Accept", "application/json")

    resp, err := c.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("error sending request: %w", err)
    }
    defer resp.Body.Close()

    bodyBytes, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("error reading response: %w", err)
    }

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("quickwit error (status %d): %s", resp.StatusCode, string(bodyBytes))
    }

    var result map[string]interface{}
    if err := json.Unmarshal(bodyBytes, &result); err != nil {
        return nil, fmt.Errorf("error decoding response: %w", err)
    }

    if errorMsg, hasError := result["error"].(string); hasError {
        return nil, fmt.Errorf("quickwit error: %s", errorMsg)
    }

    return result, nil
}

// ReadProperties reads the authentication properties from a file
func ReadProperties(filePath string) (Properties, error) {
    file, err := os.Open(filePath)
    if err != nil {
        return Properties{}, fmt.Errorf("failed to open properties file: %w", err)
    }
    defer file.Close()

    props := Properties{}
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := scanner.Text()
        if line != "" && !strings.HasPrefix(line, "#") {
            parts := strings.SplitN(line, "=", 2)
            if len(parts) == 2 {
                key := strings.TrimSpace(parts[0])
                value := strings.TrimSpace(parts[1])
                switch key {
                case "QW_USER":
                    props.QWUser = value
                case "QW_PASS":
                    props.QWPass = value
                case "QW_URL":
                    props.QWURL = strings.TrimPrefix(value, "=")
                }
            }
        }
    }
    
    if err := scanner.Err(); err != nil {
        return Properties{}, fmt.Errorf("error reading properties file: %w", err)
    }
    
    // Validate required properties
    if props.QWUser == "" || props.QWPass == "" || props.QWURL == "" {
        return Properties{}, ErrMissingConfiguration
    }
    
    return props, nil
}

// GetDomain returns the full domain name based on the input
func GetDomain(input string) string {
    switch input {
    case "etlr1":
        return "etlr1.eduroam.org"
    case "etlr2":
        return "etlr2.eduroam.org"
    default:
        return fmt.Sprintf("eduroam.%s", input)
    }
}

// Worker processes a single job
func Worker(ctx context.Context, job Job, resultChan chan<- LogEntry, query map[string]interface{}, client *HTTPClient) (int64, error) {
    // Check for cancellation
    select {
    case <-ctx.Done():
        return 0, ctx.Err()
    default:
    }

    currentQuery := map[string]interface{}{
        "query":           query["query"],
        "start_timestamp": job.StartTimestamp,
        "end_timestamp":   job.EndTimestamp,
        "max_hits":        0,
        "aggs": map[string]interface{}{
            "unique_users": map[string]interface{}{
                "terms": map[string]interface{}{
                    "field": "username",
                    "size":  10000,
                },
                "aggs": map[string]interface{}{
                    "providers": map[string]interface{}{
                        "terms": map[string]interface{}{
                            "field": "service_provider",
                            "size":  1000,
                        },
                    },
                    "daily": map[string]interface{}{
                        "date_histogram": map[string]interface{}{
                            "field":          "timestamp",
                            "fixed_interval": "86400s",
                        },
                    },
                },
            },
        },
    }

    result, err := client.SendQuickwitRequest(ctx, currentQuery)
    if err != nil {
        return 0, err
    }

    return ProcessAggregations(ctx, result, resultChan)
}

// ProcessAggregations processes the aggregation results
func ProcessAggregations(ctx context.Context, result map[string]interface{}, resultChan chan<- LogEntry) (int64, error) {
    // Check for context cancellation
    select {
    case <-ctx.Done():
        return 0, ctx.Err()
    default:
    }

    aggs, ok := result["aggregations"].(map[string]interface{})
    if !ok {
        return 0, ErrNoAggregationsInResponse
    }

    uniqueUsers, ok := aggs["unique_users"].(map[string]interface{})
    if !ok {
        return 0, fmt.Errorf("no unique_users aggregation")
    }

    buckets, ok := uniqueUsers["buckets"].([]interface{})
    if !ok {
        return 0, fmt.Errorf("no buckets in unique_users aggregation")
    }

    var totalHits int64
    for _, bucketInterface := range buckets {
        // Check for context cancellation periodically
        select {
        case <-ctx.Done():
            return totalHits, ctx.Err()
        default:
        }

        bucket, ok := bucketInterface.(map[string]interface{})
        if !ok {
            continue
        }

        username := bucket["key"].(string)
        docCount := int64(bucket["doc_count"].(float64))
        totalHits += docCount

        ProcessUserBucket(ctx, bucket, username, resultChan)
    }

    return totalHits, nil
}

// ProcessUserBucket processes a single user bucket from aggregations
func ProcessUserBucket(ctx context.Context, bucket map[string]interface{}, username string, resultChan chan<- LogEntry) {
    // Check for context cancellation
    select {
    case <-ctx.Done():
        return
    default:
    }

    if providersAgg, ok := bucket["providers"].(map[string]interface{}); ok {
        if providerBuckets, ok := providersAgg["buckets"].([]interface{}); ok {
            for _, providerBucketInterface := range providerBuckets {
                providerBucket, ok := providerBucketInterface.(map[string]interface{})
                if !ok {
                    continue
                }
                provider := providerBucket["key"].(string)
                ProcessUserProviderDaily(ctx, bucket, username, provider, resultChan)
            }
        }
    }
}

// ProcessUserProviderDaily processes daily activities for a user and provider
func ProcessUserProviderDaily(ctx context.Context, bucket map[string]interface{}, username, provider string, resultChan chan<- LogEntry) {
    // Check for context cancellation
    select {
    case <-ctx.Done():
        return
    default:
    }

    if dailyAgg, ok := bucket["daily"].(map[string]interface{}); ok {
        if dailyBuckets, ok := dailyAgg["buckets"].([]interface{}); ok {
            for _, dailyBucketInterface := range dailyBuckets {
                dailyBucket, ok := dailyBucketInterface.(map[string]interface{})
                if !ok || dailyBucket["doc_count"].(float64) == 0 {
                    continue
                }

                timestamp := time.Unix(int64(dailyBucket["key"].(float64)/1000), 0)
                select {
                case resultChan <- LogEntry{
                    Username:        username,
                    ServiceProvider: provider,
                    Timestamp:       timestamp,
                }:
                case <-ctx.Done():
                    return
                }
            }
        }
    }
}

// ProcessResults processes the search results and updates the result struct
func ProcessResults(ctx context.Context, resultChan <-chan LogEntry, result *Result, mu *sync.Mutex) {
    userMap := make(map[string]map[string]bool)
    
    for {
        select {
        case entry, ok := <-resultChan:
            if !ok {
                // Channel closed, finalize results
                FinalizeResults(userMap, result, mu)
                return
            }
            
            if _, exists := userMap[entry.Username]; !exists {
                userMap[entry.Username] = make(map[string]bool)
            }
            userMap[entry.Username][entry.ServiceProvider] = true
            
        case <-ctx.Done():
            // Context cancelled, finalize what we have
            FinalizeResults(userMap, result, mu)
            return
        }
    }
}

// FinalizeResults updates the final result structure from the working maps
func FinalizeResults(userMap map[string]map[string]bool, result *Result, mu *sync.Mutex) {
    mu.Lock()
    defer mu.Unlock()

    for username, providers := range userMap {
        if _, exists := result.Users[username]; !exists {
            result.Users[username] = &UserStats{
                Providers: make(map[string]bool),
            }
        }

        for provider := range providers {
            result.Users[username].Providers[provider] = true
            
            if _, exists := result.Providers[provider]; !exists {
                result.Providers[provider] = &ProviderStats{
                    Users: make(map[string]bool),
                }
            }
            result.Providers[provider].Users[username] = true
        }
    }
}

// CreateOutputData creates the output JSON structure
func CreateOutputData(result *Result, domain string, startDate, endDate time.Time, days int) SimplifiedOutputData {
    output := SimplifiedOutputData{}
    output.QueryInfo.Domain = domain
    output.QueryInfo.Days = days
    output.QueryInfo.StartDate = startDate.Format(DateTimeFormat)
    output.QueryInfo.EndDate = endDate.Format(DateTimeFormat)
    output.Description = "Aggregated Access-Accept events for the specified domain and time range."

    output.Summary.TotalUsers = len(result.Users)
    output.Summary.TotalProviders = len(result.Providers)

    // Process provider stats
    output.ProviderStats = make([]struct {
        Provider  string   `json:"provider"`
        UserCount int      `json:"user_count"`
        Users     []string `json:"users"`
    }, 0, len(result.Providers))

    for provider, stats := range result.Providers {
        users := make([]string, 0, len(stats.Users))
        for user := range stats.Users {
            users = append(users, user)
        }
        sort.Strings(users)
        
        output.ProviderStats = append(output.ProviderStats, struct {
            Provider  string   `json:"provider"`
            UserCount int      `json:"user_count"`
            Users     []string `json:"users"`
        }{
            Provider:  provider,
            UserCount: len(users),
            Users:     users,
        })
    }

    // Sort provider stats by number of users
    sort.Slice(output.ProviderStats, func(i, j int) bool {
        return output.ProviderStats[i].UserCount > output.ProviderStats[j].UserCount
    })

    // Process user stats
    output.UserStats = make([]struct {
        Username  string   `json:"username"`
        Providers []string `json:"providers"`
    }, 0, len(result.Users))

    for username, stats := range result.Users {
        providers := make([]string, 0, len(stats.Providers))
        for provider := range stats.Providers {
            providers = append(providers, provider)
        }
        sort.Strings(providers)
        
        output.UserStats = append(output.UserStats, struct {
            Username  string   `json:"username"`
            Providers []string `json:"providers"`
        }{
            Username:  username,
            Providers: providers,
        })
    }

    // Sort user stats by username
    sort.Slice(output.UserStats, func(i, j int) bool {
        return output.UserStats[i].Username < output.UserStats[j].Username
    })

    return output
}

// ParseDate parses the command line parameter into start and end dates
func ParseDate(param string) (startDate, endDate time.Time, days int, specificDate bool, err error) {
    // Check for year format (Ny)
    if strings.HasSuffix(param, "y") {
        yearStr := strings.TrimSuffix(param, "y")
        if years, err := strconv.Atoi(yearStr); err == nil {
            if years >= 1 && years <= MaxYearsRange {
                days = years * 365
                endDate = time.Now()
                startDate = endDate.AddDate(0, 0, -days+1)
                return startDate, endDate, days, false, nil
            }
            return time.Time{}, time.Time{}, 0, false, fmt.Errorf("invalid year range. Must be between 1y and %dy", MaxYearsRange)
        }
        return time.Time{}, time.Time{}, 0, false, fmt.Errorf("invalid year format. Use 1y-%dy", MaxYearsRange)
    }
    
    // Check for day count
    if d, err := strconv.Atoi(param); err == nil {
        if d >= 1 && d <= MaxDaysRange {
            days = d
            endDate = time.Now()
            startDate = endDate.AddDate(0, 0, -days+1)
            return startDate, endDate, days, false, nil
        }
        return time.Time{}, time.Time{}, 0, false, fmt.Errorf("invalid number of days. Must be between 1 and %d", MaxDaysRange)
    }
    
    // Check for specific date format
    specificDate = true
    startDate, err = time.Parse(SpecificDateFormat, param)
    if err != nil {
        return time.Time{}, time.Time{}, 0, false, fmt.Errorf("invalid date format. Use DD-MM-YYYY: %w", err)
    }
    endDate = startDate.AddDate(0, 0, 1)
    days = 1
    
    return startDate, endDate, days, specificDate, nil
}

// GetNumWorkers returns the number of workers to use, from environment or default
func GetNumWorkers() int {
    if value, exists := os.LookupEnv("NUM_WORKERS"); exists {
        if n, err := strconv.Atoi(value); err == nil && n > 0 {
            return n
        }
    }
    return DefaultNumWorkers
}

// SaveOutputToFile saves the output data to a JSON file
func SaveOutputToFile(outputData SimplifiedOutputData, domain string, startDate time.Time, days int, specificDate bool) (string, error) {
    outputDir := fmt.Sprintf("output/%s", domain)
    if err := os.MkdirAll(outputDir, 0755); err != nil {
        return "", fmt.Errorf("error creating output directory: %w", err)
    }

    currentTime := time.Now().Format("20060102-150405")
    var filename string
    if specificDate {
        filename = fmt.Sprintf("%s/%s-%s.json", outputDir, currentTime, startDate.Format("20060102"))
    } else {
        filename = fmt.Sprintf("%s/%s-%dd.json", outputDir, currentTime, days)
    }

    jsonData, err := json.MarshalIndent(outputData, "", "  ")
    if err != nil {
        return "", fmt.Errorf("error marshaling JSON: %w", err)
    }

    if err := os.WriteFile(filename, jsonData, 0644); err != nil {
        return "", fmt.Errorf("error writing file: %w", err)
    }
    
    return filename, nil
}

func main() {
    // Setup signal handling for graceful shutdown
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    signalChan := make(chan os.Signal, 1)
    signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-signalChan
        log.Println("Received termination signal, shutting down gracefully...")
        cancel()
    }()

    if len(os.Args) < 2 || len(os.Args) > 3 {
        fmt.Println("Usage: ./eduroam-idp <domain> [days|Ny|DD-MM-YYYY]")
        fmt.Println("  domain: domain to search for (e.g., 'example.ac.th', 'etlr1')")
        fmt.Println("  days: number of days (1-3650)")
        fmt.Println("  Ny: number of years (1y-10y)")
        fmt.Println("  DD-MM-YYYY: specific date")
        os.Exit(1)
    }

    domain := os.Args[1]
    var startDate, endDate time.Time
    var days int
    var specificDate bool

    if len(os.Args) == 3 {
        var err error
        startDate, endDate, days, specificDate, err = ParseDate(os.Args[2])
        if err != nil {
            log.Fatalf("Error parsing date parameter: %v", err)
        }
    } else {
        // Default: 1 day
        days = 1
        endDate = time.Now()
        startDate = endDate.AddDate(0, 0, -1)
        specificDate = false
    }

    // Normalize date times to beginning/end of day
    startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
    endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 999999999, endDate.Location())

    props, err := ReadProperties(PropertiesFile)
    if err != nil {
        log.Fatalf("Error reading properties: %v", err)
    }

    httpClient := NewHTTPClient(props)

    if specificDate {
        fmt.Printf("Searching for date: %s\n", startDate.Format(DateFormat))
    } else {
        fmt.Printf("Searching from %s to %s (%d days)\n", 
            startDate.Format(DateFormat), 
            endDate.Format(DateFormat),
            days)
    }

    domainName := GetDomain(domain)
    query := map[string]interface{}{
        "query":           fmt.Sprintf(`message_type:"Access-Accept" AND realm:"%s" NOT service_provider:"client"`, domainName),
        "start_timestamp": startDate.Unix(),
        "end_timestamp":   endDate.Unix(),
        "max_hits":        10000,
    }

    resultChan := make(chan LogEntry, ResultChanBuffer)
    errChan := make(chan error, 1)
    
    stats := &QueryStats{}
    stats.ProcessedDays.Store(0)
    stats.TotalHits.Store(0)
    
    var mu sync.Mutex
    var wg sync.WaitGroup

    jobs := make(chan Job, days)
    numWorkers := GetNumWorkers()

    queryStart := time.Now()
    fmt.Printf("Using %d workers\n", numWorkers)

    // Create result storage
    result := &Result{
        Users:     make(map[string]*UserStats),
        Providers: make(map[string]*ProviderStats),
    }

    // Start workers
    for w := 1; w <= numWorkers; w++ {
        wg.Add(1)
        go func(workerId int) {
            defer wg.Done()
            for job := range jobs {
                select {
                case <-ctx.Done():
                    return
                default:
                }
                
                hits, err := Worker(ctx, job, resultChan, query, httpClient)
                if err != nil {
                    select {
                    case errChan <- fmt.Errorf("worker %d error: %w", workerId, err):
                    default:
                    }
                    return
                }
                
                stats.TotalHits.Add(hits)
                current := stats.ProcessedDays.Add(1)
                
                fmt.Printf("\rProgress: %d/%d days processed, Progress hits: %d", 
                    current, days, stats.TotalHits.Load())
            }
        }(w)
    }

    // Start result processor
    processDone := make(chan struct{})
    go func() {
        ProcessResults(ctx, resultChan, result, &mu)
        close(processDone)
    }()

    // Queue jobs
    currentDate := startDate
    for currentDate.Before(endDate) {
        nextDate := currentDate.Add(24 * time.Hour)
        if nextDate.After(endDate) {
            nextDate = endDate
        }
        select {
        case jobs <- Job{
            StartTimestamp: currentDate.Unix(),
            EndTimestamp:   nextDate.Unix(),
        }:
        case <-ctx.Done():
            break
        }
        currentDate = nextDate
    }
    close(jobs)

    // Wait for workers to finish
    wg.Wait()
    close(resultChan)

    // Wait for processor to finish
    select {
    case <-processDone:
    case <-ctx.Done():
        fmt.Println("\nOperation cancelled.")
        os.Exit(1)
    }

    // Check for errors
    select {
    case err := <-errChan:
        if err != nil {
            log.Fatalf("Error occurred: %v", err)
        }
    default:
    }

    queryDuration := time.Since(queryStart)

    fmt.Printf("\n")
    fmt.Printf("Number of users: %d\n", len(result.Users))
    fmt.Printf("Number of providers: %d\n", len(result.Providers))

    // Create output
    processStart := time.Now()
    outputData := CreateOutputData(result, domain, startDate, endDate, days)
    processDuration := time.Since(processStart)

    // Save output
    filename, err := SaveOutputToFile(outputData, domain, startDate, days, specificDate)
    if err != nil {
        log.Fatalf("Error saving output: %v", err)
    }

    fmt.Printf("Results have been saved to %s\n", filename)
    fmt.Printf("Time taken:\n")
    fmt.Printf("  Quickwit query: %v\n", queryDuration)
    fmt.Printf("  Local processing: %v\n", processDuration)
    fmt.Printf("  Overall: %v\n", time.Since(queryStart))
}