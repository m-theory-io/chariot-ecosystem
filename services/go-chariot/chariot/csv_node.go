package chariot

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// CSVNode implements TreeNode for CSV data (no interface constraints)
type CSVNode struct {
	TreeNodeImpl
	// No more: cachedRows, reader, currentRow, etc.
	// All data stored in attributes/metadata for clean architecture
}

func NewCSVNode(name string) *CSVNode {
	node := &CSVNode{}
	node.TreeNodeImpl = *NewTreeNode(name)

	// Initialize CSV metadata with defaults
	node.SetMeta("delimiter", ",")
	node.SetMeta("hasHeaders", true)
	node.SetMeta("encoding", "UTF-8")
	node.SetMeta("rowCount", 0)
	node.SetMeta("columnCount", 0)

	return node
}

// ETL-optimized methods (no interface constraints)
func (n *CSVNode) LoadFromFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Store file metadata
	n.SetMeta("sourceFile", path)
	n.SetMeta("loadedAt", time.Now().Unix())

	// Get file size for ETL planning
	if stat, err := file.Stat(); err == nil {
		n.SetMeta("fileSize", stat.Size())
	}

	return n.LoadFromReader(file)
}

func (n *CSVNode) LoadFromReader(r io.Reader) error {
	delimiter := ","
	if delim, exists := n.GetMeta("delimiter"); exists {
		delimiter = delim.(string)
	}

	hasHeaders := true
	if headers, exists := n.GetMeta("hasHeaders"); exists {
		hasHeaders = headers.(bool)
	}

	reader := csv.NewReader(r)
	reader.Comma = rune(delimiter[0])

	var headers []string

	// Read headers if present
	if hasHeaders {
		headerRow, err := reader.Read()
		if err != nil {
			return err
		}
		headers = headerRow
		n.SetMeta("headers", convertFromNativeValue(headers))
		n.SetMeta("columnCount", len(headers))
	}

	// For small files, cache all rows
	// For large files, this method should be avoided in favor of streaming
	rows, err := reader.ReadAll()
	if err != nil {
		return err
	}

	// Store rows as attribute (use with caution for large files)
	n.SetAttribute("rows", convertFromNativeValue(rows))
	n.SetMeta("rowCount", len(rows))

	return nil
}

// Streaming method for ETL (no memory limitations)
func (n *CSVNode) StreamProcess(chunkSize int, processor func([][]string) error) error {
	sourceFile, exists := n.GetMeta("sourceFile")
	if !exists {
		return fmt.Errorf("no source file set - use LoadFromFile first")
	}

	file, err := os.Open(sourceFile.(string))
	if err != nil {
		return err
	}
	defer file.Close()

	delimiter := ","
	if delim, exists := n.GetMeta("delimiter"); exists {
		delimiter = delim.(string)
	}

	hasHeaders := true
	if headers, exists := n.GetMeta("hasHeaders"); exists {
		hasHeaders = headers.(bool)
	}

	reader := csv.NewReader(file)
	reader.Comma = rune(delimiter[0])

	// Skip headers if present and store them
	if hasHeaders {
		headerRow, err := reader.Read()
		if err != nil {
			return err
		}
		n.SetAttribute("headers", convertFromNativeValue(headerRow))
		n.SetMeta("columnCount", len(headerRow))
	}

	batch := make([][]string, 0, chunkSize)
	rowsProcessed := 0
	batchCount := 0

	for {
		row, err := reader.Read()
		if err == io.EOF {
			// Process final batch
			if len(batch) > 0 {
				if err := processor(batch); err != nil {
					return err
				}
				batchCount++
			}
			break
		}
		if err != nil {
			return err
		}

		batch = append(batch, row)
		rowsProcessed++

		// Process full batch
		if len(batch) == chunkSize {
			if err := processor(batch); err != nil {
				return err
			}

			// Update metadata for progress tracking
			batchCount++
			n.SetMeta("rowsProcessed", rowsProcessed)
			n.SetMeta("batchesProcessed", batchCount)
			n.SetMeta("lastBatchTime", time.Now().Unix())

			// Reset batch
			batch = batch[:0]
		}
	}

	// Final metadata update
	n.SetMeta("rowCount", rowsProcessed)
	n.SetMeta("batchesProcessed", batchCount)
	n.SetMeta("completedTime", time.Now().Unix())

	return nil
}

