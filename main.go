/*
Program: eduroam-idp (Identity Provider Accept Analysis)
Version: 2.2.0.2
Description: This program aggregates Access-Accept events for users from a specified domain
             using the Quickwit search engine's aggregation capabilities. It collects data 
             over a specified time range, processes the results, and outputs the aggregated 
             data to a JSON or CSV file.

Usage: ./eduroam-idp [flags] <domain> [days|Ny|yxxxx|DD-MM-YYYY]
      <domain>: The domain to search for (e.g., 'example.ac.th' or 'etlr1' or 'etlr2')
      [days]: Optional. The number of days (1-3650) to look back from the current date.
      [Ny]: Optional. The number of years (1y-10y) to look back from the current date.
      [yxxxx]: Optional. A specific year (e.g., 'y2024') to analyze.
      [DD-MM-YYYY]: Optional. A specific date to process data for.

Features:
- Efficient data aggregation using Quickwit's aggregation queries
- Optimized concurrent processing with worker pools
- Flexible time range specification: days, years, specific year, or specific date
- Real-time progress reporting with accurate hit counts
- Multiple output formats (JSON, CSV)
- Streamlined output format focusing on essential information
- Enhanced performance through code optimization

Changes in version 2.2.0.2:
- Added support for yxxxx parameter to specify a specific year (e.g., y2024)
- Added CSV export option with -format flag
- Implemented command-line flags for better configuration
- Enhanced output path handling for different time range specifications
- Improved year handling with leap year detection

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
Date: [February 26, 2025]
License: [License Information if applicable]
*/

package main

import (
    "bufio"
    "context"
    "encoding/csv"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "os/signal"
    "path/filepath"
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
    
    // OutputDirBase is the base directory for output files
    OutputDirBase = "output"
    
    // DefaultOutputFormat is the default output file format
    DefaultOutputFormat = "json"
)

var (
    // ErrMissingConfiguration indicates missing required configuration
    ErrMissingConfiguration = errors.New("missing required configuration")
    
    // ErrInvalidDateRange indicates an invalid date range was specified
    ErrInvalidDateRange = errors.New("invalid date range")
    
    // ErrNoAggregationsInResponse indicates missing aggregations in the response
    ErrNoAggregationsInResponse = errors.New("no aggregations in response")
    
    // ErrInvalidOutputFormat indicates an invalid output format was specified
    ErrInvalidOutputFormat = errors.New("invalid output format")
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
    FirstSeen time.Time
    LastSeen  time.Time
}

// ProviderStats contains statistics for a service provider
type ProviderStats struct {
    Users     map[string]bool
    FirstSeen time.Time
    LastSeen  time.Time
}

// Result holds the aggregated results
type Result struct {
    Users     map[string]*UserStats
    Providers map[string]*ProviderStats
    StartDate time.Time
    EndDate   time.Time
    TotalHits int64
    mu        sync.RWMutex
}

// SimplifiedOutputData represents the output JSON structure
type SimplifiedOutputData struct {
    QueryInfo struct {
        Domain    string `json:"domain"`
        Days      int    `json:"days"`
        StartDate string `json:"start_date"`
        EndDate   string `json:"end_date"`
        TotalHits int64  `json:"total_hits"`
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
        FirstSeen string   `json:"first_seen,omitempty"`
        LastSeen  string   `json:"last_seen,omitempty"`
    } `json:"provider_stats"`
    UserStats []struct {
        Username  string   `json:"username"`
        Providers []string `json:"providers"`
        FirstSeen string   `json:"first_seen,omitempty"`
        LastSeen  string   `json:"last_seen,omitempty"`
    } `json:"user_stats"`
}

// TimeRange represents the time range specification
type TimeRange struct {
    StartDate    time.Time
    EndDate      time.Time
    Days         int
    SpecificDate bool
    SpecificYear bool
    Year         int
}

// Job represents a single day's query job
type Job struct {
    StartTimestamp int64
    EndTimestamp   int64
    Date           time.Time
}

// QueryStats tracks the statistics of queries
type QueryStats struct {
    ProcessedDays atomic.Int32
    TotalHits     atomic.Int64
}

// Config holds the configuration for the program
type Config struct {
    Domain       string
    OutputFormat string
    LogLevel     string
    LogFile      string
    NumWorkers   int
    TimeRange    TimeRange
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

    return ProcessAggregations(ctx, result, resultChan, job.Date)
}

