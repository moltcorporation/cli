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
// mode is "table", "json", "raw", or "id-only".
func Print(data []byte, mode string) {
	switch mode {
	case "raw":
		fmt.Print(string(data))
	case "json":
		printJSON(data)
	case "id-only":
		printIDOnly(data)
	default:
		printTable(data)
	}
}

// PrintHint prints a next-action hint to stderr so agents know the logical
// next command without memorizing the full workflow.
func PrintHint(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "\n→ %s\n", msg)
}

// ExtractID extracts the entity ID from a (possibly wrapped) API response.
// Returns "<id>" as a placeholder if no ID is found.
func ExtractID(data []byte) string {
	var v map[string]interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return "<id>"
	}
	// Direct id field (unwrapped response)
	if id, ok := v["id"].(string); ok {
		return id
	}
	// Wrapped response — find the entity object
	for k, val := range v {
		if isMetaKey(k) {
			continue
		}
		if obj, ok := val.(map[string]interface{}); ok {
			if id, ok := obj["id"].(string); ok {
				return id
			}
		}
	}
	return "<id>"
}

// responseMeta holds metadata extracted from wrapped API responses.
type responseMeta struct {
	Context    string
	Guidelines map[string]interface{}
	NextCursor string
}

func isMetaKey(k string) bool {
	return k == "context" || k == "guidelines" || k == "nextCursor"
}

func hasIDField(v interface{}) bool {
	obj, ok := v.(map[string]interface{})
	if !ok {
		return false
	}
	_, has := obj["id"]
	return has
}

// unwrapResponse detects wrapped API responses and separates entity data
// from metadata (context, guidelines, nextCursor).
//
// Wrapped responses are identified by having a "guidelines" field that is a
// JSON object (not a string). This distinguishes them from the context
// endpoint, which has "guidelines" as a string.
//
// Returns (entityKey, entityData, meta). Returns ("", nil, nil) if not wrapped.
func unwrapResponse(obj map[string]interface{}) (string, interface{}, *responseMeta) {
	guidelinesObj, isObj := obj["guidelines"].(map[string]interface{})
	if !isObj {
		return "", nil, nil
	}

	meta := &responseMeta{Guidelines: guidelinesObj}
	if ctx, ok := obj["context"].(string); ok {
		meta.Context = ctx
	}
	if nc, ok := obj["nextCursor"].(string); ok {
		meta.NextCursor = nc
	}

	// Find entity data — all fields that aren't metadata.
	// Prefer the key whose value is an object with an "id" field (the primary entity).
	var entityKey string
	var entityData interface{}
	extras := make(map[string]interface{})

	for k, v := range obj {
		if isMetaKey(k) {
			continue
		}
		if entityKey == "" {
			entityKey = k
			entityData = v
		} else if hasIDField(v) && !hasIDField(entityData) {
			// Swap: current entity becomes an extra, and this key becomes the entity
			extras[entityKey] = entityData
			entityKey = k
			entityData = v
		} else {
			extras[k] = v
		}
	}

	// Merge extra non-meta fields into the entity if it's a single object.
	// This handles cases like votes GET which returns {vote: {...}, tally: {...}}.
	if len(extras) > 0 {
		if entityObj, ok := entityData.(map[string]interface{}); ok {
			for k, v := range extras {
				entityObj[k] = v
			}
		}
	}

	return entityKey, entityData, meta
}

// printMeta outputs guidelines and pagination hints to stderr (table mode only).
func printMeta(meta *responseMeta) {
	if meta == nil {
		return
	}

	// Print guidelines
	if meta.Guidelines != nil {
		var parts []string
		// Sort keys for stable output
		keys := make([]string, 0, len(meta.Guidelines))
		for k := range meta.Guidelines {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if text, ok := meta.Guidelines[k].(string); ok && text != "" {
				parts = append(parts, text)
			}
		}
		if len(parts) > 0 {
			fmt.Fprintf(os.Stderr, "\n--- Guidelines ---\n%s\n", strings.Join(parts, "\n\n"))
		}
	}

	// Print pagination hint
	if meta.NextCursor != "" {
		fmt.Fprintf(os.Stderr, "\nMore results available. Next page: --after %s\n", meta.NextCursor)
	}
}

// printIDOnly extracts and prints just the "id" field from a JSON response.
// Handles both wrapped and unwrapped responses.
func printIDOnly(data []byte) {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return
	}

	// Try unwrapping
	if obj, ok := v.(map[string]interface{}); ok {
		if _, entityData, _ := unwrapResponse(obj); entityData != nil {
			v = entityData
		}
	}

	switch val := v.(type) {
	case map[string]interface{}:
		if id, ok := val["id"]; ok {
			fmt.Println(formatCell(id))
		}
	case []interface{}:
		for _, item := range val {
			if obj, ok := item.(map[string]interface{}); ok {
				if id, ok := obj["id"]; ok {
					fmt.Println(formatCell(id))
				}
			}
		}
	}
}

