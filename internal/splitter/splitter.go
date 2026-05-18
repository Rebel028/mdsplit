package splitter

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Options holds all configuration for the split operation.
type Options struct {
	MaxLevel       int
	TOC            bool
	Navigation     bool
	OutputDir      string
	Force          bool
	Verbose        bool
	NumberedPrefix bool
}

// Page holds the content and metadata for each split file segment.
type Page struct {
	Filename    string
	HeadingText string
	Level       int
	Lines       []string
}

func sanitizeFilename(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	reg := regexp.MustCompile(`[^a-z0-9\-_]`)
	return reg.ReplaceAllString(s, "")
}

func parseHeading(line string) (int, string, bool) {
	if !strings.HasPrefix(line, "#") {
		return 0, "", false
	}
	count := 0
	for count < len(line) && line[count] == '#' {
		count++
	}
	if count > 6 {
		return 0, "", false
	}
	if count < len(line) && line[count] != ' ' {
		return 0, "", false
	}
	return count, strings.TrimSpace(line[count:]), true
}

// Split parses inputFile and writes split segments into the configured output directory.
func Split(inputFile string, opts Options) error {
	if opts.MaxLevel < 1 || opts.MaxLevel > 6 {
		return fmt.Errorf("max-level must be between 1 and 6")
	}

	targetDir := "."
	if opts.OutputDir != "" {
		targetDir = opts.OutputDir
		_, err := os.Stat(opts.OutputDir)
		if err == nil {
			if !opts.Force {
				return fmt.Errorf("output directory %q already exists (use -f/--force to override)", opts.OutputDir)
			}
			if opts.Verbose {
				fmt.Printf("Using existing directory: %s\n", targetDir)
			}
		} else if os.IsNotExist(err) {
			if opts.Verbose {
				fmt.Printf("Creating output directory: %s\n", targetDir)
			}
			if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}
		} else {
			return err
		}
	}

	file, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer file.Close()

	var pages []*Page
	var currentPage *Page
	usedFilenames := make(map[string]int)
	counters := make([]int, 7)

	inCodeBlock := false

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		// Toggle code block state to prevent parsing `# comments` as headings.
		if strings.HasPrefix(trimmedLine, "```") || strings.HasPrefix(trimmedLine, "~~~") {
			inCodeBlock = !inCodeBlock
		}

		var level int
		var headingText string
		var isHeading bool

		if !inCodeBlock {
			level, headingText, isHeading = parseHeading(line)
		}

		if isHeading && level <= opts.MaxLevel {
			counters[level]++
			for i := level + 1; i <= 6; i++ {
				counters[i] = 0
			}

			baseFilename := sanitizeFilename(headingText)
			if baseFilename == "" {
				baseFilename = "heading"
			}

			// Avoid collision with the dynamically generated TOC file.
			if (baseFilename == "toc" || baseFilename == "0-toc") && opts.TOC {
				baseFilename = "toc-page"
			}

			if opts.NumberedPrefix {
				var prefixParts []string
				for i := 1; i <= level; i++ {
					prefixParts = append(prefixParts, fmt.Sprintf("%d", counters[i]))
				}
				numPrefix := strings.Join(prefixParts, "-")
				baseFilename = fmt.Sprintf("%s-%s", numPrefix, baseFilename)
			}

			filename := baseFilename
			if count, exists := usedFilenames[baseFilename]; exists {
				usedFilenames[baseFilename] = count + 1
				filename = fmt.Sprintf("%s-%d", baseFilename, count+1)
			} else {
				usedFilenames[baseFilename] = 1
			}
			filename += ".md"

			currentPage = &Page{
				Filename:    filename,
				HeadingText: headingText,
				Level:       level,
			}
			pages = append(pages, currentPage)
		}

		if currentPage == nil {
			currentPage = &Page{
				Filename:    "introduction.md",
				HeadingText: "Introduction",
				Level:       0,
			}
			pages = append(pages, currentPage)
		}

		currentPage.Lines = append(currentPage.Lines, line)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading markdown file: %w", err)
	}

	if len(pages) == 0 {
		if opts.Verbose {
			fmt.Println("No content processed.")
		}
		return nil
	}

	tocFilename := "toc.md"
	if opts.NumberedPrefix {
		tocFilename = "0-toc.md"
	}

	for i, page := range pages {
		if opts.Navigation {
			var navParts []string
			if i > 0 {
				navParts = append(navParts, fmt.Sprintf("[← Previous](%s)", pages[i-1].Filename))
			}
			if opts.TOC {
				navParts = append(navParts, fmt.Sprintf("[Table of Contents](%s)", tocFilename))
			}
			if i < len(pages)-1 {
				navParts = append(navParts, fmt.Sprintf("[Next →](%s)", pages[i+1].Filename))
			}
			if len(navParts) > 0 {
				page.Lines = append(page.Lines, "", "---", strings.Join(navParts, " | "))
			}
		}

		filePath := filepath.Join(targetDir, page.Filename)
		if opts.Verbose {
			fmt.Printf("Writing structural segment: %s\n", filePath)
		}

		f, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("failed to create segment file %s: %w", filePath, err)
		}

		for _, l := range page.Lines {
			if _, err := f.WriteString(l + "\n"); err != nil {
				f.Close()
				return fmt.Errorf("failed writing data to %s: %w", filePath, err)
			}
		}
		f.Close()
	}

	if opts.TOC {
		tocPath := filepath.Join(targetDir, tocFilename)
		if opts.Verbose {
			fmt.Printf("Generating multi-level index mapping: %s\n", tocPath)
		}
		f, err := os.Create(tocPath)
		if err != nil {
			return fmt.Errorf("failed to create table of contents file: %w", err)
		}
		defer f.Close()

		if _, err := f.WriteString("# Table of Contents\n\n"); err != nil {
			return err
		}

		for _, page := range pages {
			indent := ""
			bullet := "- "

			if page.Level > 1 {
				indent = strings.Repeat("    ", page.Level-1)
			}

			if opts.NumberedPrefix && page.Level > 0 {
				bullet = "1. "
			}

			entry := fmt.Sprintf("%s%s[%s](%s)\n", indent, bullet, page.HeadingText, page.Filename)
			if _, err := f.WriteString(entry); err != nil {
				return err
			}
		}
	}

	if opts.Verbose {
		fmt.Println("Execution completed cleanly.")
	} else {
		fmt.Println("Done! Markdown processing completed successfully.")
	}

	return nil
}
