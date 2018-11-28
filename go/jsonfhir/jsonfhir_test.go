package jsonfhir

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"reflect"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"

	pb "github.com/google/fhir/proto/stu3"
)

// prettyJSON returns j indented by json.Indent, with empty prefix, and spaces
// for indentation. If j is not valid JSON then it is annotated as such, and
// returned. If json.Indent returns an error, it is ignored and j is simply
// returned unmodified.
func prettyJSON(j []byte) []byte {
	if !json.Valid(j) {
		b := bytes.NewBufferString("INVALID JSON: ")
		// b.Write always returns nil error.
		b.Write(j)
		return b.Bytes()
	}
	b := new(bytes.Buffer)
	if err := json.Indent(b, j, "", "  "); err != nil {
		return j
	}
	return b.Bytes()
}

func TestMarshalSTU3JSON(t *testing.T) {
	tests := []struct {
		name string
		msg  STU3Resource
		want string
	}{
		{
			name: "resource type other than Patient",
			msg: &pb.PaymentNotice{
				Id: &pb.Id{Value: "final"},
			},
			want: `{"id":"final","resourceType":"PaymentNotice"}`,
		},
		{
			name: "primitive type value only",
			msg: &pb.Patient{
				Active: &pb.Boolean{Value: true},
			},
			want: `{"active":true,"resourceType":"Patient"}`,
		},
		{
			name: "primitive type value with ID",
			msg: &pb.Patient{
				Active: &pb.Boolean{
					Value: true,
					Id:    &pb.String{Value: "is-active"},
				},
			},
			want: `{"_active":{"id":"is-active"},"active":true,"resourceType":"Patient"}`,
		},
		{
			name: "primitive type value with Extensions",
			msg: &pb.Patient{
				Active: &pb.Boolean{
					Value: false,
					Extension: []*pb.Extension{
						{
							Id:  &pb.String{Value: "exta"},
							Url: &pb.Uri{Value: "/ext/a"},
							Value: &pb.Extension_Value{
								Value: &pb.Extension_Value_StringValue{
									StringValue: &pb.String{Value: "extended"},
								},
							},
						},
						{
							Id:  &pb.String{Value: "extb"},
							Url: &pb.Uri{Value: "/ext/b"},
							Value: &pb.Extension_Value{
								Value: &pb.Extension_Value_Boolean{
									Boolean: &pb.Boolean{Value: false},
								},
							},
						},
					},
				},
			},
			want: `{"_active":{"extension":[{"id":"exta","url":"/ext/a","valueString":"extended"},{"id":"extb","url":"/ext/b","valueBoolean":false}]},"active":false,"resourceType":"Patient"}`,
		},
		{
			name: "FHIR choice type (i.e. proto oneof)",
			msg: &pb.Patient{
				Deceased: &pb.Patient_Deceased{
					Deceased: &pb.Patient_Deceased_Boolean{
						Boolean: &pb.Boolean{Value: true},
					},
				},
			},
			want: `{"deceasedBoolean":true,"resourceType":"Patient"}`,
		},
		{
			name: "repeated values without underscore attributes",
			msg: &pb.Patient{
				Name: []*pb.HumanName{{
					Given: []*pb.String{
						{Value: "Mary"},
						{Value: "Jane"},
					},
				}},
			},
			want: `{"name":[{"given":["Mary","Jane"]}],"resourceType":"Patient"}`,
		},
		{
			name: "repeated values with underscore attributes",
			msg: &pb.Patient{
				Name: []*pb.HumanName{{
					Given: []*pb.String{
						{Value: "Mary"},
						{
							Value: "Jane",
							Id:    &pb.String{Value: "middle"},
						},
					},
				}},
			},
			want: `{"name":[{"_given":[null,{"id":"middle"}],"given":["Mary","Jane"]}],"resourceType":"Patient"}`,
		},
		{
			name: "empty slice is not rendered",
			msg: &pb.Patient{
				Name: make([]*pb.HumanName, 0, 4),
			},
			want: `{"resourceType":"Patient"}`,
		},
		// TODO(arrans) test ContainedResources
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MarshalSTU3(tt.msg)
			if err != nil {
				t.Fatalf("MarshalSTU3() got err %v; want nil err", err)
			}
			if want := []byte(tt.want); !reflect.DeepEqual(got, want) {
				t.Errorf("MarshalSTU3()\n\ngot:\n%s\n\nwant:\n%s\n\ngot (pretty-printed):\n\n%s\n\nwant (pretty-printed):\n\n%s", got, want, prettyJSON(got), prettyJSON(want))
			}
		})
	}
}

