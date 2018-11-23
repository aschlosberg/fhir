/*
 * Copyright 2018 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package jsonfhir provides marshaling between JSON and FHIR protocol buffers.
package jsonfhir

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/golang/protobuf/proto"
	pb "github.com/google/fhir/proto/stu3"
)

// An STU3Extensible is any proto.Message that has stu3.Extensions.
type STU3Extensible interface {
	proto.Message
	GetExtension() []*pb.Extension
}

// An STU3Resource is any proto.Message that as an stu3.Id logical identifier.
type STU3Resource interface {
	proto.Message
	GetId() *pb.Id
}

// An STU3Element is any proto.Message that has an stu3.String ID. It must also
// have a means of (un)marshaling itself to and from JSON. All of the STU3
// primitives are STU3Elements.
type STU3Element interface {
	proto.Message
	GetId() *pb.String
	json.Marshaler
	json.Unmarshaler
}

// MarshalSTU3JSON IS INCOMPLETE AND SHOULD NOT BE USED YET. It returns the
// STU3Resource in JSON format.
func MarshalSTU3JSON(msg STU3Resource) ([]byte, error) {
	m, err := marshalToMap(msg, "/")
	if err != nil {
		return nil, err
	}
	return json.Marshal(m)
}

// marshalToMap converts the resource into a map of strings to empty interfaces,
// allowing for eventual marshaling with encoding/json.MarshalJSON(). It is
// called recursively, and the keyPath carries the path to the key on which the
// error occurred; the original call should have keyPath = "/".
func marshalToMap(msg STU3Resource, keyPath string) (_ map[string]interface{}, retErr error) {
	defer func() {
		if retErr != nil {
			retErr = fmt.Errorf("[%s] %v", keyPath, retErr)
		}
	}()

	mVal := reflect.ValueOf(msg)
	if msg == nil || mVal.IsNil() {
		return nil, nil
	}

	val := mVal.Elem()
	if k := val.Kind(); k != reflect.Struct {
		return nil, fmt.Errorf("cannot convert non-struct (%s) to map", k)
	}

	mapped := map[string]interface{}{
		`resourceType`: `x`,
	}

	for i, n := 0, val.NumField(); i < n; i++ {
		fld := val.Field(i)
		if !fld.CanInterface() || (fld.Kind() == reflect.Ptr && fld.IsNil()) {
			continue
		}

		key, ok := jsonName(val.Type().Field(i).Tag)
		if !ok {
			continue
		}
		keyPath = fmt.Sprintf("%s%s/", keyPath, key)

		switch fld.Kind() {
		case reflect.Slice:
			n := fld.Len()
			if n == 0 {
				break
			}
			all := make([]interface{}, n)
			for i := 0; i < n; i++ {
				mf, _, err := marshalField(fld, fmt.Sprintf("%s[%d]", keyPath, i))
				if err != nil {
					return nil, err
				}
				all[i] = mf
			}
			mapped[key] = all
		case reflect.Ptr:
			mf, uscore, err := marshalField(fld, keyPath)
			if err != nil {
				return nil, err
			}
			mapped[key] = mf
			if !uscore.empty() {
				mapped[fmt.Sprintf("_%s", key)] = uscore
			}
		}
	}

	return mapped, nil
}

// STU3Underscore represents a JSON property prepended with an underscore, as
// described by https://www.hl7.org/fhir/json.html#primitive.
type STU3Underscore struct {
	ID        *pb.String      `json:"id,omitempty"`
	Extension []*pb.Extension `json:"extension,omitempty"`
}

func (u *STU3Underscore) empty() bool {
	if u == nil {
		return true
	}
	if u.ID != nil {
		return false
	}
	for _, e := range u.Extension {
		if e != nil {
			return false
		}
	}
	return true
}

// marshalField is the field-level counterpart for marshalToMap. The first
// returned interface{} is fld.Interface() if the value is an STU3Element,
// otherwise the field is sent back to marshalToMap if is is an STU3Resrouce.
func marshalField(fld reflect.Value, keyPath string) (interface{}, *STU3Underscore, error) {
	ifc := fld.Interface()

	uscore := new(STU3Underscore)

	if el, ok := ifc.(STU3Element); ok {
		uscore.ID = el.GetId()
		// TODO(arrans) just return el directly once they're all implemented.
		buf, err := el.MarshalJSON()
		if err != nil && !strings.Contains(err.Error(), "unimplemented") {
			return nil, nil, err
		}
		return json.RawMessage(buf), uscore, nil
	}

	if res, ok := ifc.(STU3Resource); ok {
		mm, err := marshalToMap(res, keyPath)
		if err != nil {
			return nil, nil, err
		}
		return mm, uscore, nil
	}

	return nil, nil, fmt.Errorf("unsupported field type %T", ifc)
}

func jsonName(t reflect.StructTag) (string, bool) {
	tag, ok := t.Lookup("json")
	if !ok {
		return "", false
	}
	parts := strings.Split(tag, ",")
	if len(parts) == 0 || parts[0] == "" {
		return "", false
	}
	return parts[0], true
}
