package jsonfhir

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/golang/protobuf/descriptor"
	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	pb "github.com/google/fhir/proto/stu3"
	jsoniter "github.com/json-iterator/go"
)

// value returns a new reflect.Value based on the Type. Pointers are non-nil and
// all other types result in simply their zero value.
func value(t reflect.Type) reflect.Value {
	switch t.Kind() {
	case reflect.Ptr:
		return reflect.New(t.Elem())
	default:
		return reflect.Zero(t)
	}
}

// UnmarshalSTU3 IS INCOMPLETE AND SHOULD NOT BE USED YET. It populates the STU3Resource
// from data in the JSON buffer.
func UnmarshalSTU3(buf []byte, msg STU3Resource) error {
	if !json.Valid(buf) {
		return errors.New("invalid JSON")
	}

	cfg := jsoniter.Config{
		UseNumber: true,
	}.Froze()
	var raw map[string]interface{}
	if err := cfg.Unmarshal(buf, &raw); err != nil {
		return fmt.Errorf("jsoniter.Unmarshal(): %v", err)
	}

	return unmarshalRaw(raw, msg, "")
}

func unmarshalRaw(raw map[string]interface{}, msgInterface descriptor.Message, baseNodePath string) (retErr error) {
	nodePath := baseNodePath
	defer func() {
		if retErr != nil {
			retErr = fmt.Errorf("[%s] %v", nodePath, retErr)
		}
	}()

	msgPtr := reflect.ValueOf(msgInterface)
	if msgInterface == nil || msgPtr.IsNil() {
		return fmt.Errorf("cannot unmarshal to %T(%v)", msgInterface, msgInterface)
	}

	msg := msgPtr.Elem()
	if k := msg.Kind(); k != reflect.Struct {
		return fmt.Errorf("cannot unmarshal to non-struct (%s)", k)
	}

	fields := make(map[string]reflect.StructField)
	for i, n := 0, msg.NumField(); i < n; i++ {
		sf := msg.Type().Field(i)
		fields[sf.Name] = sf
	}

	fldDescs := make(map[string]*dpb.FieldDescriptorProto)
	_, md := descriptor.ForMessage(msgInterface)
	for _, f := range md.Field {
		fldDescs[f.GetName()] = f
	}

	for name, val := range raw {
		if name == "resourceType" {
			// TODO(arrans) add check for being same as msg
			continue
		} else if name[0] == '_' {
			continue
		}

		nodePath = fmt.Sprintf("%s/%s", baseNodePath, name)

		fld, ok := fields[upperCaseFirst(name)]
		fldType := fld.Type
		if !ok {
			// TODO(arrans) provide an option to ignore / fail due to unknown
			// input fields. Also check for choice-types which will need their
			// type-suffix extracted here, and fldType overwritten accordingly.
			continue
		}

		emptyMsg := func() (descriptor.Message, error) {
			t := fldType
			if t.Kind() == reflect.Slice {
				t = t.Elem()
			}
			v := value(t)
			if !v.CanInterface() {
				return nil, fmt.Errorf("cannot get interface of value generated for field %s", fld.Name)
			}
			msg, ok := v.Interface().(descriptor.Message)
			if !ok {
				return nil, fmt.Errorf("field %s does not hold a descriptor.Message", fld.Name)
			}
			return msg, nil
		}

		switch val := val.(type) {
		case map[string]interface{}:
			el, err := emptyMsg()
			if err != nil {
				return err
			}
			unmarshalRaw(val, el, nodePath)
			msg.FieldByName(fld.Name).Set(reflect.ValueOf(el))
		case []interface{}:
			np := nodePath
			slice := value(reflect.SliceOf(fld.Type.Elem()))
			for i, v := range val {
				nodePath = fmt.Sprintf("%s[%d]", np, i)
				el, err := emptyMsg()
				if err != nil {
					return err
				}
				unmarshalField(v, el, nodePath)
				slice = reflect.Append(slice, reflect.ValueOf(el))
			}
			msg.FieldByName(fld.Name).Set(slice)
		case interface{}:
			el, err := emptyMsg()
			if err != nil {
				return err
			}
			unmarshalField(val, el, nodePath)
			msg.FieldByName(fld.Name).Set(reflect.ValueOf(el))
		}
	}

	return nil
}

func unmarshalField(val interface{}, msg descriptor.Message, nodePath string) error {
	switch val := val.(type) {
	case map[string]interface{}:
		return unmarshalRaw(val, msg, nodePath)
	case nil:
		return nil
	case string:
		el, ok := msg.(stu3Element)
		if !ok {
			return fmt.Errorf("cannot unmarshal string into %T", msg)
		}
		// TODO(arrans) modify UnmarshalJSON for string types such that they
		// don't need the redundant quote-escaping and unescaping.
		return el.UnmarshalJSON(pb.JSONString(val))
	case bool:
		el, ok := msg.(stu3Element)
		if !ok {
			return fmt.Errorf("cannot unmarshal bool into %T", msg)
		}
		if val {
			return el.UnmarshalJSON([]byte("true"))
		}
		return el.UnmarshalJSON([]byte("false"))
	case json.Number:
		el, ok := msg.(stu3Element)
		if !ok {
			return fmt.Errorf("cannot unmarshal number into %T", msg)
		}
		return el.UnmarshalJSON([]byte(val.String()))
	default:
		return fmt.Errorf("unmarshalling unsupported type %T", val)
	}

	return nil
}

func upperCaseFirst(s string) string {
	if s == "" {
		return ""
	}
	if f := s[0]; f < 'a' || f > 'z' {
		return s
	}
	b := new(strings.Builder)
	b.WriteByte(s[0] + 'A' - 'a')
	if len(s) > 1 {
		b.WriteString(s[1:])
	}
	return b.String()
}
