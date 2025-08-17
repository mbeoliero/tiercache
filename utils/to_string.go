package utils

import (
	"encoding/json"
	"fmt"
	"strconv"
)

func ToString(value any) string {
	switch v := value.(type) {
	case fmt.Stringer:
		return v.String()
	case string:
		return v
	case int:
		return strconv.FormatInt(int64(v), 10)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case float32:
		return strconv.FormatFloat(float64(v), 'g', -1, 64)
	case float64:
		return strconv.FormatFloat(v, 'g', -1, 64)
	case []byte:
		return string(v)
	case nil:
		return ""
	case error:
		return v.Error()
	case bool:
		return strconv.FormatBool(v)
	default:
		bs, _ := json.Marshal(v)
		return string(bs)
	}
}