// Metadata-based accessors (memory efficient)
func (n *CSVNode) GetHeaders() []string {
	if headersAttr, exists := n.GetMeta("headers"); exists {
		if h, ok := convertValueToNative(headersAttr).([]interface{}); ok {
			headers := make([]string, len(h))
			for i, v := range h {
				headers[i] = v.(string)
			}
			return headers
		}
	}
	return []string{}
}

func (n *CSVNode) GetRowCount() int {
	if count, exists := n.GetMeta("rowCount"); exists {
		if c, ok := count.(float64); ok {
			return int(c)
		}
	}
	return 0
}

func (n *CSVNode) GetColumnCount() int {
	if count, exists := n.GetMeta("columnCount"); exists {
		if c, ok := count.(float64); ok {
			return int(c)
		}
	}
	return 0
}

// Safe row access (for small files or cached data)
func (n *CSVNode) GetRow(index int) (map[string]string, error) {
	if index < 0 || index >= n.GetRowCount() {
		return nil, fmt.Errorf("row %d out of range", index)
	}

	headers := n.GetHeaders()

	if rowsAttr, exists := n.GetAttribute("rows"); exists {
		if rows, ok := convertValueToNative(rowsAttr).([]interface{}); ok {
			if index < len(rows) {
				if row, ok := rows[index].([]interface{}); ok {
					result := make(map[string]string)
					for i, value := range row {
						if i < len(headers) {
							result[headers[i]] = value.(string)
						} else {
							result[fmt.Sprintf("col_%d", i)] = value.(string)
						}
					}
					return result, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("row data not available - file may be too large for in-memory access")
}

// Safe cell access
func (n *CSVNode) GetCell(row int, col interface{}) (string, error) {
	rowData, err := n.GetRow(row)
	if err != nil {
		return "", err
	}

	switch v := col.(type) {
	case int:
		headers := n.GetHeaders()
		if v < 0 || v >= len(headers) {
			return "", fmt.Errorf("column %d out of range", v)
		}
		return rowData[headers[v]], nil

	case string:
		if value, exists := rowData[v]; exists {
			return value, nil
		}
		return "", fmt.Errorf("column '%s' not found", v)

	default:
		return "", fmt.Errorf("column must be string or int")
	}
}

// Compatibility methods (with safety checks)
func (n *CSVNode) GetRows() ([][]string, error) {
	rowCount := n.GetRowCount()
	if rowCount > 10000 {
		return nil, fmt.Errorf("file too large (%d rows) for GetRows(), use StreamProcess instead", rowCount)
	}

	if rowsAttr, exists := n.GetAttribute("rows"); exists {
		if rows, ok := convertValueToNative(rowsAttr).([]interface{}); ok {
			result := make([][]string, len(rows))
			for i, row := range rows {
				if rowSlice, ok := row.([]interface{}); ok {
					result[i] = make([]string, len(rowSlice))
					for j, cell := range rowSlice {
						result[i][j] = cell.(string)
					}
				}
			}
			return result, nil
		}
	}

	return nil, fmt.Errorf("row data not available")
}

func (n *CSVNode) ToCSV() (string, error) {
	rows, err := n.GetRows()
	if err != nil {
		return "", err
	}

	headers := n.GetHeaders()

	var result strings.Builder

	// Write headers if present
	if len(headers) > 0 {
		for i, header := range headers {
			if i > 0 {
				result.WriteString(",")
			}
			result.WriteString(header)
		}
		result.WriteString("\n")
	}

	// Write data rows
	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				result.WriteString(",")
			}
			result.WriteString(cell)
		}
		result.WriteString("\n")
	}

	return result.String(), nil
}

// Transform integration for ETL
func (n *CSVNode) ApplyTransform(transform *Transform) error {
	return n.StreamProcess(1000, func(batch [][]string) error {
		headers := n.GetHeaders()

		for _, csvRow := range batch {
			// Convert CSV row to map
			rowMap := make(map[string]string)
			for j, value := range csvRow {
				if j < len(headers) {
					rowMap[headers[j]] = value
				}
			}

			// Apply transformation
			_, err := transform.ApplyToRow(nil, rowMap) // Add runtime parameter as needed
			if err != nil {
				return err
			}
		}

		return nil
	})
}
