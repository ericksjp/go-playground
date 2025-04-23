package validator

import (
	"regexp"
	"slices"
)

var (
	EmailRX = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
)

// type that will hold a map containing all the errors
// {"email": "email is not valid", "id": "id is not uuid"} ...
type Validator struct {
	Errors map[string]string
}

func New() *Validator {
	validator := &Validator{
		Errors: make(map[string]string),
	}
	return validator
}

// return true if theres no entries in the validator Errors map
func (v Validator) Valid() bool  {
	return len(v.Errors) < 1
}

// add en error to the map
func (v *Validator) AddError(key, message string) {
	// keep the first error assigned
	_, ok := v.Errors[key];
	if !ok {
		v.Errors[key] = message
	}
}

// adds an error message to the map if the condition is true
func (v *Validator) Check(condition bool, key, message string)  {
	if condition {
		v.AddError(key, message)
	}
}

// returns true if the value is in the list
func In[T comparable](value T, list []T) bool {
	return slices.Contains(list, value)
}

// return true if a string value matches the regex expression
func Matches(value string, rx *regexp.Regexp) bool {
	return rx.MatchString(value)
}

// return true if all string values in a slice are unique
func Unique[T comparable](list []T) bool {
	set := make(map[T]struct{}, len(list));

	for _, v := range list {
		_, ok := set[v];
		if ok {
			return false
		}
		set[v] = struct{}{}
	}

	return true
}
