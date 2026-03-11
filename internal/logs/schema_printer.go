package logs

import (
	"fmt"
	"strings"
)

const (
	Reset = "\033[0m"

	Green = "\033[32m"
	Gray  = "\033[90m"
	Red   = "\033[31m"
)

func GetWidth(sqlFiles []string) int {

	maxLen := 9 // minimum width for "MIGRATION"

	for _, name := range sqlFiles {
		if len(name) > maxLen {
			maxLen = len(name)
		}
	}

	return maxLen
}

func PrintMigration(status, name string, width int) {

	color := ""
	icon := ""

	switch status {
	case "APPLY":
		color = Green
		icon = "✔"
	case "SKIP":
		color = Gray
		icon = "↺"
	case "FAIL":
		color = Red
		icon = "✖"
	default:
		color = Reset
	}

	fmt.Printf("│ %s%-2s %-6s%s │ %-*s │\n", color, icon, status, Reset, width, name)

	if status == "FAIL" {
		PrintFooter(width)
	}
}

func PrintBox(width int) {

	line := strings.Repeat("─", width+2)

	fmt.Printf("┌──────────┬%s┐\n", line)
	fmt.Printf("│ STATUS   │ %-*s │\n", width, "MIGRATION")
	fmt.Printf("├──────────┼%s┤\n", line)
}

func PrintFooter(width int) {

	line := strings.Repeat("─", width+2)

	fmt.Printf("└──────────┴%s┘\n", line)
}

func PrintSummary(applied, skipped, failed int) {

	fmt.Printf("\n")

	if applied > 0 {
		fmt.Printf("%s✔ %d applied%s  ", Green, applied, Reset)
	}

	if skipped > 0 {
		fmt.Printf("%s↺ %d skipped%s  ", Gray, skipped, Reset)
	}

	if failed > 0 {
		fmt.Printf("%s✖ %d failed%s", Red, failed, Reset)
	}

	fmt.Printf("\n")
}
