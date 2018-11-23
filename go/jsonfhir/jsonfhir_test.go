package jsonfhir

import (
	"testing"

	pb "github.com/google/fhir/proto/stu3"
)

func TestMarshalSTU3JSON(t *testing.T) {
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
		ManagingOrganization: &pb.Reference{
			Display: &pb.String{
				Value: "foo",
			},
		},
	}
	got, err := MarshalSTU3JSON(p)
	t.Errorf("\n\n******************\n\n%v : %s\n\n", err, got)
}
