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

	// JSONFHIRMarshaler is a no-op function, required internally by this
	// package simply to improve type safety at compile time, instead of using
	// the empty interface{}.
	JSONFHIRMarshaler()
}

// MarshalSTU3JSON IS INCOMPLETE AND SHOULD NOT BE USED YET. It returns the
// STU3Resource in JSON format.
func MarshalSTU3JSON(msg STU3Resource) ([]byte, error) {
	m, err := marshalToTree(msg, "/")
	if err != nil {
		return nil, err
	}
	return json.Marshal(m)
}

// A marshalTree is a recursive structure mapping strings to marshalNodes, which
// are either other marshalTrees, or concrete types that can be marshalled to
// FHIR-compliant JSON. It is used instead of a map[string]interface{} simply to
// improve type safety, but is still passed to json.Marshal() to be handled in
// the same manner.
type marshalTree map[string]marshalNode

func (marshalTree) JSONFHIRMarshaler() {}

// A marshalNode indicates that an arbitrary type can act as a node of a
// marshalTree.
type marshalNode interface {
	JSONFHIRMarshaler()
}

type nodeSlice []marshalNode

func (nodeSlice) JSONFHIRMarshaler() {}

// marshalToTree converts the resource into marshalTree, allowing for eventual
// marshaling with encoding/json.MarshalJSON(). It acts recursively via calls to
// marshalField, which may call marshalToTree, and is terminated by discovery of
// STU3Elements in the STU3Resource. The nodePath carries the path to the node
// in the tree that is currently being processed; therefore the original call
// should have nodePath = "/".
func marshalToTree(msg STU3Resource, nodePath string) (_ marshalTree, retErr error) {
	defer func() {
		if retErr != nil {
			retErr = fmt.Errorf("[%s] %v", nodePath, retErr)
		}
	}()

	mVal := reflect.ValueOf(msg)
	if msg == nil || mVal.IsNil() {
		return nil, nil
	}

	val := mVal.Elem()
	if k := val.Kind(); k != reflect.Struct {
		return nil, fmt.Errorf("cannot convert non-struct (%s) to tree", k)
	}

	tree := marshalTree{
		"resourceType": &pb.String{Value: val.Type().Name()},
	}

	for i, n := 0, val.NumField(); i < n; i++ {
		fld := val.Field(i)
		if !fld.CanInterface() || (fld.Kind() == reflect.Ptr && fld.IsNil()) {
			continue
		}

		lbl, ok := jsonName(val.Type().Field(i).Tag)
		if !ok {
			continue
		}
		nodePath := fmt.Sprintf("%s%s/", nodePath, lbl)

		switch fld.Kind() {
		case reflect.Ptr:
			mf, uscore, err := marshalField(fld, nodePath)
			if err != nil {
				return nil, err
			}
			tree[lbl] = mf
			if !uscore.empty() {
				tree[fmt.Sprintf("_%s", lbl)] = uscore
			}
		// A slice is treated in the exact same way as a pointer, but once for
		// each element. The slice of Underscore properties is added if any
		// element has a non-nil value.
		case reflect.Slice:
			n := fld.Len()
			if n == 0 {
				break
			}
			all := make(nodeSlice, n)
			uscores := make(nodeSlice, n)
			var hasUnderscore bool
			for i := 0; i < n; i++ {
				mf, uscore, err := marshalField(fld, fmt.Sprintf("%s[%d]", nodePath, i))
				if err != nil {
					return nil, err
				}
				all[i] = mf
				if !uscore.empty() {
					uscores[i] = uscore
					hasUnderscore = true
				}
			}
			tree[lbl] = all
			if hasUnderscore {
				tree[fmt.Sprintf("_%s", lbl)] = uscores
			}
		}
	}

	return tree, nil
}

// STU3Underscore represents a JSON property prepended with an underscore, as
// described by https://www.hl7.org/fhir/json.html#primitive.
type STU3Underscore struct {
	ID        *pb.String      `json:"id,omitempty"`
	Extension []*pb.Extension `json:"extension,omitempty"`

	marshalNode
}

// empty returns false if either u.ID contains a non-empty string, or there is
// at least one non-nil Extension.
func (u *STU3Underscore) empty() bool {
	if u == nil {
		return true
	}
	if u.ID != nil && u.ID.Value != "" {
		return false
	}
	for _, e := range u.Extension {
		if e != nil {
			return false
		}
	}
	return true
}

// marshalField is the field-level counterpart for marshalToTree, producing a
// single node from a struct field. If the field contains an STU3Element, itself
// a marshalNode, it is simply returned, otherwise the field's value is recursed
// back into marshalToTree.
func marshalField(fld reflect.Value, nodePath string) (marshalNode, *STU3Underscore, error) {
	ifc := fld.Interface()

	uscore := new(STU3Underscore)
	if ext, ok := ifc.(STU3Extensible); ok {
		uscore.Extension = ext.GetExtension()
	}

	if el, ok := ifc.(STU3Element); ok {
		uscore.ID = el.GetId()
		// TODO(arrans) remove this once MarshalJSON is implemented on all
		// Elements.
		if _, err := el.MarshalJSON(); err != nil && strings.Contains(err.Error(), "unimplemented") {
			return nil, nil, err
		}
		return el, uscore, nil
	}

	if res, ok := ifc.(STU3Resource); ok {
		mt, err := marshalToTree(res, nodePath)
		if err != nil {
			return nil, nil, err
		}
		return mt, uscore, nil
	}

	return nil, nil, fmt.Errorf("unsupported field type %T", ifc)
}

func jsonName(t reflect.StructTag) (string, bool) {
	tag, ok := t.Lookup("json")
	if !ok {
		return "", false
	}
	parts := strings.Split(tag, ",")
	if len(parts) == 0 || parts[0] == "" || parts[0] == "-" {
		return "", false
	}
	return snakeToCamel(parts[0]), true
}

func snakeToCamel(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	b.Grow(len(s))

	var last int
	for i, n := 0, len(s); i < n; i++ {
		if s[i] == '_' {
			b.WriteString(s[last:i])
			i++
			if i < n && s[i] >= 'a' && s[i] <= 'z' {
				b.WriteByte(s[i] - 'a' + 'A')
				i++
			}
			last = i
		}
	}
	if last < len(s) {
		b.WriteString(s[last:])
	}
	return b.String()
}
