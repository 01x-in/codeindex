package tui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Preview displays source context for a selected tree node.
type Preview struct {
	FilePath string
	Line     int
	Lines    []string
	StartLine int
	Visible  bool
	Height   int
	Scroll   int
}

// contextLines is the number of lines to show above and below the symbol.
const contextLines = 5

// LoadPreview reads the source file and extracts context around the given line.
func LoadPreview(filePath string, line int) (Preview, error) {
	if filePath == "" || line <= 0 {
		return Preview{}, fmt.Errorf("invalid file path or line number")
	}

	f, err := os.Open(filePath)
	if err != nil {
		return Preview{}, fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	var allLines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return Preview{}, fmt.Errorf("reading file: %w", err)
	}

	// Clamp line to file bounds to avoid panic on stale nodes.
	if len(allLines) == 0 {
		return Preview{
			FilePath: filePath,
			Line:     line,
			Visible:  true,
		}, nil
	}
	if line > len(allLines) {
		line = len(allLines)
	}

	startLine := line - contextLines
	if startLine < 1 {
		startLine = 1
	}
	endLine := line + contextLines
	if endLine > len(allLines) {
		endLine = len(allLines)
	}

	return Preview{
		FilePath:  filePath,
		Line:      line,
		Lines:     allLines[startLine-1 : endLine],
		StartLine: startLine,
		Visible:   true,
		Height:    endLine - startLine + 1,
	}, nil
}

// Render returns the preview as formatted text.
func (p *Preview) Render(width int, styles Styles) string {
	if !p.Visible || len(p.Lines) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(styles.Dimmed.Render(fmt.Sprintf("  %s:%d", p.FilePath, p.Line)))
	sb.WriteByte('\n')

	for i, line := range p.Lines {
		lineNum := p.StartLine + i
		numStr := fmt.Sprintf("%4d", lineNum)

		if lineNum == p.Line {
			sb.WriteString(styles.Header.Render(fmt.Sprintf("  %s │ %s", numStr, line)))
		} else {
			sb.WriteString(styles.Dimmed.Render(fmt.Sprintf("  %s │ ", numStr)))
			sb.WriteString(line)
		}
		sb.WriteByte('\n')
	}

	return sb.String()
}
