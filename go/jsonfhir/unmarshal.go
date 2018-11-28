package jsonfhir

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/golang/protobuf/proto"
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
	var raw map[string]interface{}
	if err := json.Unmarshal(buf, &raw); err != nil {
		return fmt.Errorf("json.Unmarshal(): %v", err)
	}
	return unmarshalRaw(raw, msg)
}

func unmarshalRaw(raw map[string]interface{}, msg STU3Resource) error {
	return nil
}