func TestNewSTU3Resource(t *testing.T) {
	t.Run("errors", func(t *testing.T) {
		tests := []struct {
			typ, want string
		}{
			{
				typ:  "foo",
				want: "name",
			},
			{
				typ:  "Boolean",
				want: "Resource",
			},
		}

		for _, tt := range tests {
			_, err := newSTU3Resource(tt.typ)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("newSTU3Resource(%q) got err %v; want containing %q", tt.typ, err, tt.want)
			}
		}
	})

	t.Run("good", func(t *testing.T) {
		for _, resourceType := range []string{"Patient", "PaymentNotice"} {
			t.Run(resourceType, func(t *testing.T) {
				got, err := newSTU3Resource(resourceType)
				if err != nil {
					t.Fatalf(`newSTU3Resource(%q) got err %v; want nil err`, resourceType, err)
				}

				if got, want := reflect.TypeOf(got).String(), fmt.Sprintf("*stu3.%s", resourceType); got != want {
					t.Errorf("newSTU3Resource(%q) got reflect.Type %s; want %s", resourceType, got, want)
				}
				if v := reflect.ValueOf(got); got == nil || v.Kind() != reflect.Ptr || v.IsNil() {
					t.Errorf("newSTU3Resource(%q) got %T(%v); want non-nil pointer", resourceType, got, got)
				}
			})
		}
	})
}

func TestExamples(t *testing.T) {
	const (
		jsonDir  = "../../testdata/stu3/ndjson"
		protoDir = "../../testdata/stu3/examples"

		jsonExt  = ".ndjson"
		protoExt = ".prototxt"
	)

	tests := []struct {
		name string
	}{
		{
			name: "Patient-null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			json, err := ioutil.ReadFile(path.Join(jsonDir, fmt.Sprintf("%s%s", tt.name, jsonExt)))
			if err != nil {
				t.Fatalf("JSON ReadFile(); got err %v; want nil err", err)
			}
			_ = json

			protoStr, err := ioutil.ReadFile(path.Join(protoDir, fmt.Sprintf("%s%s", tt.name, protoExt)))
			if err != nil {
				t.Fatalf("proto ReadFile(); got err %v; want nil err", err)
			}

			newRes := func() STU3Resource {
				typ := strings.Split(tt.name, "-")[0]
				res, err := newSTU3Resource(typ)
				if err != nil {
					t.Fatalf("NewSTU3Resource(%q) got err %v; want nil err", typ, err)
				}
				return res
			}

			t.Run("proto to JSON", func(t *testing.T) {
				t.Skip("Currently for development purposes only until package is fully implemented")

				msg := newRes()
				if err := proto.UnmarshalText(string(protoStr), msg); err != nil {
					t.Fatalf("unmarshal text proto: %v", err)
				}
				got, err := MarshalSTU3(msg)
				if err != nil {
					t.Fatalf("MarshalSTU3() got err %v; want nil err", err)
				}
				t.Errorf("%s", prettyJSON(got))
			})

			t.Run("JSON to proto", func(t *testing.T) {
				t.Skip("Currently for development purposes only until package is fully implemented")

				msg := newRes()
				if err := UnmarshalSTU3(json, msg); err != nil {
					t.Fatalf("UnmarshalSTU3() got err %v; want nil err", err)
				}
				t.Error(proto.MarshalTextString(msg))
			})
		})
	}
}

func TestExtractIfChoiceType(t *testing.T) {
	// TODO(arrans)
}

func TestJSONName(t *testing.T) {
	// TODO(arrans)
}

func TestSnakeToCamel(t *testing.T) {
	tests := []struct {
		snake, camel string
	}{
		{
			snake: `hello`,
			camel: `hello`,
		},
		{
			snake: `hello_world`,
			camel: `helloWorld`,
		},
		{
			snake: `hello__world`,
			camel: `helloWorld`,
		},
		{
			snake: `he_llo_wo_rld`,
			camel: `heLloWoRld`,
		},
		{
			snake: `hello_`,
			camel: `hello`,
		},
		{
			snake: `hello_3orld`,
			camel: `hello3orld`,
		},
		{
			snake: `hELLo_world`,
			camel: `hELLoWorld`,
		},
	}

	for _, tt := range tests {
		if got, want := snakeToCamel(tt.snake), tt.camel; got != want {
			t.Errorf("snakeToCamel(%s) got %s; want %s", tt.snake, got, want)
		}
	}
}

func TestUnderscore(t *testing.T) {
	// TODO(arrans) test the underscore() function
}
