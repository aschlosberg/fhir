package json

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/fhir/go/jsonfhir"

	pb "github.com/google/fhir/proto/stu3"
)

// Compile-time checks that each primitive-type proto satisfies the necessary
// interfaces for JSON marshalling.
var (
	_ jsonfhir.STU3Element = (*pb.Base64Binary)(nil)
	_ jsonfhir.STU3Element = (*pb.Boolean)(nil)
	_ jsonfhir.STU3Element = (*pb.Code)(nil)
	_ jsonfhir.STU3Element = (*pb.Date)(nil)
	_ jsonfhir.STU3Element = (*pb.DateTime)(nil)
	_ jsonfhir.STU3Element = (*pb.Decimal)(nil)
	_ jsonfhir.STU3Element = (*pb.Id)(nil)
	_ jsonfhir.STU3Element = (*pb.Instant)(nil)
	_ jsonfhir.STU3Element = (*pb.Integer)(nil)
	_ jsonfhir.STU3Element = (*pb.Markdown)(nil)
	_ jsonfhir.STU3Element = (*pb.Oid)(nil)
	_ jsonfhir.STU3Element = (*pb.PositiveInt)(nil)
	_ jsonfhir.STU3Element = (*pb.String)(nil)
	_ jsonfhir.STU3Element = (*pb.Time)(nil)
	_ jsonfhir.STU3Element = (*pb.UnsignedInt)(nil)
	_ jsonfhir.STU3Element = (*pb.Uri)(nil)
	_ jsonfhir.STU3Element = (*pb.Uuid)(nil)
	// TODO(arrans): why doesn't Xhtml have Extensions? The pb.FHIRMessage
	// interface may need to be split.
	// _ jsonfhir.STU3Element = (*pb.Xhtml)(nil)
)

// newEmptyElement returns a new FHIRPrimitive with the same concrete type as
// msg.
func newEmptyElement(t *testing.T, msg jsonfhir.STU3Element) jsonfhir.STU3Element {
	t.Helper()
	// TODO(arrans) is there a simpler way to get a new(x) without
	// explicitly having the type?
	concrete := reflect.ValueOf(msg).Elem().Type()
	empty, ok := reflect.New(concrete).Interface().(jsonfhir.STU3Element)
	if !ok {
		// If this happens then the test is badly coded, not actually failing.
		t.Fatalf("bad test setup; got ok==false when casting zero-valued proto message to Message interface")
	}
	return empty
}

func TestGoodConversions(t *testing.T) {
	tests := []struct {
		msg  jsonfhir.STU3Element
		json string
	}{
		{
			msg: &pb.Base64Binary{
				Value: []byte("foo\x00bar"),
			},
			json: `"Zm9vAGJhcg=="`,
		},
		{
			msg: &pb.Boolean{
				Value: true,
			},
			json: `true`,
		},
		{
			msg: &pb.Boolean{
				Value: false,
			},
			json: `false`,
		},
		{
			msg: &pb.Date{
				ValueUs:   504921600000000,
				Precision: pb.Date_YEAR,
			},
			json: `"1986"`,
		},
		{
			msg: &pb.Date{
				ValueUs:   528508800000000,
				Precision: pb.Date_MONTH,
			},
			json: `"1986-10"`,
		},
		{
			msg: &pb.Date{
				ValueUs:   529977600000000,
				Precision: pb.Date_DAY,
			},
			json: `"1986-10-18"`,
		},
		{
			msg: &pb.Decimal{
				Value: `2.71828`,
			},
			json: `2.71828`,
		},
		{
			msg: &pb.Decimal{
				Value: `-3.14159`,
			},
			json: `-3.14159`,
		},
		{
			msg: &pb.Decimal{
				Value: `0.000`,
			},
			json: `0.000`,
		},
		{
			// Very important for Decimal as they must maintain precision.
			msg: &pb.Decimal{
				Value: `0.14285700000`,
			},
			json: `0.14285700000`,
		},
		{
			msg: &pb.Decimal{
				Value: `42`,
			},
			json: `42`,
		},
		{
			msg: &pb.Integer{
				Value: 0,
			},
			json: `0`,
		},
		{
			msg: &pb.Integer{
				Value: 42,
			},
			json: `42`,
		},
		{
			msg: &pb.Integer{
				Value: -42,
			},
			json: `-42`,
		},
		{
			msg: &pb.Integer{
				Value: math.MaxInt32,
			},
			json: `2147483647`,
		},
		{
			msg: &pb.Integer{
				Value: math.MinInt32,
			},
			json: `-2147483648`,
		},
		{
			msg: &pb.PositiveInt{
				Value: 1,
			},
			json: `1`,
		},
		{
			msg: &pb.PositiveInt{
				Value: 42,
			},
			json: `42`,
		},
		{
			msg: &pb.PositiveInt{
				Value: math.MaxUint32,
			},
			json: `4294967295`,
		},
		{
			msg: &pb.String{
				Value: `hello world`,
			},
			json: `"hello world"`,
		},
		{
			msg: &pb.String{
				Value: `"double" 'single'`,
			},
			json: `"\"double\" 'single'"`,
		},
		{
			msg: &pb.UnsignedInt{
				Value: 0,
			},
			json: `0`,
		},
		{
			msg: &pb.UnsignedInt{
				Value: 42,
			},
			json: `42`,
		},
		{
			msg: &pb.UnsignedInt{
				Value: math.MaxUint32,
			},
			json: `4294967295`,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%T", tt.msg), func(t *testing.T) {
			t.Run("proto to JSON", func(t *testing.T) {
				got, err := json.Marshal(tt.msg)
				if err != nil {
					t.Fatalf("json.Marshal() got err %v; want nil err", err)
				}
				if got, want := string(got), tt.json; got != want {
					t.Errorf("marshalling %T(%+v) got JSON %s; want %s", tt.msg, tt.msg, got, want)
				}
			})

			t.Run("JSON to proto", func(t *testing.T) {
				got := newEmptyElement(t, tt.msg)
				if err := json.Unmarshal([]byte(tt.json), got); err != nil {
					t.Fatalf("json.Unmarshal(%s, %T) got err %v; want nil err", tt.json, tt.msg, err)
				}
				if want := tt.msg; !proto.Equal(got, want) {
					t.Errorf("unmarshalling JSON %s got %+v; want %+v", tt.json, got, want)
				}
			})
		})
	}
}

