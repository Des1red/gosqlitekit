package logs

import (
	"fmt"
	"strings"

	"github.com/Des1red/gosqlitekit/internal/models"
)

func PrintDBInit(path string, cfg models.Config) int {
	values := []string{
		"SQLite",
		path,
		fmt.Sprintf("%v", cfg.WAL),
		fmt.Sprintf("%v", cfg.ForeignKeys),
		fmt.Sprintf("%d", cfg.MaxOpenConns),
		fmt.Sprintf("%d", cfg.MaxIdleConns),
		"✔ READY",
	}

	max := 0
	for _, v := range values {
		if len(v) > max {
			max = len(v)
		}
	}

	line := strings.Repeat("─", max+2)

	fmt.Println()
	fmt.Printf("┌──────────┬%s┐\n", line)
	fmt.Printf("│ DB INIT  │ %-*s │\n", max, "SQLite")
	fmt.Printf("├──────────┼%s┤\n", line)

	fmt.Printf("│ PATH     │ %-*s │\n", max, path)
	fmt.Printf("│ WAL      │ %-*v │\n", max, cfg.WAL)
	fmt.Printf("│ FKEYS    │ %-*v │\n", max, cfg.ForeignKeys)
	fmt.Printf("│ OPEN     │ %-*d │\n", max, cfg.MaxOpenConns)
	fmt.Printf("│ IDLE     │ %-*d │\n", max, cfg.MaxIdleConns)

	return max
}

func PrintDBReady(width int) {
	fmt.Printf("│ STATUS   │ %-*s │\n", width, "✔ READY")
	fmt.Printf("└──────────┴%s┘\n", strings.Repeat("─", width+2))
}

func PrintDBFail(err error, width int) {
	msg := fmt.Sprintf("✖ FAILED: %v", err)
	if len(msg) > width {
		width = len(msg)
	}

	fmt.Printf("│ STATUS   │ %-*s │\n", width, "✖ FAILED")
	fmt.Printf("│ ERROR    │ %-*v │\n", width, err)
	fmt.Printf("└──────────┴%s┘\n", strings.Repeat("─", width+2))
}