// ProcessAggregations processes the aggregation results
func ProcessAggregations(ctx context.Context, result map[string]interface{}, resultChan chan<- LogEntry, jobDate time.Time) (int64, error) {
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

        ProcessUserBucket(ctx, bucket, username, resultChan, jobDate)
    }

    return totalHits, nil
}

// ProcessUserBucket processes a single user bucket from aggregations
func ProcessUserBucket(ctx context.Context, bucket map[string]interface{}, username string, resultChan chan<- LogEntry, jobDate time.Time) {
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
                ProcessUserProviderDaily(ctx, bucket, username, provider, resultChan, jobDate)
            }
        }
    }
}

// ProcessUserProviderDaily processes daily activities for a user and provider
func ProcessUserProviderDaily(ctx context.Context, bucket map[string]interface{}, username, provider string, resultChan chan<- LogEntry, jobDate time.Time) {
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
                
                // If jobDate is provided, use it to ensure consistent date
                if !jobDate.IsZero() {
                    timestamp = time.Date(
                        jobDate.Year(), jobDate.Month(), jobDate.Day(),
                        timestamp.Hour(), timestamp.Minute(), timestamp.Second(),
                        0, timestamp.Location(),
                    )
                }
                
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
func ProcessResults(ctx context.Context, resultChan <-chan LogEntry, result *Result) {
    userMap := make(map[string]map[string]bool)
    userFirstSeen := make(map[string]time.Time)
    userLastSeen := make(map[string]time.Time)
    providerFirstSeen := make(map[string]time.Time)
    providerLastSeen := make(map[string]time.Time)
    
    for {
        select {
        case entry, ok := <-resultChan:
            if !ok {
                // Channel closed, finalize results
                FinalizeResults(userMap, userFirstSeen, userLastSeen, providerFirstSeen, providerLastSeen, result)
                return
            }
            
            if _, exists := userMap[entry.Username]; !exists {
                userMap[entry.Username] = make(map[string]bool)
                userFirstSeen[entry.Username] = entry.Timestamp
                userLastSeen[entry.Username] = entry.Timestamp
            }
            userMap[entry.Username][entry.ServiceProvider] = true
            
            // Update user's first/last seen
            if entry.Timestamp.Before(userFirstSeen[entry.Username]) {
                userFirstSeen[entry.Username] = entry.Timestamp
            }
            if entry.Timestamp.After(userLastSeen[entry.Username]) {
                userLastSeen[entry.Username] = entry.Timestamp
            }
            
            // Update provider's first/last seen
            if firstSeen, exists := providerFirstSeen[entry.ServiceProvider]; !exists || entry.Timestamp.Before(firstSeen) {
                providerFirstSeen[entry.ServiceProvider] = entry.Timestamp
            }
            if lastSeen, exists := providerLastSeen[entry.ServiceProvider]; !exists || entry.Timestamp.After(lastSeen) {
                providerLastSeen[entry.ServiceProvider] = entry.Timestamp
            }
            
        case <-ctx.Done():
            // Context cancelled, finalize what we have
            FinalizeResults(userMap, userFirstSeen, userLastSeen, providerFirstSeen, providerLastSeen, result)
            return
        }
    }
}

// FinalizeResults updates the final result structure from the working maps
func FinalizeResults(
    userMap map[string]map[string]bool,
    userFirstSeen map[string]time.Time,
    userLastSeen map[string]time.Time,
    providerFirstSeen map[string]time.Time,
    providerLastSeen map[string]time.Time,
    result *Result) {
    
    result.mu.Lock()
    defer result.mu.Unlock()

    for username, providers := range userMap {
        if _, exists := result.Users[username]; !exists {
            result.Users[username] = &UserStats{
                Providers: make(map[string]bool),
                FirstSeen: userFirstSeen[username],
                LastSeen:  userLastSeen[username],
            }
        } else {
            // Update existing user's first/last seen
            if userFirstSeen[username].Before(result.Users[username].FirstSeen) {
                result.Users[username].FirstSeen = userFirstSeen[username]
            }
            if userLastSeen[username].After(result.Users[username].LastSeen) {
                result.Users[username].LastSeen = userLastSeen[username]
            }
        }

        for provider := range providers {
            result.Users[username].Providers[provider] = true
            
            if _, exists := result.Providers[provider]; !exists {
                result.Providers[provider] = &ProviderStats{
                    Users:     make(map[string]bool),
                    FirstSeen: providerFirstSeen[provider],
                    LastSeen:  providerLastSeen[provider],
                }
            } else {
                // Update existing provider's first/last seen
                if providerFirstSeen[provider].Before(result.Providers[provider].FirstSeen) {
                    result.Providers[provider].FirstSeen = providerFirstSeen[provider]
                }
                if providerLastSeen[provider].After(result.Providers[provider].LastSeen) {
                    result.Providers[provider].LastSeen = providerLastSeen[provider]
                }
            }
            result.Providers[provider].Users[username] = true
        }
    }
}

// CreateOutputData creates the output JSON structure
func CreateOutputData(result *Result, domain string, timeRange TimeRange) SimplifiedOutputData {
    output := SimplifiedOutputData{}
    output.QueryInfo.Domain = domain
    output.QueryInfo.Days = timeRange.Days
    output.QueryInfo.StartDate = timeRange.StartDate.Format(DateTimeFormat)
    output.QueryInfo.EndDate = timeRange.EndDate.Format(DateTimeFormat)
    output.QueryInfo.TotalHits = result.TotalHits
    output.Description = "Aggregated Access-Accept events for the specified domain and time range."

    result.mu.RLock()
    defer result.mu.RUnlock()

    output.Summary.TotalUsers = len(result.Users)
    output.Summary.TotalProviders = len(result.Providers)

    // Process provider stats
    output.ProviderStats = make([]struct {
        Provider  string   `json:"provider"`
        UserCount int      `json:"user_count"`
        Users     []string `json:"users"`
        FirstSeen string   `json:"first_seen,omitempty"`
        LastSeen  string   `json:"last_seen,omitempty"`
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
            FirstSeen string   `json:"first_seen,omitempty"`
            LastSeen  string   `json:"last_seen,omitempty"`
        }{
            Provider:  provider,
            UserCount: len(users),
            Users:     users,
            FirstSeen: stats.FirstSeen.Format(DateFormat),
            LastSeen:  stats.LastSeen.Format(DateFormat),
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
        FirstSeen string   `json:"first_seen,omitempty"`
        LastSeen  string   `json:"last_seen,omitempty"`
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
            FirstSeen string   `json:"first_seen,omitempty"`
            LastSeen  string   `json:"last_seen,omitempty"`
        }{
            Username:  username,
            Providers: providers,
            FirstSeen: stats.FirstSeen.Format(DateFormat),
            LastSeen:  stats.LastSeen.Format(DateFormat),
        })
    }

    // Sort user stats by username
    sort.Slice(output.UserStats, func(i, j int) bool {
        return output.UserStats[i].Username < output.UserStats[j].Username
    })

    return output
}