func TestBadJSON(t *testing.T) {
	tests := []struct {
		// msg is used merely to define the type to which the JSON should be
		// unmarshalled.
		msg             jsonfhir.STU3Element
		json            string
		wantErrContains string
	}{
		{
			msg:             &pb.Date{},
			json:            `"1986-"`,
			wantErrContains: "regex",
		},
		{
			msg:             &pb.Decimal{},
			json:            `"42y"`,
			wantErrContains: "regex",
		},
		{
			msg:             &pb.Integer{},
			json:            `"x"`,
			wantErrContains: "regex",
		},
		{
			msg:             &pb.Integer{},
			json:            fmt.Sprintf("%d", int64(math.MaxInt32)+1),
			wantErrContains: "32",
		},
		{
			msg:             &pb.Integer{},
			json:            fmt.Sprintf("%d", int64(math.MinInt32)-1),
			wantErrContains: "32",
		},
		{
			msg:             &pb.PositiveInt{},
			json:            `"y"`,
			wantErrContains: "regex",
		},
		{
			msg:  &pb.PositiveInt{},
			json: `0`,
			// there is also an explicit test, but the regex catches it first
			wantErrContains: "regex",
		},
		{
			msg:             &pb.PositiveInt{},
			json:            fmt.Sprintf("%d", uint64(math.MaxUint32)+1),
			wantErrContains: "32",
		},
		{
			msg:             &pb.String{},
			json:            `""`,
			wantErrContains: "empty",
		},
		{
			msg:             &pb.UnsignedInt{},
			json:            `"z"`,
			wantErrContains: "regex",
		},
		{
			msg:             &pb.UnsignedInt{},
			json:            fmt.Sprintf("%d", uint64(math.MaxUint32)+1),
			wantErrContains: "32",
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%T", tt.msg), func(t *testing.T) {
			got := newEmptyElement(t, tt.msg)
			if err := json.Unmarshal([]byte(tt.json), got); err == nil {
				t.Fatalf("json.Unmarshal(%s, %T) got nil err; want err", tt.json, got)
			} else if !strings.Contains(err.Error(), tt.wantErrContains) {
				t.Fatalf("json.Unmarshal(%s, %T) got err %v; want containing %q", tt.json, got, err, tt.wantErrContains)
			}
		})
	}
}

func TestBadProto(t *testing.T) {
	tests := []struct {
		// msg is used merely to define the type to which the JSON should be
		msg             jsonfhir.STU3Element
		wantErrContains string
	}{
		{
			msg: &pb.Date{
				Timezone: "+0100",
			},
			wantErrContains: "zone",
		},
		{
			msg: &pb.Decimal{
				Value: "42x",
			},
			wantErrContains: "regex",
		},
		{
			msg: &pb.String{
				Value: "",
			},
			wantErrContains: "empty",
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%T", tt.msg), func(t *testing.T) {
			if _, err := json.Marshal(tt.msg); err == nil {
				t.Fatalf("json.Marshal(%T(%+v)) got nil err; want err", tt.msg, tt.msg)
			} else if !strings.Contains(err.Error(), tt.wantErrContains) {
				t.Fatalf("json.Marshal(%T(%+v)) got err %v; want containing %q", tt.msg, tt.msg, err, tt.wantErrContains)
			}
		})
	}
}

// // TODO(arrans) test pb.Date.Time() rounding
