package data

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrInvalidRuntimeFormat = errors.New("invalid runtime format")

type Runtime int32

// this method satisfies the Marshaller Interface and will convert any Runtime type to
// 10 -> "10 mins" when a call to json.Marshal is called
func (r Runtime) MarshalJSON() ([]byte, error) {
	jsonVal := strconv.Quote(fmt.Sprintf("%d mins", r))
	return []byte(jsonVal), nil
}

// this method satisfies the Unmarshaller Interface and will convert any JSON value
// "10 mins" or "10" -> 10 when a call to json.Unmarshal is called
func (r *Runtime) UnmarshalJSON(data []byte) error {
    unquoted, err := strconv.Unquote(string(data))
    if err != nil {
        return ErrInvalidRuntimeFormat
    }

    parts := strings.Fields(unquoted)
    if len(parts) == 0 || len(parts) > 2 {
        return ErrInvalidRuntimeFormat
    }

    duration, err := strconv.ParseInt(parts[0], 10, 32)
    if err != nil {
        return ErrInvalidRuntimeFormat
    }

    // validate the optional second part
    if len(parts) == 2 && parts[1] != "mins" {
        return ErrInvalidRuntimeFormat
    }

    // assign the parsed value to the runtime
    *r = Runtime(duration)
    return nil
}
