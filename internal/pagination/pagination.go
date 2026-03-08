package pagination

import (
	"encoding/json"
	"fmt"

	"api-cli/internal/client"
)

// Style defines the pagination strategy.
type Style string

const (
	Cursor Style = "cursor"
	Page   Style = "page"
	Offset Style = "offset"
)

// Options configures pagination for a specific API endpoint.
type Options struct {
	Style Style

	// Cursor-based: the JSON field name containing the next cursor/token.
	CursorField string
	// Cursor-based: the query parameter name to send the cursor value.
	CursorParam string

	// Page-number: the query parameter name for the page number.
	PageParam string
	// Page-number: the starting page (usually 1).
	StartPage int

	// Offset-based: the query parameter name for the offset.
	OffsetParam string
	// Offset-based: number of items per page.
	PageSize int

	// DataField is the JSON field containing the array of results.
	// If empty, the response is expected to be an array.
	DataField string

	// MaxPages limits the number of pages to fetch (0 = no limit).
	MaxPages int
}

// FetchAll retrieves all pages and returns the combined results.
func FetchAll(c *client.Client, method, path string, pathParams, queryParams map[string]string, opts Options) ([]interface{}, error) {
	if queryParams == nil {
		queryParams = make(map[string]string)
	}

	var all []interface{}
	pageCount := 0

	switch opts.Style {
	case Cursor:
		cursor := ""
		for {
			if cursor != "" {
				queryParams[opts.CursorParam] = cursor
			}
			data, items, err := fetchPage(c, method, path, pathParams, queryParams, opts.DataField)
			if err != nil {
				return nil, err
			}
			all = append(all, items...)
			pageCount++

			if opts.MaxPages > 0 && pageCount >= opts.MaxPages {
				break
			}

			next, _ := getNestedString(data, opts.CursorField)
			if next == "" {
				break
			}
			cursor = next
		}

	case Page:
		page := opts.StartPage
		if page == 0 {
			page = 1
		}
		for {
			queryParams[opts.PageParam] = fmt.Sprintf("%d", page)
			_, items, err := fetchPage(c, method, path, pathParams, queryParams, opts.DataField)
			if err != nil {
				return nil, err
			}
			if len(items) == 0 {
				break
			}
			all = append(all, items...)
			pageCount++

			if opts.MaxPages > 0 && pageCount >= opts.MaxPages {
				break
			}
			page++
		}

	case Offset:
		offset := 0
		for {
			queryParams[opts.OffsetParam] = fmt.Sprintf("%d", offset)
			_, items, err := fetchPage(c, method, path, pathParams, queryParams, opts.DataField)
			if err != nil {
				return nil, err
			}
			if len(items) == 0 {
				break
			}
			all = append(all, items...)
			pageCount++

			if opts.MaxPages > 0 && pageCount >= opts.MaxPages {
				break
			}
			offset += opts.PageSize
		}
	}

	return all, nil
}

func fetchPage(c *client.Client, method, path string, pathParams, queryParams map[string]string, dataField string) (map[string]interface{}, []interface{}, error) {
	body, err := c.Request(method, path, pathParams, queryParams, nil, "")
	if err != nil {
		return nil, nil, err
	}

	// Try to parse as object with data field
	if dataField != "" {
		var obj map[string]interface{}
		if err := json.Unmarshal(body, &obj); err != nil {
			return nil, nil, fmt.Errorf("parsing response: %w", err)
		}
		items, ok := obj[dataField].([]interface{})
		if !ok {
			return obj, nil, nil
		}
		return obj, items, nil
	}

	// No data field — expect a direct array
	var items []interface{}
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, nil, fmt.Errorf("parsing response as array: %w", err)
	}
	return nil, items, nil
}

func getNestedString(data map[string]interface{}, field string) (string, bool) {
	if data == nil {
		return "", false
	}
	v, ok := data[field]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}
