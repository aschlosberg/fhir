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
		// Name: []*pb.HumanName{
		// 	{
		// 		Family: &pb.String{Value: "Smith"},
		// 		Given: []*pb.String{
		// 			{
		// 				Value: "Mary",
		// 			},
		// 			{
		// 				Value: "Jane",
		// 				Id:    &pb.String{Value: "middle"},
		// 			},
		// 		},
		// 	},
		// },
		Deceased: &pb.Patient_Deceased{
			Deceased: &pb.Patient_Deceased_Boolean{
				Boolean: &pb.Boolean{Value: true},
			},
		},
	}
	got, err := MarshalSTU3JSON(p)
	if err != nil {
		t.Fatalf("MarshalSTU3JSON(%T %s) got err %v; want nil err", p, p, err)
	}
	want := []byte(`{"_birthDate":{"id":"theday"},"active":true,"birthDate":"1986-10-18","deceasedBoolean":true,"resourceType":"Patient"}`)
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
