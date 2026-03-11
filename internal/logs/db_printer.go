package logs

import (
	"fmt"
	"strings"

	"github.com/Des1red/sqlitekit/internal/models"
)

func PrintDBInit(path string, cfg models.Config) {

	values := []string{
		path,
		fmt.Sprintf("%v", cfg.WAL),
		fmt.Sprintf("%v", cfg.ForeignKeys),
		fmt.Sprintf("%d", cfg.MaxOpenConns),
		fmt.Sprintf("%d", cfg.MaxIdleConns),
	}

	max := len("SQLite")

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
}

func PrintDBReady() {
	fmt.Println("│ STATUS   │ ✔ READY                      │")
	fmt.Println("└──────────┴───────────────────────────────┘")
}

func PrintDBFail(err error) {
	fmt.Println("│ STATUS   │ ✖ FAILED                      │")
	fmt.Printf("│ ERROR    │ %v\n", err)
	fmt.Println("└──────────┴───────────────────────────────┘")
}
