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
	// TODO(arrans) implement more tests once the function is mature and all
	// STU3Elements actually have MarshalJSON() implemented.
	p := &pb.Patient{
		Active: &pb.Boolean{
			Value: true,
		},
		BirthDate: &pb.Date{
			Id: &pb.String{
				Value: "theday",
			},
			Extension: []*pb.Extension{
				{
					Id:  &pb.String{Value: "thedayext"},
					Url: &pb.Uri{Value: "www.example.com"},
				},
			},
			ValueUs:   529977600000000,
			Precision: pb.Date_DAY,
		},
		Name: []*pb.HumanName{
			{
				Id: &pb.String{Value: "thename"},
				Extension: []*pb.Extension{
					{Id: &pb.String{Value: "nameext"}},
				},
				Family: &pb.String{Value: "Smith"},
				Given: []*pb.String{
					{
						Value: "Mary",
					},
					{
						Value: "Jane",
						Id:    &pb.String{Value: "middle"},
					},
				},
			},
		},
		Deceased: &pb.Patient_Deceased{
			Deceased: &pb.Patient_Deceased_Boolean{
				Boolean: &pb.Boolean{Value: true},
			},
		},
	}
	got, err := MarshalSTU3(p)
	if err != nil {
		t.Fatalf("MarshalSTU3JSON(%T %s) got err %v; want nil err", p, p, err)
	}
	want := []byte(`{"_birthDate":{"id":"theday","extension":[{"id":"thedayext","url":"www.example.com"}]},"active":true,"birthDate":"1986-10-18","deceasedBoolean":true,"name":[{"_given":[null,{"id":"middle"}],"extension":[{"id":"nameext"}],"family":"Smith","given":["Mary","Jane"],"id":"thename"}]}`)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("MarshalSTU3JSON(%T %s) got:\n%s\nwant:\n%s", p, p, got, prettyJSON(want))
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
