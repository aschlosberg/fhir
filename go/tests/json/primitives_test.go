package json

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"

	pb "github.com/google/fhir/proto/stu3"
)

// Compile-time checks that each primitive-type proto satisfies the necessary
// interfaces for JSON marshalling.
var (
	_ pb.FHIRPrimitive = (*pb.Base64Binary)(nil)
	_ pb.FHIRPrimitive = (*pb.Boolean)(nil)
	_ pb.FHIRPrimitive = (*pb.Code)(nil)
	_ pb.FHIRPrimitive = (*pb.Date)(nil)
	_ pb.FHIRPrimitive = (*pb.DateTime)(nil)
	_ pb.FHIRPrimitive = (*pb.Decimal)(nil)
	_ pb.FHIRPrimitive = (*pb.Id)(nil)
	_ pb.FHIRPrimitive = (*pb.Instant)(nil)
	_ pb.FHIRPrimitive = (*pb.Integer)(nil)
	_ pb.FHIRPrimitive = (*pb.Markdown)(nil)
	_ pb.FHIRPrimitive = (*pb.Oid)(nil)
	_ pb.FHIRPrimitive = (*pb.PositiveInt)(nil)
	_ pb.FHIRPrimitive = (*pb.String)(nil)
	_ pb.FHIRPrimitive = (*pb.Time)(nil)
	_ pb.FHIRPrimitive = (*pb.UnsignedInt)(nil)
	_ pb.FHIRPrimitive = (*pb.Uri)(nil)
	_ pb.FHIRPrimitive = (*pb.Uuid)(nil)
	// TODO(arrans): why doesn't Xhtml have Extensions? The pb.FHIRMessage
	// interface may need to be split.
	// _ pb.FHIRPrimitive = (*pb.Xhtml)(nil)
)

// newEmptyPrimitive returns a new FHIRPrimitive with the same concrete type as msg.
func newEmptyPrimitive(t *testing.T, msg pb.FHIRPrimitive) pb.FHIRPrimitive {
	t.Helper()
	// TODO(arrans) is there a simpler way to get a new(x) without
	// explicitly having the type?
	concrete := reflect.ValueOf(msg).Elem().Type()
	empty, ok := reflect.New(concrete).Interface().(pb.FHIRPrimitive)
	if !ok {
		// If this happens then the test is badly coded, not actually failing.
		t.Fatalf("bad test setup; got ok==false when casting zero-valued proto message to Message interface")
	}
	return empty
}

func TestGoodConversions(t *testing.T) {
	tests := []struct {
		msg  pb.FHIRPrimitive
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
		msg := tt.msg
		json := tt.json
		t.Run(fmt.Sprintf("%T", msg), func(t *testing.T) {
			t.Run("proto to JSON", func(t *testing.T) {
				got, err := msg.MarshalFHIRJSONValue()
				if err != nil {
					t.Fatalf("MarshalFHIRJSONValue() got err %v; want nil err", err)
				}
				if got, want := string(got), json; got != want {
					t.Errorf("marshalling %T(%+v) got JSON %s; want %s", msg, msg, got, want)
				}
			})

			t.Run("JSON to proto", func(t *testing.T) {
				got := newEmptyPrimitive(t, msg)
				if err := got.UnmarshalFHIRJSONValue([]byte(json)); err != nil {
					t.Fatalf("%T.UnmarshalFHIRJSONValue(%s) got err %v; want nil err", got, json, err)
				}
				if want := msg; !proto.Equal(got, want) {
					t.Errorf("unmarshalling JSON %s got %+v; want %+v", json, got, want)
				}
			})
		})
	}
}

func TestBadJSON(t *testing.T) {
	tests := []struct {
		// msg is used merely to define the type to which the JSON should be
		// unmarshalled.
		msg             pb.FHIRPrimitive
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
			msg: &pb.Decimal{},
			// leading zero
			json:            `042`,
			wantErrContains: "regex",
		},
		{
			msg:             &pb.Integer{},
			json:            `x`,
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
			json:            `y`,
			wantErrContains: "regex",
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
			json:            `z`,
			wantErrContains: "regex",
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
		msg := tt.msg
		json := tt.json
		t.Run(fmt.Sprintf("%T", msg), func(t *testing.T) {
			got := newEmptyPrimitive(t, msg)
			if err := got.UnmarshalFHIRJSONValue([]byte(json)); err == nil {
				t.Fatalf("UnmarshalFHIRJSONValue(%s, %T) got nil err; want err", json, got)
			} else if !strings.Contains(err.Error(), tt.wantErrContains) {
				t.Fatalf("UnmarshalFHIRJSONValue(%s, %T) got err %v; want containing %q", json, got, err, tt.wantErrContains)
			}
		})
	}
}

func TestBadProto(t *testing.T) {
	tests := []struct {
		// msg is used merely to define the type to which the JSON should be
		msg             pb.FHIRPrimitive
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
		msg := tt.msg
		t.Run(fmt.Sprintf("%T", msg), func(t *testing.T) {
			if _, err := msg.MarshalFHIRJSONValue(); err == nil {
				t.Fatalf("MarshalFHIRJSONValue(%T(%+v)) got nil err; want err", msg, msg)
			} else if !strings.Contains(err.Error(), tt.wantErrContains) {
				t.Fatalf("MarshalFHIRJSONValue(%T(%+v)) got err %v; want containing %q", msg, msg, err, tt.wantErrContains)
			}
		})
	}
}

// // TODO(arrans) test pb.Date.Time() rounding
