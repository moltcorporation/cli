package output

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

const maxCellWidth = 40

// Print formats and prints data based on the output mode.
// mode is "table", "json", or "raw".
func Print(data []byte, mode string) {
	switch mode {
	case "raw":
		fmt.Print(string(data))
	case "json":
		printJSON(data)
	default:
		printTable(data)
	}
}

func printJSON(data []byte) {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		// Not valid JSON, print as-is
		fmt.Print(string(data))
		return
	}
	pretty, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Print(string(data))
		return
	}
	fmt.Println(string(pretty))
}

func printTable(data []byte) {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		fmt.Print(string(data))
		return
	}

	switch val := v.(type) {
	case []interface{}:
		if len(val) == 0 {
			fmt.Fprintln(os.Stderr, "No results.")
			return
		}
		printArrayTable(val)
	case map[string]interface{}:
		printObjectTable(val)
	default:
		// Scalar or other type — fall back to JSON
		printJSON(data)
	}
}

func printArrayTable(items []interface{}) {
	// Collect column headers from first object
	first, ok := items[0].(map[string]interface{})
	if !ok {
		// Array of non-objects — fall back to JSON
		data, _ := json.MarshalIndent(items, "", "  ")
		fmt.Println(string(data))
		return
	}

	// Stable column order: collect keys and sort with weight heuristic
	var headers []string
	for k := range first {
		headers = append(headers, k)
	}
	sortKeys(headers)

	// Build rows
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		obj, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		row := make([]string, len(headers))
		for i, h := range headers {
			row[i] = formatCell(obj[h])
		}
		rows = append(rows, row)
	}

	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}
	// Cap widths
	for i := range widths {
		if widths[i] > maxCellWidth {
			widths[i] = maxCellWidth
		}
	}

	// Print header
	printRow(headers, widths)
	// Print separator
	sep := make([]string, len(headers))
	for i, w := range widths {
		sep[i] = strings.Repeat("-", w)
	}
	printRow(sep, widths)
	// Print data
	for _, row := range rows {
		printRow(row, widths)
	}
}

func printObjectTable(obj map[string]interface{}) {
	// Key-value display
	maxKey := 0
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
		if len(k) > maxKey {
			maxKey = len(k)
		}
	}
	sortKeys(keys)

	for _, k := range keys {
		val := formatCell(obj[k])
		fmt.Printf("%-*s  %s\n", maxKey, k, val)
	}
}

// sortKeys sorts column/key names with a lightweight weight heuristic:
// id first, name/title second, description/body/content near end, rest alphabetical.
func sortKeys(keys []string) {
	sort.Slice(keys, func(i, j int) bool {
		wi, wj := keyWeight(keys[i]), keyWeight(keys[j])
		if wi != wj {
			return wi < wj
		}
		return keys[i] < keys[j]
	})
}

func keyWeight(k string) int {
	switch strings.ToLower(k) {
	case "id":
		return -2
	case "name", "title":
		return -1
	case "description", "body", "content":
		return 1
	default:
		return 0
	}
}

func printRow(cells []string, widths []int) {
	parts := make([]string, len(cells))
	for i, cell := range cells {
		truncated := truncate(cell, widths[i])
		parts[i] = fmt.Sprintf("%-*s", widths[i], truncated)
	}
	fmt.Println(strings.Join(parts, "  "))
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

func formatCell(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	case bool:
		return fmt.Sprintf("%t", val)
	case map[string]interface{}, []interface{}:
		data, _ := json.Marshal(val)
		return string(data)
	default:
		return fmt.Sprintf("%v", val)
	}
}
