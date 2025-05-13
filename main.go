package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type FileInfo struct {
	Path     string
	Size     int64
	Ext      string
	LastUsed time.Time
}

type AnalysisResults struct {
	TotalFiles        int
	TotalSize         int64
	LargestFiles      []FileInfo
	FilesByExtension  map[string]int
	SizeByExtension   map[string]int64
	OldestFiles       []FileInfo
	DirectorySizes    map[string]int64
}

func main() {
	// Parse command line flags
	dirPtr := flag.String("dir", ".", "Directory to analyze")
	topPtr := flag.Int("top", 10, "Number of top items to show")
	minSizePtr := flag.Int64("min-size", 1000000, "Minimum file size to consider (in bytes)")
	daysUnusedPtr := flag.Int("days-unused", 30, "Consider files unused if not accessed in this many days")
	flag.Parse()

	if *dirPtr == "" {
		fmt.Println("Error: directory path cannot be empty")
		flag.Usage()
		os.Exit(1)
	}

	fmt.Printf("Analyzing files in: %s\n", *dirPtr)
	fmt.Printf("Showing top %d results\n", *topPtr)
	fmt.Printf("Considering files larger than %s as significant\n", formatSize(*minSizePtr))
	fmt.Printf("Considering files unused if not accessed in %d days\n\n", *daysUnusedPtr)

	// Collect file information
	var files []FileInfo
	var totalSize int64
	directorySizes := make(map[string]int64)

	err := filepath.WalkDir(*dirPtr, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Printf("Error accessing %s: %v\n", path, err)
			return nil
		}

		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			fmt.Printf("Error getting info for %s: %v\n", path, err)
			return nil
		}

		// Skip small files if they're below our threshold
		if info.Size() < *minSizePtr {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext == "" {
			ext = "no_extension"
		}

		fileData := FileInfo{
			Path:     path,
			Size:     info.Size(),
			Ext:      ext,
			LastUsed: info.ModTime(), // Using ModTime as fallback
		}

		// Try to get last access time (platform independent way)
		if atime := getAccessTime(info); !atime.IsZero() {
			fileData.LastUsed = atime
		}

		files = append(files, fileData)
		totalSize += info.Size()

		// Track directory sizes
		dir := filepath.Dir(path)
		directorySizes[dir] += info.Size()

		return nil
	})

	if err != nil {
		fmt.Printf("Error walking directory: %v\n", err)
		return
	}

	if len(files) == 0 {
		fmt.Println("No files found matching the criteria.")
		return
	}

	// Analyze the collected data
	results := analyzeFiles(files, totalSize, directorySizes, *daysUnusedPtr)

	// Print results
	printAnalysis(results, *topPtr, *daysUnusedPtr)
}

// Platform independent way to get access time
func getAccessTime(info fs.FileInfo) time.Time {
	// First try using os.File
	file, err := os.Open(info.Name())
	if err != nil {
		return time.Time{}
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return time.Time{}
	}

	// On Unix systems, this will return access time
	// On Windows, it will return the last write time
	return fileInfo.ModTime()
}

func analyzeFiles(files []FileInfo, totalSize int64, directorySizes map[string]int64, daysUnused int) AnalysisResults {
	// Create copies for sorting
	largestFiles := make([]FileInfo, len(files))
	copy(largestFiles, files)
	oldestFiles := make([]FileInfo, len(files))
	copy(oldestFiles, files)

	// Sort files by size (descending)
	sort.Slice(largestFiles, func(i, j int) bool {
		return largestFiles[i].Size > largestFiles[j].Size
	})

	// Sort files by last used (ascending - oldest first)
	sort.Slice(oldestFiles, func(i, j int) bool {
		return oldestFiles[i].LastUsed.Before(oldestFiles[j].LastUsed)
	})

	// Group files by extension
	filesByExt := make(map[string]int)
	sizeByExt := make(map[string]int64)
	for _, file := range files {
		filesByExt[file.Ext]++
		sizeByExt[file.Ext] += file.Size
	}

	return AnalysisResults{
		TotalFiles:        len(files),
		TotalSize:         totalSize,
		LargestFiles:      largestFiles,
		FilesByExtension:  filesByExt,
		SizeByExtension:   sizeByExt,
		OldestFiles:       oldestFiles,
		DirectorySizes:    directorySizes,
	}
}

func printAnalysis(results AnalysisResults, topN int, daysUnused int) {
	fmt.Printf("\n=== General Information ===\n")
	fmt.Printf("Total files analyzed: %d\n", results.TotalFiles)
	fmt.Printf("Total size analyzed: %s\n", formatSize(results.TotalSize))

	fmt.Printf("\n=== Top %d Largest Files ===\n", topN)
	for i := 0; i < topN && i < len(results.LargestFiles); i++ {
		file := results.LargestFiles[i]
		fmt.Printf("%s - %s (Last used: %s)\n", 
			formatSize(file.Size), file.Path, formatTime(file.LastUsed))
	}

	fmt.Printf("\n=== Top %d Oldest/Unused Files (not accessed in %d days) ===\n", topN, daysUnused)
	cutoff := time.Now().AddDate(0, 0, -daysUnused)
	printed := 0
	for _, file := range results.OldestFiles {
		if file.LastUsed.Before(cutoff) {
			fmt.Printf("%s - %s (Last used: %s)\n", 
				formatSize(file.Size), file.Path, formatTime(file.LastUsed))
			printed++
			if printed >= topN {
				break
			}
		}
	}
	if printed == 0 {
		fmt.Println("No files found that haven't been used in this time period.")
	}

	fmt.Printf("\n=== File Extensions by Count ===\n")
	printSortedMap(results.FilesByExtension, topN, func(a, b int) bool { return a > b }, func(v int) string { return fmt.Sprintf("%d files", v) })

	fmt.Printf("\n=== File Extensions by Size ===\n")
	printSortedMap(results.SizeByExtension, topN, func(a, b int64) bool { return a > b }, formatSize)

	fmt.Printf("\n=== Top %d Largest Directories ===\n", topN)
	printSortedMap(results.DirectorySizes, topN, func(a, b int64) bool { return a > b }, formatSize)
}

func printSortedMap[K comparable, V any](m map[K]V, topN int, less func(a, b V) bool, format func(V) string) {
	type kv struct {
		Key   K
		Value V
	}

	var sorted []kv
	for k, v := range m {
		sorted = append(sorted, kv{k, v})
	}

	sort.Slice(sorted, func(i, j int) bool {
		return less(sorted[i].Value, sorted[j].Value)
	})

	for i := 0; i < topN && i < len(sorted); i++ {
		fmt.Printf("%s - %v\n", format(sorted[i].Value), sorted[i].Key)
	}
}

func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(size)/float64(div), "KMGTPE"[exp])
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		return fmt.Sprintf("%.0f minutes ago", diff.Minutes())
	case diff < 24*time.Hour:
		return fmt.Sprintf("%.0f hours ago", diff.Hours())
	case diff < 30*24*time.Hour:
		return fmt.Sprintf("%.0f days ago", diff.Hours()/24)
	case diff < 365*24*time.Hour:
		return fmt.Sprintf("%.0f months ago", diff.Hours()/24/30)
	default:
		return fmt.Sprintf("%.0f years ago", diff.Hours()/24/365)
	}
}