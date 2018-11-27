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

// An STU3Resource is any descriptor.Message that has an stu3.Id logical
// identifier. All of the stu3 resource protos are STU3Resources.
type STU3Resource interface {
	descriptor.Message
	GetId() *pb.Id
}

// An stu3Extensible is any descriptor.Message that has stu3.Extensions.
type stu3Extensible interface {
	descriptor.Message
	GetExtension() []*pb.Extension
}

// An stu3Identifiable is any descriptor.Message that has an stu3.Id String
// identifier.
type stu3Identifiable interface {
	descriptor.Message
	GetId() *pb.String
}

// An stu3Element is an stu3Identifiable that also has a means of (un)marshaling
// itself to and from JSON. All of the stu3 primitive primitive are
// stu3Elements.
type stu3Element interface {
	stu3Identifiable
	json.Marshaler
	json.Unmarshaler

	// IsJSONFHIRNode is a no-op function, required internally by this
	// package simply to improve type safety at compile time, instead of using
	// the empty interface{}.
	IsJSONFHIRNode()
}

// MarshalSTU3 IS INCOMPLETE AND SHOULD NOT BE USED YET. It returns the Resource
// in JSON format.
func MarshalSTU3(msg STU3Resource) ([]byte, error) {
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

func (marshalTree) IsJSONFHIRNode() {}

// A marshalNode indicates that an arbitrary type can act as a node of a
// marshalTree.
type marshalNode interface {
	IsJSONFHIRNode()
}

type nodeSlice []marshalNode

func (nodeSlice) IsJSONFHIRNode() {}

// marshalToTree converts the message into marshalTree, allowing for eventual
// marshaling with encoding/json.MarshalJSON(). It acts recursively via calls to
// marshalValue, which may call marshalToTree, and is terminated by discovery of
// stu3Elements in the STU3Resource. The baseNodePath carries the path to the
// node in the tree that is currently being processed, and is used for error
// reporting; therefore the original call should have nodePath = "/".
func marshalToTree(msgInterface descriptor.Message, baseNodePath string) (_ marshalTree, retErr error) {
	nodePath := baseNodePath
	defer func() {
		if retErr != nil {
			retErr = fmt.Errorf("[%s] %v", nodePath, retErr)
		}
	}()

	msgPtr := reflect.ValueOf(msgInterface)
	if msgInterface == nil || msgPtr.IsNil() {
		return nil, nil
	}

	msg := msgPtr.Elem()
	if k := msg.Kind(); k != reflect.Struct {
		return nil, fmt.Errorf("cannot convert non-struct (%s) to tree", k)
	}

	// FieldDescriptorProtos are used to check for choice-type option
	// extensions.
	fldDescs := make(map[string]*dpb.FieldDescriptorProto)
	_, md := descriptor.ForMessage(msgInterface)
	for _, f := range md.Field {
		fldDescs[f.GetName()] = f
	}

	tree := make(marshalTree)
	if _, ok := msgPtr.Interface().(STU3Resource); ok {
		tree["resourceType"] = &pb.String{Value: msg.Type().Name()}
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

		name, ok := jsonName(fld.Tag)
		if !ok {
			continue
		}

		// FHIR defines "choice types" which are equivalent to proto oneof
		// except that they cannot repeat (as per
		// https://www.hl7.org/fhir/formats.html#choice). They are treated
		// identically to the regular types except that they have their type
		// name appended to the JSON property name.
		choice, err := extractIfChoiceType(val, fld, fldDescs)
		if err != nil {
			return nil, fmt.Errorf("detecting choice-type field: %v", err)
		}
		if choice.IsValid() {
			val = choice
			name = fmt.Sprintf("%s%s", name, val.Elem().Type().Name())
		}

		switch val.Kind() {
		case reflect.Ptr:
			nodePath = fmt.Sprintf("%s%s/", baseNodePath, name)

			mv, uscore, err := marshalValue(val, nodePath)
			if err != nil {
				return nil, err
			}

			tree[name] = mv
			if !uscore.empty() {
				tree[fmt.Sprintf("_%s", name)] = uscore
			}
		// A slice is treated in the exact same way as a pointer, but once for
		// each element. The slice of Underscore properties is added iff at
		// leaste one element has a non-nil underscore value.
		case reflect.Slice:
			n := val.Len()
			all := make(nodeSlice, n)
			uscores := make(nodeSlice, n)
			var hasUnderscore bool

			for i := 0; i < n; i++ {
				nodePath = fmt.Sprintf("%s%s[%d]/", baseNodePath, name, i)

				mv, uscore, err := marshalValue(val.Index(i), nodePath)
				if err != nil {
					return nil, err
				}
				all[i] = mv
				if !uscore.empty() {
					uscores[i] = uscore
					hasUnderscore = true
				}
			}

			tree[name] = all
			if hasUnderscore {
				tree[fmt.Sprintf("_%s", name)] = uscores
			}
		}
	}

	return tree, nil
}

// stu3Underscore represents a JSON property prepended with an underscore, used
// for ID and extension attributes of primitives, as described by
// https://www.hl7.org/fhir/json.html#primitive. The Extensions are converted
// to marshalTrees so as to properly handle their Value elements which can be of
// any data type.
type stu3Underscore struct {
	ID        *pb.String    `json:"id,omitempty"`
	Extension []marshalTree `json:"extension,omitempty"`

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

// underscore populates a new stu3Underscore from the msg. It returns an error
// if the msg is not an stu3Element.
func underscore(msg proto.Message, nodePath string) (*stu3Underscore, error) {
	if _, ok := msg.(stu3Element); !ok {
		return nil, fmt.Errorf("underscore properties must only be used for primitive types; got %T", msg)
	}
	u := new(stu3Underscore)
	if i, ok := msg.(stu3Identifiable); ok {
		u.ID = i.GetId()
	}
	if e, ok := msg.(stu3Extensible); ok {
		exts := e.GetExtension()
		u.Extension = make([]marshalTree, len(exts))
		var err error
		for i, ex := range exts {
			u.Extension[i], err = marshalToTree(ex, fmt.Sprintf("%s/%s[%d]", nodePath, "Extension", i))
			if err != nil {
				return nil, err
			}
		}
	}
	return u, nil
}

// marshalValue is the field-level counterpart for marshalToTree, producing a
// single node from a struct field's value. If the field contains an
// stu3Element, itself a marshalNode, it is simply returned, otherwise the
// field's value is recursed back into marshalToTree.
func marshalValue(val reflect.Value, nodePath string) (marshalNode, *stu3Underscore, error) {
	ifc := val.Interface()

	// TODO(arrans) remove this once all Elements have marshalling implemented.
	unimplemented := func(v interface{}) *pb.String {
		return &pb.String{
			Value: fmt.Sprintf("UNIMPLEMENTED: %T", v),
		}
	}

	if el, ok := ifc.(stu3Element); ok {
		u, err := underscore(el, nodePath)
		if err != nil {
			return nil, nil, err
		}
		if _, err := el.MarshalJSON(); err != nil && strings.Contains(err.Error(), "unimplemented") {
			return unimplemented(el), u, nil
		}
		return el, u, nil
	}

	if id, ok := ifc.(stu3Identifiable); ok {
		mt, err := marshalToTree(id, nodePath)
		if err != nil {
			return nil, nil, err
		}
		return mt, nil, nil
	}

	if res, ok := ifc.(STU3Resource); ok {
		mt, err := marshalToTree(res, nodePath)
		if err != nil {
			return nil, nil, err
		}
		return mt, nil, nil
	}

	switch msg := ifc.(type) {
	case *pb.ContainedResource:
		// TODO(arrans) implement marshalling of a ContainedResource (will
		// likely use of extractIfChoiceType). For now, just return nil while
		// implementing everything else.
		return unimplemented(msg), nil, nil
	}

	return nil, nil, fmt.Errorf("unsupported field type %T", ifc)
}

// extractIfChoiceType receives a struct field's value and respective
// StructField, along with a map from proto field names to descriptor protos.
// The StructField is used to extract the protoName from the tag, which is then
// retrieved from the descs map. If the retrieved value indicates that val holds
// a choice type, the underlying value is extracted and returned.
//
// If no error occurs, but it is not a choice-type field, then the returned
// Value will be zero and this can be checked with its IsValid() method.
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
	if t := choice[0].Type(); t.Kind() != reflect.Interface || choice[0].IsNil() {
		return reflect.Value{}, fmt.Errorf("%s.%s() must return interface containing non-nil value; got %s", fld.Name, fn, t.Kind())
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

// jsonName extracts the name from the json section of the tag, and returns it
// in camel case.
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

// snakeToCamel does what it says on the tin. Underscores are removed and, if
// the next character is a lower-case alpha then it is converted to upper case.
// All other characters remain unmodified.
func snakeToCamel(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	var last int
	for i, n := 0, len(s); i < n; i++ {
		if s[i] == '_' {
			b.WriteString(s[last:i])
			// Skip all the underscores
			for i = i + 1; i < n && s[i] == '_'; i++ {
			}
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