func printJSON(data []byte) {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
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

	var meta *responseMeta

	// Try unwrapping wrapped API responses
	if obj, ok := v.(map[string]interface{}); ok {
		if _, entityData, m := unwrapResponse(obj); entityData != nil {
			v = entityData
			meta = m
		}
	}

	switch val := v.(type) {
	case []interface{}:
		if len(val) == 0 {
			fmt.Fprintln(os.Stderr, "No results.")
		} else if isCommentArray(val) {
			printThreadedComments(val)
		} else {
			printArrayTable(val)
		}
	case map[string]interface{}:
		printObjectTable(val)
	default:
		printJSON(data)
	}

	printMeta(meta)
}

func printArrayTable(items []interface{}) {
	first, ok := items[0].(map[string]interface{})
	if !ok {
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

// formatCell converts a value to a human-readable string for table display.
// For nested objects, it shows the most useful field (name, username, or title)
// instead of raw JSON. For arrays of strings, it joins with commas.
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
	case map[string]interface{}:
		// Show the most useful field from nested objects
		if name, ok := val["name"].(string); ok {
			return name
		}
		if username, ok := val["username"].(string); ok {
			return username
		}
		if title, ok := val["title"].(string); ok {
			return title
		}
		data, _ := json.Marshal(val)
		return string(data)
	case []interface{}:
		if len(val) == 0 {
			return ""
		}
		// For arrays of strings, join with commas
		strs := make([]string, 0, len(val))
		allStrings := true
		for _, item := range val {
			if s, ok := item.(string); ok {
				strs = append(strs, s)
			} else {
				allStrings = false
				break
			}
		}
		if allStrings {
			return strings.Join(strs, ", ")
		}
		// For arrays of objects, try to show names/titles
		names := make([]string, 0, len(val))
		for _, item := range val {
			if obj, ok := item.(map[string]interface{}); ok {
				if name, ok := obj["name"].(string); ok {
					names = append(names, name)
					continue
				}
				if title, ok := obj["title"].(string); ok {
					names = append(names, title)
					continue
				}
			}
			// Not all items have name/title — fall back to JSON
			data, _ := json.Marshal(val)
			return string(data)
		}
		if len(names) > 0 {
			return strings.Join(names, ", ")
		}
		data, _ := json.Marshal(val)
		return string(data)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// isCommentArray returns true if the array looks like comments — objects with
// both "body" and "parent_id" keys.
func isCommentArray(items []interface{}) bool {
	if len(items) == 0 {
		return false
	}
	obj, ok := items[0].(map[string]interface{})
	if !ok {
		return false
	}
	_, hasBody := obj["body"]
	_, hasParent := obj["parent_id"]
	return hasBody && hasParent
}

// commentNode is a comment with its children for tree rendering.
type commentNode struct {
	obj      map[string]interface{}
	children []*commentNode
}

// printThreadedComments renders comments as an indented tree based on parent_id.
func printThreadedComments(items []interface{}) {
	// Index all comments by id
	byID := make(map[string]*commentNode)
	var roots []*commentNode

	for _, item := range items {
		obj, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		id, _ := obj["id"].(string)
		node := &commentNode{obj: obj}
		if id != "" {
			byID[id] = node
		}
		roots = append(roots, node)
	}

	// Build tree: attach children to parents
	var topLevel []*commentNode
	for _, node := range roots {
		parentID, _ := node.obj["parent_id"].(string)
		if parentID != "" {
			if parent, ok := byID[parentID]; ok {
				parent.children = append(parent.children, node)
				continue
			}
		}
		topLevel = append(topLevel, node)
	}

	// Pick display columns: agent/username, body, created_at
	for _, node := range topLevel {
		printCommentNode(node, 0)
	}
}

func printCommentNode(node *commentNode, depth int) {
	prefix := ""
	if depth > 0 {
		prefix = strings.Repeat("  ", depth-1) + "  └ "
	}

	author := ""
	if a, ok := node.obj["agent"].(map[string]interface{}); ok {
		if u, ok := a["username"].(string); ok {
			author = u
		} else if n, ok := a["name"].(string); ok {
			author = n
		}
	}

	body := formatCell(node.obj["body"])
	ts := formatCell(node.obj["created_at"])

	if author != "" {
		fmt.Printf("%s[%s] %s  (%s)\n", prefix, author, body, ts)
	} else {
		fmt.Printf("%s%s  (%s)\n", prefix, body, ts)
	}

	for _, child := range node.children {
		printCommentNode(child, depth+1)
	}
}
