// Copyright (c) 2018-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

/*
Package validator provides common validation helpers commonly used
in operations tools.  Additionally structures can be marked up with
tags indicating the validation of individual keys and the entire struct
can be validated in one go
*/
package validator

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/choria-io/go-choria/validator/duration"
	"github.com/choria-io/go-choria/validator/enum"
	"github.com/choria-io/go-choria/validator/ipaddress"
	"github.com/choria-io/go-choria/validator/ipv4"
	"github.com/choria-io/go-choria/validator/ipv6"
	"github.com/choria-io/go-choria/validator/maxlength"
	"github.com/choria-io/go-choria/validator/regex"
	"github.com/choria-io/go-choria/validator/shellsafe"
)

// ValidateStruct validates all keys in a struct using their validate tag
func ValidateStruct(target interface{}) (bool, error) {
	val := reflect.ValueOf(target)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	return validateStructValue(val)
}

// ValidateStructField validates one field in a struct
func ValidateStructField(target interface{}, field string) (bool, error) {
	val := reflect.ValueOf(target)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	valueField := val.FieldByName(field)
	typeField, ok := val.Type().FieldByName(field)
	if !ok {
		return false, fmt.Errorf("unknown field %s", field)
	}

	validation := strings.TrimSpace(typeField.Tag.Get("validate"))

	err := validateStructField(valueField, typeField, validation)
	if err != nil {
		return false, err
	}

	return true, nil
}

func validateStructValue(val reflect.Value) (bool, error) {
	for i := 0; i < val.NumField(); i++ {
		valueField := val.Field(i)
		typeField := val.Type().Field(i)
		validation := strings.TrimSpace(typeField.Tag.Get("validate"))

		err := validateStructField(valueField, typeField, validation)
		if err != nil {
			return false, err
		}
	}

	return true, nil
}

func validateStructField(valueField reflect.Value, typeField reflect.StructField, validation string) error {
	if valueField.Kind() == reflect.Struct {
		ok, err := validateStructValue(valueField)
		if !ok {
			return err
		}
	}

	if validation == "" {
		return nil
	}

	if validation == "shellsafe" {
		if ok, err := shellsafe.ValidateStructField(valueField, validation); !ok {
			return fmt.Errorf("%s shellsafe validation failed: %s", typeField.Name, err)
		}

	} else if validation == "ipv4" {
		if ok, err := ipv4.ValidateStructField(valueField, validation); !ok {
			return fmt.Errorf("%s IPv4 validation failed: %s", typeField.Name, err)
		}

	} else if validation == "ipv6" {
		if ok, err := ipv6.ValidateStructField(valueField, validation); !ok {
			return fmt.Errorf("%s IPv6 validation failed: %s", typeField.Name, err)
		}

	} else if validation == "ipaddress" {
		if ok, err := ipaddress.ValidateStructField(valueField, validation); !ok {
			return fmt.Errorf("%s IP address validation failed: %s", typeField.Name, err)
		}

	} else if strings.HasPrefix(validation, "regex") {
		if ok, err := regex.ValidateStructField(valueField, validation); !ok {
			return fmt.Errorf("%s regular expression validation failed: %s", typeField.Name, err)
		}

	} else if strings.HasPrefix(validation, "maxlength") {
		if ok, err := maxlength.ValidateStructField(valueField, validation); !ok {
			return fmt.Errorf("%s maxlength validation failed: %s", typeField.Name, err)
		}

	} else if strings.HasPrefix(validation, "enum") {
		if ok, err := enum.ValidateStructField(valueField, validation); !ok {
			return fmt.Errorf("%s enum validation failed: %s", typeField.Name, err)
		}

	} else if strings.HasPrefix(validation, "duration") {
		if ok, err := duration.ValidateStructField(valueField, validation); !ok {
			return fmt.Errorf("%s duration validation failed: %s", typeField.Name, err)
		}
	}

	return nil
}
