package csv

import "strings"

func CreateCSVFile(data [][]string) ([]byte, error) {
	sb := strings.Builder{}
	for _, row := range data {
		for _, cell := range row {
			sb.WriteString(cell)
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}
	return []byte(sb.String()), nil
}
