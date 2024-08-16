package util

import (
	"sort"
	"strconv"
	"strings"
)

func FileListNameFilter(FileList []string, filter string) []string {
	result := []string{}
	for _, v := range FileList {
		if strings.Contains(v, filter) {
			result = append(result, v)
		}
	}
	return result
}

func Sortdata(data [][]string) {
	sort.Slice(data, func(i, j int) bool {
		num1, _ := strconv.Atoi(data[i][1])
		num2, _ := strconv.Atoi(data[j][1])
		return num1 > num2
	})
}
