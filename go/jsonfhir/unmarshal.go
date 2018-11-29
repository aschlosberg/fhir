package jsonfhir

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/golang/protobuf/proto"
	jsoniter "github.com/json-iterator/go"
)

// newSTU3Resource returns a new STU3Resource of the specified concrete type.
func newSTU3Resource(typ string) (STU3Resource, error) {
	t := proto.MessageType(fmt.Sprintf("google.fhir.stu3.proto.%s", typ))
	if t == nil || t.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("no stu3 proto.Message named %s", typ)
	}
	v := reflect.New(t.Elem())
	if !v.CanInterface() {
		return nil, fmt.Errorf("cannot get interface of stu3.%s message", typ)
	}
	ifc := v.Interface()
	res, ok := ifc.(STU3Resource)
	if !ok {
		return nil, fmt.Errorf("%T does not implement STU3Resource interface", ifc)
	}
	return res, nil
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

func unmarshalRaw(raw map[string]interface{}, msg proto.Message, baseNodePath string) error {
	for name, val := range raw {
		nodePath := fmt.Sprintf("%s/%s", baseNodePath, name)
		switch val := val.(type) {
		case map[string]interface{}:
			unmarshalRaw(val, nil, nodePath)
		case []interface{}:
			for i, el := range val {
				unmarshalField(el, nil, fmt.Sprintf("%s[%d]", nodePath, i))
			}
		case interface{}:
			unmarshalField(val, nil, nodePath)
		}
	}

	return nil
}

func unmarshalField(val interface{}, msg proto.Message, nodePath string) error {
	print := func(typ string, a interface{}) {
		fmt.Printf("%s:(%s): %v\n", nodePath, typ, a)
	}

	switch val := val.(type) {
	case map[string]interface{}:
		unmarshalRaw(val, msg, nodePath)
	case string:
		print("string", val)
	case nil:
	case bool:
		print("bool", val)
	case json.Number:
		print("Number", val.String())
	default:
		fmt.Printf("### %s: %T %v\n", nodePath, val, val)
	}
	return nil
}
