package api_sql

import (
	"strconv"
	"strings"

	"runner/internal/creds"
)

func Deserialize(str string, delim string) []string {
	return strings.Split(str, delim)
}

func DeserializeNL(str string) []string {
	return strings.Split(strings.ReplaceAll(str, "\r\n", "\n"), "\n")
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

func GetSqlDataSource() string {
	return creds.MySQLUsername+":"+creds.MySQLPassword+"@tcp("+creds.MySQLIP+")/runner_db"
}