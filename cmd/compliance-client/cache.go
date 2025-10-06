package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"compliancetoolkit/pkg/api"
)

// SubmissionCache provides local storage for submissions when server is unavailable
type SubmissionCache struct {
	path      string
	maxSizeMB int
	maxAge    time.Duration
}

// NewSubmissionCache creates a new submission cache
func NewSubmissionCache(path string, maxSizeMB int, maxAge time.Duration) (*SubmissionCache, error) {
	// Create cache directory
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return &SubmissionCache{
		path:      path,
		maxSizeMB: maxSizeMB,
		maxAge:    maxAge,
	}, nil
}

// Store saves a submission to the cache
func (c *SubmissionCache) Store(submission *api.ComplianceSubmission) error {
	// Generate filename based on submission ID and timestamp
	filename := fmt.Sprintf("%s_%s.json",
		submission.SubmissionID,
		submission.Timestamp.Format("20060102_150405"),
	)
	filePath := filepath.Join(c.path, filename)

	// Marshal to JSON
	data, err := json.MarshalIndent(submission, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal submission: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// List returns all cached submissions
func (c *SubmissionCache) List() ([]*api.ComplianceSubmission, error) {
	// Read directory
	entries, err := os.ReadDir(c.path)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache directory: %w", err)
	}

	submissions := make([]*api.ComplianceSubmission, 0)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process JSON files
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(c.path, entry.Name())

		// Read file
		data, err := os.ReadFile(filePath)
		if err != nil {
			// Skip files that can't be read
			continue
		}

		// Unmarshal
		var submission api.ComplianceSubmission
		if err := json.Unmarshal(data, &submission); err != nil {
			// Skip invalid files
			continue
		}

		submissions = append(submissions, &submission)
	}

	return submissions, nil
}

// Remove deletes a submission from the cache
func (c *SubmissionCache) Remove(submissionID string) error {
	// Find file by submission ID
	entries, err := os.ReadDir(c.path)
	if err != nil {
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Check if filename starts with submission ID
		if len(entry.Name()) < len(submissionID) {
			continue
		}
		if entry.Name()[:len(submissionID)] != submissionID {
			continue
		}

		// Delete file
		filePath := filepath.Join(c.path, entry.Name())
		if err := os.Remove(filePath); err != nil {
			return fmt.Errorf("failed to remove cache file: %w", err)
		}

		return nil
	}

	return fmt.Errorf("submission not found in cache: %s", submissionID)
}

// Clean removes old and excess cached submissions
func (c *SubmissionCache) Clean() error {
	// Get all cache files
	entries, err := os.ReadDir(c.path)
	if err != nil {
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	now := time.Now()
	totalSize := int64(0)
	files := make([]cacheFile, 0)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		filePath := filepath.Join(c.path, entry.Name())

		// Check age
		age := now.Sub(info.ModTime())
		if age > c.maxAge {
			// Remove old file
			os.Remove(filePath)
			continue
		}

		files = append(files, cacheFile{
			path:    filePath,
			size:    info.Size(),
			modTime: info.ModTime(),
		})
		totalSize += info.Size()
	}

	// Check total size
	maxBytes := int64(c.maxSizeMB) * 1024 * 1024
	if totalSize > maxBytes {
		// Sort by modification time (oldest first)
		sortCacheFilesByAge(files)

		// Remove oldest files until size is under limit
		for _, file := range files {
			if totalSize <= maxBytes {
				break
			}

			os.Remove(file.path)
			totalSize -= file.size
		}
	}

	return nil
}

// Size returns the current cache size in bytes
func (c *SubmissionCache) Size() (int64, error) {
	entries, err := os.ReadDir(c.path)
	if err != nil {
		return 0, fmt.Errorf("failed to read cache directory: %w", err)
	}

	var totalSize int64
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		totalSize += info.Size()
	}

	return totalSize, nil
}

// Count returns the number of cached submissions
func (c *SubmissionCache) Count() (int, error) {
	entries, err := os.ReadDir(c.path)
	if err != nil {
		return 0, fmt.Errorf("failed to read cache directory: %w", err)
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) == ".json" {
			count++
		}
	}

	return count, nil
}

// cacheFile represents a file in the cache
type cacheFile struct {
	path    string
	size    int64
	modTime time.Time
}

// sortCacheFilesByAge sorts cache files by modification time (oldest first)
func sortCacheFilesByAge(files []cacheFile) {
	// Simple bubble sort (fine for small numbers of files)
	n := len(files)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if files[j].modTime.After(files[j+1].modTime) {
				files[j], files[j+1] = files[j+1], files[j]
			}
		}
	}
}
