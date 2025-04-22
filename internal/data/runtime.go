package data

import (
	"fmt"
	"strconv"
)

type Runtime int32

// this method satisfies the Marshaller Interface and will convert any Runtime type to
// 10 -> "10 mins" when a call to json.Marshal is called
func (r Runtime) MarshalJSON() ([]byte, error) {
	jsonVal := strconv.Quote(fmt.Sprintf("%d mins", r))
	return []byte(jsonVal), nil
}
