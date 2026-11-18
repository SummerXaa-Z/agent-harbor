package main

import (
	"os"
	"reflect"
)

func wrapLinkCount(info os.FileInfo) uint64 {
	value := reflect.ValueOf(info.Sys())
	if value.Kind() == reflect.Pointer {
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return 0
	}
	field := value.FieldByName("Nlink")
	if field.IsValid() && field.CanUint() {
		return field.Uint()
	}
	return 0
}
