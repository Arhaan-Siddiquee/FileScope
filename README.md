# File Analyzer

A Go-based tool to analyze disk usage and file statistics on your computer. Identifies large files, unused files, and categorizes files by type and size.

## Output
![image](https://github.com/user-attachments/assets/be2256cd-e32e-4771-b9d1-5f271ec96c84)


## Features

- 🔍 Scan directories recursively
- 📊 Identify largest files
- ⏱️ Find unused/old files (configurable timeframe)
- 📂 Categorize files by extension
- 📦 Show largest directories
- 🖥️ Cross-platform (Windows, macOS, Linux)

## Installation

1. Ensure you have [Go installed](https://golang.org/dl/) (version 1.16+ recommended)
2. Clone this repository:
   ```bash
   git clone https://github.com/yourusername/file-analyzer.git
   cd file-analyzer
   ```
3. Command Line Options
   ```bash
   Flag	                 Description	                                 Default Value
   -dir	                Directory to analyze	                        Current dir (.)
   -top	                Number of top items to show	                      10
   -min-size       	Minimum file size to consider (bytes)	        1,000,000 (1MB)
   -days-unused    	Consider files unused after X days	              30
   ```
4. Scan your home directory showing top 20 items:
   ```bash
   go run main.go -dir ~/ -top 20
   ```
