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

func isSlice(i interface{}) bool {
	return reflect.TypeOf(i).Kind() == reflect.Slice
}
