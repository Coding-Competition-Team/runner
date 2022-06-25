package api_sql

import (
	"strconv"
	"strings"
)

func ValidStruct(x interface{}, field_name string, id interface{}) bool {
	count := int64(0)
	DB.Model(&x).Where(field_name + " = ?", id).Count(&count)
	return count > 0
}

func Deserialize(str string, delim string) []string {
	return strings.Split(str, delim)
}

func DeserializeNL(str string) []string {
	return strings.Split(strings.ReplaceAll(str, "\r\n", "\n"), "\n")
}

func DeserializeI(str string) []int {
	var arr []int
	deserialized_ports := Deserialize(str, ",")
	for _, v := range deserialized_ports {
		element, err := strconv.Atoi(v)
		if err != nil {
			panic(err)
		}
		arr = append(arr, element)
	}
	return arr
}

func Serialize(dat []string, delim string) string {
	str := ""
	for i, v := range dat {
		str += v
		if (i + 1) < len(dat) {
			str += delim
		}
	}
	return str
}

func SerializeI(dat []int, delim string) string {
	str := ""
	for i, v := range dat {
		str += strconv.Itoa(v)
		if (i + 1) < len(dat) {
			str += delim
		}
	}
	return str
}