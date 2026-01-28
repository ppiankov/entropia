// Test program to demonstrate Wikipedia conflict detection
// This shows edit war detection and historical entity detection working
package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ppiankov/entropia/internal/extract/adapters"
)

func main() {
	fmt.Println("=== Wikipedia Conflict Detection Test ===\n")

	// Test URLs with known conflicts
	testURLs := []string{
		"https://en.wikipedia.org/wiki/Борщ",  // Borscht - contested origin
		"https://en.wikipedia.org/wiki/Borscht", // English version
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, url := range testURLs {
		fmt.Printf("Testing: %s\n", url)
		fmt.Println(strings.Repeat("-", 60))

		// Test edit war detection
		editWar, err := adapters.DetectEditWar(ctx, url)
		if err != nil {
			fmt.Printf("  Edit war check error: %v\n", err)
		} else if editWar.IsHighConflict {
			fmt.Printf("  ⚠️  EDIT WAR DETECTED\n")
			fmt.Printf("     - Recent edits (30 days): %d\n", editWar.RecentEdits)
			fmt.Printf("     - Reverts: %d\n", editWar.RevertCount)
			fmt.Printf("     - Unique editors: %d\n", editWar.UniqueEditors)
			fmt.Printf("     - Edit frequency: %.2f edits/day\n", editWar.EditFrequency)
			fmt.Printf("     - Severity: %s\n", editWar.ConflictSeverity)
			fmt.Printf("     - Last edit: %s\n", editWar.LastEditTime.Format("2006-01-02"))
		} else {
			fmt.Println("  ✓ No significant edit conflict detected")
			if editWar.RecentEdits > 0 {
				fmt.Printf("    (Recent edits: %d, Reverts: %d)\n", editWar.RecentEdits, editWar.RevertCount)
			}
		}

		// Test historical entity detection
		testText := `
		The dish originates from Kyivan Rus in the 13th century.
		According to some sources, it was first documented in Ukraine.
		The Polish-Lithuanian Commonwealth had its own version.
		In the Soviet Union, borscht became a symbol of Eastern European cuisine.
		`

		conflicts := adapters.DetectHistoricalEntities(testText)
		if len(conflicts) > 0 {
			fmt.Printf("\n  ⚠️  HISTORICAL ENTITIES DETECTED: %d\n", len(conflicts))
			for _, hc := range conflicts {
				yearsAgo := 2026 - hc.Entity.EndYear
				fmt.Printf("     - %s (ended %d, %d years ago)\n",
					hc.Entity.Name, hc.Entity.EndYear, yearsAgo)
				fmt.Printf("       %s\n", hc.Entity.Description)
				if len(hc.Context) > 0 {
					fmt.Printf("       Context: %s\n", hc.Context[0])
				}
			}
		}

		fmt.Println()
	}

	fmt.Println("\n=== Test Complete ===")
	fmt.Println("\nNote: These conflict detectors are experimental.")
	fmt.Println("Edit war detection requires Wikipedia API access.")
	fmt.Println("Historical entity detection scans for non-existent states.")
}
