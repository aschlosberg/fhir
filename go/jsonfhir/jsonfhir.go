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

	"github.com/golang/protobuf/descriptor"
	"github.com/golang/protobuf/proto"
	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	pb "github.com/google/fhir/proto/stu3"
)

// NOTE: a descriptor.Message is simply a proto.Message with one additoinal
// method, allowing for extraction of the descriptor protos, and hence checking
// of extension options.

// An STU3Resource is any descriptor.Message that as an stu3.Id logical
// identifier.
type STU3Resource interface {
	descriptor.Message
	GetId() *pb.Id
}

// An stu3Extensible is any descriptor.Message that has stu3.Extensions.
type stu3Extensible interface {
	descriptor.Message
	GetExtension() []*pb.Extension
}

// An stu3Identifiable is any descriptor.Message that as an stu3.Id String
// identifier.
type stu3Identifiable interface {
	descriptor.Message
	GetId() *pb.String
}

// An stu3Element is an stu3Identifiable that also has a means of (un)marshaling
// itself to and from JSON. All of the stu3 primitives are stu3Elements.
type stu3Element interface {
	stu3Identifiable
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
// marshalValue, which may call marshalToTree, and is terminated by discovery of
// stu3Elements in the STU3Resource. The nodePath carries the path to the node
// in the tree that is currently being processed; therefore the original call
// should have nodePath = "/".
func marshalToTree(r STU3Resource, nodePath string) (_ marshalTree, retErr error) {
	defer func() {
		if retErr != nil {
			retErr = fmt.Errorf("[%s] %v", nodePath, retErr)
		}
	}()

	rVal := reflect.ValueOf(r)
	if r == nil || rVal.IsNil() {
		return nil, nil
	}

	// msg holds the protobuf message struct
	msg := rVal.Elem()
	if k := msg.Kind(); k != reflect.Struct {
		return nil, fmt.Errorf("cannot convert non-struct (%s) to tree", k)
	}

	fldDescs := make(map[string]*dpb.FieldDescriptorProto)
	_, md := descriptor.ForMessage(r)
	for _, f := range md.Field {
		fldDescs[f.GetName()] = f
	}

	tree := marshalTree{
		"resourceType": &pb.String{Value: msg.Type().Name()},
	}

	for i, n := 0, msg.NumField(); i < n; i++ {
		// val holds the actual value in the struct field whereas fld is a
		// reflect.StructField, describing the field. Reflection is confusing,
		// but at least it's fun :)
		val := msg.Field(i)
		fld := msg.Type().Field(i)

		if !val.CanInterface() ||
			(val.Kind() == reflect.Ptr && val.IsNil()) ||
			(val.Kind() == reflect.Slice && val.Len() == 0) {
			continue
		}

		lbl, ok := jsonName(fld.Tag)
		if !ok {
			continue
		}
		nodePath := fmt.Sprintf("%s%s/", nodePath, lbl)

		choice, err := extractIfChoiceType(val, fld, fldDescs)
		if err != nil {
			return nil, fmt.Errorf("detecting choice-type field: %v", err)
		}
		if choice.IsValid() {
			val = choice
			lbl = fmt.Sprintf("%s%s", lbl, val.Elem().Type().Name())
		}

		switch val.Kind() {
		case reflect.Ptr:
			mv, uscore, err := marshalValue(val, nodePath)
			if err != nil {
				return nil, err
			}
			tree[lbl] = mv
			if !uscore.empty() {
				tree[fmt.Sprintf("_%s", lbl)] = uscore
			}
		// A slice is treated in the exact same way as a pointer, but once for
		// each element. The slice of Underscore properties is added if any
		// element has a non-nil value.
		case reflect.Slice:
			n := val.Len()
			if n == 0 {
				break
			}
			all := make(nodeSlice, n)
			uscores := make(nodeSlice, n)
			var hasUnderscore bool
			for i := 0; i < n; i++ {
				mv, uscore, err := marshalValue(val, fmt.Sprintf("%s[%d]", nodePath, i))
				if err != nil {
					return nil, err
				}
				all[i] = mv
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

// stu3Underscore represents a JSON property prepended with an underscore, as
// described by https://www.hl7.org/fhir/json.html#primitive.
type stu3Underscore struct {
	ID        *pb.String      `json:"id,omitempty"`
	Extension []*pb.Extension `json:"extension,omitempty"`

	marshalNode
}

// empty returns false if either u.ID contains a non-empty string, or there is
// at least one non-nil Extension.
func (u *stu3Underscore) empty() bool {
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

func underscore(msg proto.Message) *stu3Underscore {
	u := new(stu3Underscore)
	if i, ok := msg.(stu3Identifiable); ok {
		u.ID = i.GetId()
	}
	if e, ok := msg.(stu3Extensible); ok {
		u.Extension = e.GetExtension()
	}
	return u
}

// marshalValue is the field-level counterpart for marshalToTree, producing a
// single node from a struct field's value. If the field contains an
// stu3Element, itself a marshalNode, it is simply returned, otherwise the
// field's value is recursed back into marshalToTree.
func marshalValue(val reflect.Value, nodePath string) (marshalNode, *stu3Underscore, error) {
	ifc := val.Interface()

	msg, ok := ifc.(descriptor.Message)
	if !ok {
		return nil, nil, fmt.Errorf("cannot marshal non-descriptor.Message %T", ifc)
	}
	uscore := underscore(msg)

	if el, ok := ifc.(stu3Element); ok {
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

func extractIfChoiceType(val reflect.Value, fld reflect.StructField, descs map[string]*dpb.FieldDescriptorProto) (reflect.Value, error) {
	name, ok := protoName(fld.Tag)
	if !ok {
		return reflect.Value{}, nil
	}
	desc, ok := descs[name]
	if !ok {
		return reflect.Value{}, fmt.Errorf("no FieldDescriptor proto available for %s", name)
	}
	if !proto.HasExtension(desc.Options, pb.E_IsChoiceType) {
		return reflect.Value{}, nil
	}

	// All the choice types have a oneof field of their same name, so we simply
	// call GetX on a field named X.
	fn := fmt.Sprintf("Get%s", fld.Name)
	get := val.MethodByName(fn)
	if !get.IsValid() {
		return reflect.Value{}, fmt.Errorf("field %s has no %s() method", fld.Name, fn)
	}
	if t := get.Type(); t.Kind() != reflect.Func || t.NumIn() != 0 || t.NumOut() != 1 {
		return reflect.Value{}, fmt.Errorf("%s.%s is not proto getter; must be function with nil input, returning 1 value", fld.Name, fn)
	}
	choice := get.Call(nil)
	if n := len(choice); n != 1 {
		return reflect.Value{}, fmt.Errorf("%s.%s() returned %d values; expecting 1", fld.Name, fn, n)
	}

	// As it's a oneof, the value will be an interface and the concrete type
	// will be a pointer to a struct with exactly one field.
	if t := choice[0].Type(); t.Kind() != reflect.Interface {
		return reflect.Value{}, fmt.Errorf("%s.%s() must return interface; got %s", fld.Name, fn, t.Kind())
	}
	one := reflect.ValueOf(choice[0].Interface()).Elem()
	if t := one.Type(); t.Kind() != reflect.Struct || t.NumField() != 1 {
		return reflect.Value{}, fmt.Errorf("%s.%s() must have concrete-type *struct with one field; got Type %s", fld.Name, fn, t)
	}
	oneVal := one.Field(0)
	if oneVal.Kind() != reflect.Ptr || oneVal.IsNil() {
		return reflect.Value{}, fmt.Errorf("expecting non-nil pointer in single field of struct returned by %s.%s()", fld.Name, fn)
	}

	return oneVal, nil
}

func protoName(t reflect.StructTag) (string, bool) {
	tag, ok := t.Lookup("protobuf")
	if !ok {
		return "", false
	}
	parts := strings.Split(tag, ",")
	for _, p := range parts {
		const pref = "name="
		if strings.HasPrefix(p, pref) {
			return strings.TrimPrefix(p, pref), true
		}
	}
	return "", false
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