// ParseTimeRange parses the command line parameter into a TimeRange struct
func ParseTimeRange(param string) (TimeRange, error) {
    var timeRange TimeRange
    
    // Check for year format (yxxxx)
    if strings.HasPrefix(param, "y") && len(param) == 5 {
        yearStr := param[1:]
        if year, err := strconv.Atoi(yearStr); err == nil {
            if year >= 2000 && year <= 2100 {
                timeRange.SpecificYear = true
                timeRange.Year = year
                timeRange.StartDate = time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
                timeRange.EndDate = time.Date(year, 12, 31, 23, 59, 59, 999999999, time.Local)
                
                // Calculate days (accounting for leap years)
                timeRange.Days = 365
                if isLeapYear(year) {
                    timeRange.Days = 366
                }
                
                return timeRange, nil
            }
            return timeRange, fmt.Errorf("invalid year range. Must be between 2000 and 2100")
        }
        return timeRange, fmt.Errorf("invalid year format. Use y followed by 4 digits (e.g., y2024)")
    }
    
    // Check for year format (Ny)
    if strings.HasSuffix(param, "y") {
        yearStr := strings.TrimSuffix(param, "y")
        if years, err := strconv.Atoi(yearStr); err == nil {
            if years >= 1 && years <= MaxYearsRange {
                timeRange.Days = years * 365
                timeRange.EndDate = time.Now()
                timeRange.StartDate = timeRange.EndDate.AddDate(-years, 0, 0)
                return timeRange, nil
            }
            return timeRange, fmt.Errorf("invalid year range. Must be between 1y and %dy", MaxYearsRange)
        }
        return timeRange, fmt.Errorf("invalid year format. Use 1y-%dy", MaxYearsRange)
    }
    
    // Check for day count
    if d, err := strconv.Atoi(param); err == nil {
        if d >= 1 && d <= MaxDaysRange {
            timeRange.Days = d
            timeRange.EndDate = time.Now()
            timeRange.StartDate = timeRange.EndDate.AddDate(0, 0, -d+1)
            return timeRange, nil
        }
        return timeRange, fmt.Errorf("invalid number of days. Must be between 1 and %d", MaxDaysRange)
    }
    
    // Check for specific date format
    timeRange.SpecificDate = true
    var err error
    timeRange.StartDate, err = time.Parse(SpecificDateFormat, param)
    if err != nil {
        return timeRange, fmt.Errorf("invalid date format. Use DD-MM-YYYY: %w", err)
    }
    timeRange.EndDate = timeRange.StartDate.AddDate(0, 0, 1)
    timeRange.Days = 1
    
    return timeRange, nil
}

