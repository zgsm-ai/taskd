package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var lokiURL string

// LogEntry represents a single log entry from Loki.
type LogEntry struct {
	Timestamp time.Time
	Line      string
}

// QueryResponse represents the JSON response from Loki.
type QueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Stream map[string]string `json:"stream"`
			Values [][]string        `json:"values"` // Each value is [timestamp, line]
		} `json:"result"`
	} `json:"data"`
}

func InitLokiLog(lokiURi string) {
	lokiURL = lokiURi
}

func getLokiLog(params url.Values) ([]LogEntry, error) {
	params.Add("direction", "backward")
	addr := fmt.Sprintf("%s/loki/api/v1/query_range?%s", lokiURL, params.Encode())

	// Send the request to Loki
	resp, err := http.Get(addr)
	if err != nil {
		return nil, fmt.Errorf("error sending request to Loki: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	// Unmarshal the JSON response
	var queryResp QueryResponse
	if err := json.Unmarshal(body, &queryResp); err != nil {
		// utils.ReportError(string(body))
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	// Extract log entries from the response
	var entries []LogEntry
	for _, result := range queryResp.Data.Result {
		for _, value := range result.Values {
			// utils.ReportError("%+v", value)
			nanoTimestamp, err := strconv.ParseInt(value[0], 0, 64)
			if err != nil {
				return nil, fmt.Errorf("error parsing timestamp: %w", err)
			}
			// Convert nanosecond timestamp to seconds and nanoseconds
			seconds := nanoTimestamp / int64(time.Second)
			nanoseconds := nanoTimestamp % int64(time.Second)
			// Use time.Unix to get time.Time object
			ts := time.Unix(seconds, nanoseconds)

			entries = append(entries, LogEntry{
				Timestamp: ts,
				Line:      value[1],
			})
		}
	}

	return entries, nil
}

// GetPodLogs queries Loki for logs from a specific pod with a given label.
func GetPodLogs(namespace, podName string, beginTime time.Time, limit uint) ([]LogEntry, error) {
	// Adjust beginTime
	beginTime = adjustBeginTime(beginTime)
	// Construct the query
	query := fmt.Sprintf(`{pod="%s", namespace="%s"}`, podName, namespace)
	params := url.Values{}
	params.Add("query", query)
	params.Add("start", fmt.Sprintf("%d", beginTime.UnixNano()))
	// params.Add("end", fmt.Sprintf("%d", end.UnixNano()))
	if limit == 0 {
		limit = 100
	}
	params.Add("limit", fmt.Sprintf("%d", limit)) // Add limit parameter
	return getLokiLog(params)
}

// GetTaskLogs queries Loki for logs from a specific pod with a given label.
func GetTaskLogs(namespace string, taskID string, limit uint) ([]LogEntry, error) {
	// Construct the query
	query := fmt.Sprintf(`{task_id="%v",namespace="%s"}`, taskID, namespace)
	params := url.Values{}
	params.Add("query", query)

	// Get current time
	now := time.Now()

	// Calculate seconds in past 30 days
	thirtyDaysBefore := now.AddDate(0, 0, -25)

	// Get timestamp (in seconds) from 30 days ago
	timestamp := thirtyDaysBefore.Unix()

	params.Add("start", fmt.Sprintf("%d", timestamp))
	if limit == 0 {
		limit = 100
	}
	params.Add("limit", fmt.Sprintf("%d", limit)) // Add limit parameter

	return getLokiLog(params)
}

func GetPodFollowLogs(namespace string, beginTime time.Time, taskID string, limit uint) (io.ReadCloser, error) {
	// utils.ReportError(namespace, taskID)
	// Adjust beginTime
	beginTime = adjustBeginTime(beginTime)
	// Construct the query
	query := fmt.Sprintf(`{namespace="%s", task_id="%v"}`, namespace, taskID)
	params := url.Values{}
	params.Add("query", query)
	params.Add("follow", "true")
	params.Add("start", fmt.Sprintf("%d", beginTime.UnixNano()))
	if limit == 0 {
		limit = 100
	}
	params.Add("limit", fmt.Sprintf("%d", limit)) // Add limit parameter

	addr := fmt.Sprintf("%s/loki/api/v1/query_range?%s", lokiURL, params.Encode())

	// utils.ReportError(addr)
	// Create request
	req, err := http.NewRequestWithContext(context.Background(), "GET", addr, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request to Loki: %v", err)
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code from Loki: %d", resp.StatusCode)
	}

	// Return response body as io.ReadCloser
	return resp.Body, nil
}

// GetContainerLogs queries Loki for logs from a specific pod with a given label.
func GetContainerLogs(namespace, podName, container string, beginTime time.Time, limit uint) ([]LogEntry, error) {
	// Adjust beginTime
	beginTime = adjustBeginTime(beginTime)
	// Construct the query
	query := fmt.Sprintf(
		`{pod="%s", container="%s", namespace="%s"}`,
		podName, container, namespace)
	params := url.Values{}
	params.Add("query", query)
	params.Add("start", fmt.Sprintf("%d", beginTime.UnixNano()))
	if limit == 0 {
		limit = 100
	}
	params.Add("limit", fmt.Sprintf("%d", limit)) // Add limit parameter

	return getLokiLog(params)
}

// Check beginTime
func adjustBeginTime(beginTime time.Time) time.Time {
	// Get time from 25 days ago
	twentyFiveDaysAgo := time.Now().AddDate(0, 0, -25)
	// Check if beginTime is earlier than 25 days ago
	if beginTime.Before(twentyFiveDaysAgo) {
		return twentyFiveDaysAgo
	}
	return beginTime
}
