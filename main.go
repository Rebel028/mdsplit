package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

// Page holds the content and metadata for each split file segment
type Page struct {
	Filename    string
	HeadingText string
	Level       int // Heading depth (1-6)
	Lines       []string
}

var (
	maxLevel       int
	toc            bool
	navigation     bool
	outputDir      string
	forceFlags     bool
	verbose        bool
	numberedPrefix bool
)

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

var rootCmd = &cobra.Command{
	Use:   "mdsplit [input-file]",
	Short: "Split a markdown file into multiple files by headings",
	Long:  `A robust CLI tool to parse and split monolithic markdown documents into discrete chunks based on ATX headings.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inputFile := args[0]

		if maxLevel < 1 || maxLevel > 6 {
			return fmt.Errorf("max-level must be between 1 and 6")
		}

		targetDir := "."
		if outputDir != "" {
			targetDir = outputDir
			_, err := os.Stat(outputDir)
			if err == nil {
				if !forceFlags {
					return fmt.Errorf("output directory %q already exists (use -f/--force to override)", outputDir)
				}
				if verbose {
					fmt.Printf("Using existing directory: %s\n", targetDir)
				}
			} else if os.IsNotExist(err) {
				if verbose {
					fmt.Printf("Creating output directory: %s\n", targetDir)
				}
				if err := os.MkdirAll(outputDir, 0755); err != nil {
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

		// Tracks if the scanner is currently inside a markdown code block
		inCodeBlock := false

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			trimmedLine := strings.TrimSpace(line)

			// Toggle code block state to prevent parsing `# comments` as headings
			if strings.HasPrefix(trimmedLine, "```") || strings.HasPrefix(trimmedLine, "~~~") {
				inCodeBlock = !inCodeBlock
			}

			var level int
			var headingText string
			var isHeading bool

			// Only look for headings if we are NOT inside a code block
			if !inCodeBlock {
				level, headingText, isHeading = parseHeading(line)
			}

			if isHeading && level <= maxLevel {
				counters[level]++
				for i := level + 1; i <= 6; i++ {
					counters[i] = 0
				}

				baseFilename := sanitizeFilename(headingText)
				if baseFilename == "" {
					baseFilename = "heading"
				}

				// Keep base filename from conflicting with dynamic TOC files
				if (baseFilename == "toc" || baseFilename == "0-toc") && toc {
					baseFilename = "toc-page"
				}

				if numberedPrefix {
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
			if verbose {
				fmt.Println("No content processed.")
			}
			return nil
		}

		// Determine the TOC filename for links and file generation
		tocFilename := "toc.md"
		if numberedPrefix {
			tocFilename = "0-toc.md"
		}

		// Write pages to disk with navigation
		for i, page := range pages {
			if navigation {
				var navParts []string
				if i > 0 {
					navParts = append(navParts, fmt.Sprintf("[← Previous](%s)", pages[i-1].Filename))
				}
				if toc {
					// Link to the correct TOC file
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
			if verbose {
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

		// Generate the Table of Contents file
		if toc {
			tocPath := filepath.Join(targetDir, tocFilename)
			if verbose {
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

				if numberedPrefix && page.Level > 0 {
					bullet = "1. "
				}

				entry := fmt.Sprintf("%s%s[%s](%s)\n", indent, bullet, page.HeadingText, page.Filename)
				if _, err := f.WriteString(entry); err != nil {
					return err
				}
			}
		}

		if verbose {
			fmt.Println("Execution completed cleanly.")
		} else {
			fmt.Println("🚀 Done! Markdown processing completed successfully.")
		}

		return nil
	},
}

func main() {
	rootCmd.Flags().IntVarP(&maxLevel, "max-level", "l", 1, "maximum heading level to split (1-6)")
	rootCmd.Flags().BoolVarP(&toc, "table-of-contents", "t", false, "generate a table of contents")
	rootCmd.Flags().BoolVarP(&navigation, "navigation", "n", false, "add a navigation footer on each page")
	rootCmd.Flags().StringVarP(&outputDir, "output", "o", "", "path to output folder (must not exist)")
	rootCmd.Flags().BoolVarP(&forceFlags, "force", "f", false, "write into output folder even if it already exists")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "enable detailed processing output logs")
	rootCmd.Flags().BoolVarP(&numberedPrefix, "numbered", "u", false, "enable numeric prefix for files and multi-level ordered lists in toc")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
