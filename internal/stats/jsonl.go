package stats

import (
	"encoding/json"
	"os"
)

// writeJSONL marshals v as JSON and writes it as a single line to f.
func writeJSONL(f *os.File, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = f.Write(data)
	return err
}