// isLeapYear checks if a year is a leap year
func isLeapYear(year int) bool {
    return year%4 == 0 && (year%100 != 0 || year%400 == 0)
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

// SaveOutputToJSON saves the output data to a JSON file
func SaveOutputToJSON(outputData SimplifiedOutputData, domain string, timeRange TimeRange) (string, error) {
    outputDir := filepath.Join(OutputDirBase, domain)
    if err := os.MkdirAll(outputDir, 0755); err != nil {
        return "", fmt.Errorf("error creating output directory: %w", err)
    }

    currentTime := time.Now().Format("20060102-150405")
    var filename string
    
    if timeRange.SpecificDate {
        filename = fmt.Sprintf("%s/%s-%s.json", outputDir, currentTime, timeRange.StartDate.Format("20060102"))
    } else if timeRange.SpecificYear {
        filename = fmt.Sprintf("%s/%s-y%d.json", outputDir, currentTime, timeRange.Year)
    } else {
        filename = fmt.Sprintf("%s/%s-%dd.json", outputDir, currentTime, timeRange.Days)
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

// ExportToCSV exports the results to CSV files
func ExportToCSV(result *Result, domain string, timeRange TimeRange) ([]string, error) {
    outputDir := filepath.Join(OutputDirBase, domain)
    if err := os.MkdirAll(outputDir, 0755); err != nil {
        return nil, fmt.Errorf("error creating output directory: %w", err)
    }

    currentTime := time.Now().Format("20060102-150405")
    var baseFilename string
    
    if timeRange.SpecificDate {
        baseFilename = fmt.Sprintf("%s-%s", currentTime, timeRange.StartDate.Format("20060102"))
    } else if timeRange.SpecificYear {
        baseFilename = fmt.Sprintf("%s-y%d", currentTime, timeRange.Year)
    } else {
        baseFilename = fmt.Sprintf("%s-%dd", currentTime, timeRange.Days)
    }
    
    // Create users CSV file
    usersFilename := filepath.Join(outputDir, baseFilename+"-users.csv")
    usersFile, err := os.Create(usersFilename)
    if err != nil {
        return nil, fmt.Errorf("error creating users CSV file: %w", err)
    }
    defer usersFile.Close()

    usersWriter := csv.NewWriter(usersFile)
    defer usersWriter.Flush()

    // Write users CSV header
    if err := usersWriter.Write([]string{"Username", "Providers Count", "Providers", "First Seen", "Last Seen"}); err != nil {
        return nil, fmt.Errorf("error writing users CSV header: %w", err)
    }

    // Write users data
    result.mu.RLock()
    for username, stats := range result.Users {
        providers := make([]string, 0, len(stats.Providers))
        for provider := range stats.Providers {
            providers = append(providers, provider)
        }
        sort.Strings(providers)
        
        record := []string{
            username,
            strconv.Itoa(len(providers)),
            strings.Join(providers, "; "),
            stats.FirstSeen.Format(DateFormat),
            stats.LastSeen.Format(DateFormat),
        }
        if err := usersWriter.Write(record); err != nil {
            result.mu.RUnlock()
            return nil, fmt.Errorf("error writing user record: %w", err)
        }
    }
    
    // Create providers CSV file
    providersFilename := filepath.Join(outputDir, baseFilename+"-providers.csv")
    providersFile, err := os.Create(providersFilename)
    if err != nil {
        result.mu.RUnlock()
        return nil, fmt.Errorf("error creating providers CSV file: %w", err)
    }
    defer providersFile.Close()

    providersWriter := csv.NewWriter(providersFile)
    defer providersWriter.Flush()

    // Write providers CSV header
    if err := providersWriter.Write([]string{"Provider", "Users Count", "First Seen", "Last Seen"}); err != nil {
        result.mu.RUnlock()
        return nil, fmt.Errorf("error writing providers CSV header: %w", err)
    }

    // Write providers data
    for provider, stats := range result.Providers {
        record := []string{
            provider,
            strconv.Itoa(len(stats.Users)),
            stats.FirstSeen.Format(DateFormat),
            stats.LastSeen.Format(DateFormat),
        }
        if err := providersWriter.Write(record); err != nil {
            result.mu.RUnlock()
            return nil, fmt.Errorf("error writing provider record: %w", err)
        }
    }
    result.mu.RUnlock()
    
    // Create summary CSV file
    summaryFilename := filepath.Join(outputDir, baseFilename+"-summary.csv")
    summaryFile, err := os.Create(summaryFilename)
    if err != nil {
        return nil, fmt.Errorf("error creating summary CSV file: %w", err)
    }
    defer summaryFile.Close()

    summaryWriter := csv.NewWriter(summaryFile)
    defer summaryWriter.Flush()

    // Write summary CSV header and data
    if err := summaryWriter.Write([]string{"Parameter", "Value"}); err != nil {
        return nil, fmt.Errorf("error writing summary CSV header: %w", err)
    }
    
    summaryData := [][]string{
        {"Domain", domain},
        {"Start Date", timeRange.StartDate.Format(DateTimeFormat)},
        {"End Date", timeRange.EndDate.Format(DateTimeFormat)},
        {"Total Days", strconv.Itoa(timeRange.Days)},
        {"Total Users", strconv.Itoa(len(result.Users))},
        {"Total Providers", strconv.Itoa(len(result.Providers))},
        {"Total Hits", strconv.FormatInt(result.TotalHits, 10)},
        {"Exported At", time.Now().Format(DateTimeFormat)},
    }
    
    for _, record := range summaryData {
        if err := summaryWriter.Write(record); err != nil {
            return nil, fmt.Errorf("error writing summary record: %w", err)
        }
    }
    
    return []string{usersFilename, providersFilename, summaryFilename}, nil
}

func main() {
    // Define command line flags
    outputFormat := flag.String("format", DefaultOutputFormat, "Output format (json or csv)")
    configFile := flag.String("config", PropertiesFile, "Path to configuration file")
    // Defined but not implemented yet in this version - ignoring in code to avoid compile errors
    _ = flag.String("log-level", "info", "Log level (error, warn, info, debug)")
    _ = flag.String("log-file", "", "Path to log file")
    numWorkers := flag.Int("workers", 0, "Number of worker goroutines (overrides environment variable)")
    
    // Parse flags
    flag.Parse()
    
    // Validate output format
    if *outputFormat != "json" && *outputFormat != "csv" {
        fmt.Fprintf(os.Stderr, "Error: Invalid output format. Must be 'json' or 'csv'.\n")
        os.Exit(1)
    }
    
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

    // Check remaining arguments
    args := flag.Args()
    if len(args) < 1 || len(args) > 2 {
        fmt.Println("Usage: ./eduroam-idp [flags] <domain> [days|Ny|yxxxx|DD-MM-YYYY]")
        fmt.Println("  <domain>: domain to search for (e.g., 'example.ac.th', 'etlr1')")
        fmt.Println("  [days]: number of days (1-3650)")
        fmt.Println("  [Ny]: number of years (1y-10y)")
        fmt.Println("  [yxxxx]: specific year (e.g., y2024)")
        fmt.Println("  [DD-MM-YYYY]: specific date")
        fmt.Println()
        fmt.Println("Flags:")
        flag.PrintDefaults()
        os.Exit(1)
    }

    domain := args[0]
    var timeRange TimeRange

    if len(args) == 2 {
        var err error
        timeRange, err = ParseTimeRange(args[1])
        if err != nil {
            log.Fatalf("Error parsing time range parameter: %v", err)
        }
    } else {
        // Default: 1 day
        timeRange.Days = 1
        timeRange.EndDate = time.Now()
        timeRange.StartDate = timeRange.EndDate.AddDate(0, 0, -1)
    }

    // Normalize date times to beginning/end of day
    timeRange.StartDate = time.Date(timeRange.StartDate.Year(), timeRange.StartDate.Month(), timeRange.StartDate.Day(), 0, 0, 0, 0, timeRange.StartDate.Location())
    timeRange.EndDate = time.Date(timeRange.EndDate.Year(), timeRange.EndDate.Month(), timeRange.EndDate.Day(), 23, 59, 59, 999999999, timeRange.EndDate.Location())

    props, err := ReadProperties(*configFile)
    if err != nil {
        log.Fatalf("Error reading properties: %v", err)
    }

    httpClient := NewHTTPClient(props)

    // Display query parameters
    if timeRange.SpecificDate {
        fmt.Printf("Searching for date: %s\n", timeRange.StartDate.Format(DateFormat))
    } else if timeRange.SpecificYear {
        fmt.Printf("Searching for year: %d\n", timeRange.Year)
    } else {
        fmt.Printf("Searching from %s to %s (%d days)\n", 
            timeRange.StartDate.Format(DateFormat), 
            timeRange.EndDate.Format(DateFormat),
            timeRange.Days)
    }

    domainName := GetDomain(domain)
    query := map[string]interface{}{
        "query":           fmt.Sprintf(`message_type:"Access-Accept" AND realm:"%s" NOT service_provider:"client"`, domainName),
        "start_timestamp": timeRange.StartDate.Unix(),
        "end_timestamp":   timeRange.EndDate.Unix(),
        "max_hits":        10000,
    }

    resultChan := make(chan LogEntry, ResultChanBuffer)
    errChan := make(chan error, 1)
    
    stats := &QueryStats{}
    stats.ProcessedDays.Store(0)
    stats.TotalHits.Store(0)
    
    var wg sync.WaitGroup

    // Determine workers count
    workersCount := GetNumWorkers()
    if *numWorkers > 0 {
        workersCount = *numWorkers
    }

    jobs := make(chan Job, timeRange.Days)

    queryStart := time.Now()
    fmt.Printf("Using %d workers\n", workersCount)

    // Create result storage
    result := &Result{
        Users:     make(map[string]*UserStats),
        Providers: make(map[string]*ProviderStats),
        StartDate: timeRange.StartDate,
        EndDate:   timeRange.EndDate,
    }

    // Start workers
    for w := 1; w <= workersCount; w++ {
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
                    current, timeRange.Days, stats.TotalHits.Load())
            }
        }(w)
    }

    // Start result processor
    processDone := make(chan struct{})
    go func() {
        ProcessResults(ctx, resultChan, result)
        close(processDone)
    }()

    // Queue jobs
    currentDate := timeRange.StartDate
    for currentDate.Before(timeRange.EndDate) {
        nextDate := currentDate.Add(24 * time.Hour)
        if nextDate.After(timeRange.EndDate) {
            nextDate = timeRange.EndDate
        }
        select {
        case jobs <- Job{
            StartTimestamp: currentDate.Unix(),
            EndTimestamp:   nextDate.Unix(),
            Date:           currentDate,
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

    // Store final total hits
    result.TotalHits = stats.TotalHits.Load()

    queryDuration := time.Since(queryStart)

    fmt.Printf("\n")
    fmt.Printf("Number of users: %d\n", len(result.Users))
    fmt.Printf("Number of providers: %d\n", len(result.Providers))
    fmt.Printf("Total hits: %d\n", result.TotalHits)

    // Export according to format
    exportStart := time.Now()
    if *outputFormat == "csv" {
        filenames, err := ExportToCSV(result, domain, timeRange)
        if err != nil {
            log.Fatalf("Error exporting to CSV: %v", err)
        }
        fmt.Printf("Results have been saved to:\n")
        for _, filename := range filenames {
            fmt.Printf("  - %s\n", filename)
        }
    } else {
        // Create output
        outputData := CreateOutputData(result, domain, timeRange)
        
        // Save output
        filename, err := SaveOutputToJSON(outputData, domain, timeRange)
        if err != nil {
            log.Fatalf("Error saving output: %v", err)
        }
        
        fmt.Printf("Results have been saved to %s\n", filename)
    }
    
    exportDuration := time.Since(exportStart)

    fmt.Printf("Time taken:\n")
    fmt.Printf("  Quickwit query: %v\n", queryDuration)
    fmt.Printf("  Export processing: %v\n", exportDuration)
    fmt.Printf("  Overall: %v\n", time.Since(queryStart))
}