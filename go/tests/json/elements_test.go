package json

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"

	pb "github.com/google/fhir/proto/stu3"
)

// newEmptyElement returns a new FHIRPrimitive with the same concrete type as
// msg.
func newEmptyElement(t *testing.T, msg pbMarshaler) pbMarshaler {
	t.Helper()
	// TODO(arrans) is there a simpler way to get a new(x) without
	// explicitly having the type?
	concrete := reflect.ValueOf(msg).Elem().Type()
	empty, ok := reflect.New(concrete).Interface().(pbMarshaler)
	if !ok {
		// If this happens then the test is badly coded, not actually failing.
		t.Fatalf("bad test setup; got ok==false when casting zero-valued proto message to Message interface")
	}
	return empty
}

type pbMarshaler interface {
	proto.Message
	json.Marshaler
	json.Unmarshaler
}

// TODO(arrans) test "null" JSON input

func TestGoodConversions(t *testing.T) {
	tests := []struct {
		msg  pbMarshaler
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
			msg: &pb.Code{
				Value: `secret`,
			},
			json: `"secret"`,
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
			msg: &pb.Id{
				Value: `1234`,
			},
			json: `"1234"`,
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
			msg: &pb.Markdown{
				Value: "foobar",
			},
			json: `"foobar"`,
		},
		{
			msg: &pb.Oid{
				Value: "urn:oid:1.2345.678.9",
			},
			json: `"urn:oid:1.2345.678.9"`,
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
		{
			msg: &pb.Uri{
				Value: `urn:uuid:53fefa32-fcbb-4ff8-8a92-55ee120877b7`,
			},
			json: `"urn:uuid:53fefa32-fcbb-4ff8-8a92-55ee120877b7"`,
		},
		{
			msg: &pb.Uuid{
				Value: `urn:uuid:53fefa32-fcbb-4ff8-8a92-55ee120877b7`,
			},
			json: `"urn:uuid:53fefa32-fcbb-4ff8-8a92-55ee120877b7"`,
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
		msg             pbMarshaler
		json            string
		wantErrContains string
	}{
		{
			msg:             &pb.Code{},
			json:            `" leading space"`,
			wantErrContains: "regex",
		},
		{
			msg:             &pb.Code{},
			json:            `"trailing space "`,
			wantErrContains: "regex",
		},
		{
			msg:             &pb.Code{},
			json:            `"two  spaces"`,
			wantErrContains: "regex",
		},
		{
			msg:             &pb.Code{},
			json:            `42`,
			wantErrContains: "string",
		},
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
			msg:             &pb.Markdown{},
			json:            `42`,
			wantErrContains: "string",
		},
		{
			msg:             &pb.Oid{},
			json:            `"urn:oid:03.1234"`,
			wantErrContains: "regex", // leading zero
		},
		{
			msg:             &pb.Oid{},
			json:            `1234.5678`,
			wantErrContains: "string",
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
			msg:             &pb.String{},
			json:            `42`,
			wantErrContains: "string",
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
		{
			msg:             &pb.Uuid{},
			json:            `"urn:uuid:xyz"`,
			wantErrContains: "regex",
		},
		{
			msg:             &pb.Uuid{},
			json:            `42`,
			wantErrContains: "string",
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
		msg             pbMarshaler
		wantErrContains string
	}{
		// See TODO in pb.Date.MarshalJSON re disabling of this test.
		// {
		// 	msg: &pb.Date{
		// 		Timezone: "+0100",
		// 	},
		// 	wantErrContains: "zone",
		// },
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
