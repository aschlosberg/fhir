package jsonfhir

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

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
			want: `{"name":[{"_given":[null,{"id":"middle"}],"given":["Mary","Jane"]}],"resourceType":"Patient"}`,,
		},
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

func TestExtractIfChoiceType(t *testing.T) {
	// TODO(arrans)
}

func TestJSONName(t *testing.T) {
	// TODO(arrans)
}

func TestSnakeToCamel(t *testing.T) {
	// TODO(arrans)
}

func TestUnderscore(t *testing.T) {
	// TODO(arrans) test the underscore() function
}
