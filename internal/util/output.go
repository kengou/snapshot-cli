package util

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"

	"github.com/olekukonko/tablewriter"
)

const (
	OutputTable = "table"
	OutputJSON  = "json"
)

// WriteJSON marshals d to JSON and writes it to stdout followed by a newline.
func WriteJSON(d any) error {
	b, err := json.Marshal(d)
	if err != nil {
		return fmt.Errorf("%s: %w", "could not marshal JSON", err)
	}

	if _, err = fmt.Fprintln(os.Stdout, string(b)); err != nil {
		return fmt.Errorf("%s: %w", "could not write JSON data", err)
	}

	return nil
}

// WriteAsTable renders input as an ASCII table to stdout using the provided column headers.
// If input is a slice, each element is added as a row; otherwise input is rendered as a single row.
func WriteAsTable(input, header interface{}) (err error) {
	t := tablewriter.NewWriter(os.Stdout)
	t.Header(header)
	if isSlice(input) {
		err = t.Bulk(input)
	} else {
		err = t.Append(input)
	}
	if err != nil {
		return err
	}
	err = t.Render()
	if err != nil {
		return err
	}
	return nil
}

// isSlice reports whether i is a slice type. Returns false for nil.
func isSlice(i any) bool {
	if i == nil {
		return false
	}
	return reflect.TypeOf(i).Kind() == reflect.Slice
}
