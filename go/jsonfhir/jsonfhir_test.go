package jsonfhir

import (
	"reflect"
	"testing"

	pb "github.com/google/fhir/proto/stu3"
)

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
			ValueUs:   529977600000000,
			Precision: pb.Date_DAY,
		},
		// ManagingOrganization: &pb.Reference{
		// 	Display: &pb.String{
		// 		Value: "foo",
		// 	},
		// },
	}
	got, err := MarshalSTU3JSON(p)
	if err != nil {
		t.Fatalf("MarshalSTU3JSON(%T %s) got err %v; want nil err", p, p, err)
	}
	want := []byte(`{"_birthDate":{"id":"theday"},"active":true,"birthDate":"1986-10-18","resourceType":"Patient"}`)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("MarshalSTU3JSON(%T %s) got:\n%s\nwant:\n%s", p, p, got, want)
	}
}

func TestJSONName(t *testing.T) {
	// TODO(arrans)
}

func TestSnakeToCamel(t *testing.T) {
	// TODO(arrans)
}
