/*
 * Copyright 2018 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package stu3 implements JSON marshalling for primitive data types in the FHIR
// STU3 proto.
package stu3

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/descriptor"
	"github.com/golang/protobuf/proto"
)

const (
	doubleQuote        = `"`
	escapedDoubleQuote = `\"`
)

// JSONString returns s as a JSON string in double quotes, escaping any existing
// double quotes.
func JSONString(s string) []byte {
	return []byte(fmt.Sprintf(
		`"%s"`,
		strings.Replace(s, doubleQuote, escapedDoubleQuote, -1),
	))
}

// reMatchString strips surrounding double quotes and unescapes remaining double
// quotes, i.e. undoes JSONString(), before checking the resulting string with
// reMatch() and returning it.
func reMatchString(msg descriptor.Message, json []byte) (string, error) {
	n := len(json)
	if n < 2 || json[0] != '"' || json[n-1] != '"' {
		return "", fmt.Errorf("invalid JSON string %s", json)
	}
	json = bytes.Replace(
		json[1:n-1],
		[]byte(escapedDoubleQuote), []byte(doubleQuote), -1,
	)
	if err := reMatch(msg, json); err != nil {
		return "", err
	}
	return string(json), nil
}

// reMatch checks if the byte slices matches the regular expression in the proto
// option "value_regex". If json contains a JSON string, it must already parsed;
// see reMatchString().
func reMatch(msg descriptor.Message, json []byte) error {
	// TODO(arrans) memoise and benchmark extraction and compilation of regexes.
	_, md := descriptor.ForMessage(msg)
	if !proto.HasExtension(md.Options, E_ValueRegex) {
		return nil
	}
	ex, err := proto.GetExtension(md.Options, E_ValueRegex)
	if err != nil {
		return fmt.Errorf("proto.GetExtension(value_regex) of %T: %v", msg, err)
	}

	switch s := ex.(type) {
	case *string:
		if s == nil {
			return nil
		}
		// The FHIR definition states that "regexes should be qualified with
		// start of string and end of string anchors based on the regex
		// implementation used" - http://hl7.org/fhir/datatypes.html
		reStr := fmt.Sprintf("^%s$", *s)

		re, err := regexp.Compile(reStr)
		if err != nil {
			// This would indicate a bug in the proto conversion process, or
			// that the specification has a bad regex, rather than poorly formed
			// JSON input.
			return fmt.Errorf("compiling regex %s for %T: %v", reStr, msg, err)
		}
		if !re.Match(json) {
			return fmt.Errorf("does not match regex %s for %T", reStr, msg)
		}
		return nil
	default:
		return fmt.Errorf("value_regex for %T option of type %T; expecting string", msg, s)
	}
}

// MarshalJSON returns the message's value as FHIR-conformant JSON.
func (p *Base64Binary) MarshalJSON() ([]byte, error) {
	return JSONString(base64.StdEncoding.EncodeToString(p.Value)), nil
}

// UnmarshalJSON populates the message's value based on the
// FHIR-conformant JSON buffer.
func (p *Base64Binary) UnmarshalJSON(buf []byte) error {
	s, err := reMatchString(p, buf)
	if err != nil {
		return err
	}
	buf, err = base64.StdEncoding.DecodeString(s)
	if err != nil {
		return fmt.Errorf("decoding base64: %v", err)
	}
	p.Value = buf
	return nil
}

// MarshalJSON returns the message's value as FHIR-conformant JSON.
func (p *Boolean) MarshalJSON() ([]byte, error) {
	if p.Value {
		return []byte(`true`), nil
	}
	return []byte(`false`), nil
}

// UnmarshalJSON populates the message's value based on the FHIR-conformant JSON
// buffer.
func (p *Boolean) UnmarshalJSON(buf []byte) error {
	if err := reMatch(p, buf); err != nil {
		return err
	}
	switch string(buf) {
	case `true`:
		p.Value = true
		return nil
	case `false`:
		p.Value = false
		return nil
	}
	return fmt.Errorf(`invalid JSON boolean %s`, buf)
}

// MarshalJSON returns the message's value as FHIR-conformant JSON.
func (p *Code) MarshalJSON() ([]byte, error) {
	return JSONString(p.Value), nil
}

// UnmarshalJSON populates the message's value based on the FHIR-conformant JSON
// buffer.
func (p *Code) UnmarshalJSON(buf []byte) error {
	s, err := reMatchString(p, buf)
	if err != nil {
		return err
	}
	p.Value = s
	return nil
}

const (
	dateLayoutYear  = `2006`
	dateLayoutMonth = `2006-01`
	dateLayoutDay   = `2006-01-02`

	// Go layout format for convenience rather than checking the docs:
	// Mon Jan 2 15:04:05 -0700 MST 2006
)

const (
	unitToMicro = int64(time.Second / time.Microsecond)
	microToNano = int64(time.Microsecond / time.Nanosecond)
)

// utcFromMicroSec returns a time in UTC from unix microseconds.
func utcFromMicroSec(usec int64) time.Time {
	return time.Unix(usec/unitToMicro, (usec%microToNano)*microToNano).UTC()
}

// MarshalJSON returns the message's value as FHIR-conformant JSON.
func (p *Date) MarshalJSON() ([]byte, error) {
	// TODO(arrans) investigate this further as the validation test Patient-null
	// includes a timezone of Australia/Sydney.
	//
	// if p.Timezone != "" {
	//  return nil, fmt.Errorf("FHIR date primitive SHALL not have timezone; has %q", p.Timezone)
	// }
	var layout string
	switch p.Precision {
	case Date_YEAR:
		layout = dateLayoutYear
	case Date_MONTH:
		layout = dateLayoutMonth
	case Date_DAY:
		layout = dateLayoutDay
	default:
		return nil, fmt.Errorf("unknown precision[%d]", p.Precision)
	}
	return JSONString(p.Time().Format(layout)), nil
}

var datePrecisionLayout = map[Date_Precision]string{
	Date_YEAR:  dateLayoutYear,
	Date_MONTH: dateLayoutMonth,
	Date_DAY:   dateLayoutDay,
}

// UnmarshalJSON populates the message's value based on the FHIR-conformant JSON
// buffer.
func (p *Date) UnmarshalJSON(buf []byte) error {
	s, err := reMatchString(p, buf)
	if err != nil {
		return err
	}
	for prec, layout := range datePrecisionLayout {
		t, err := time.ParseInLocation(layout, s, time.UTC)
		if err != nil {
			continue
		}
		p.ValueUs = t.UnixNano() / microToNano
		p.Precision = prec
		return nil
	}
	return fmt.Errorf("invalid date %s", buf)
}

// MarshalJSON returns the message's value as FHIR-conformant JSON.
func (p *DateTime) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("MarshalJSON unimplemented for %T", p)
}

// UnmarshalJSON populates the message's value based on the FHIR-conformant JSON
// buffer.
func (p *DateTime) UnmarshalJSON(buf []byte) error {
	if err := reMatch(p, buf); err != nil {
		return err
	}
	return fmt.Errorf("UnmarshalJSON unimplemented for %T", p)
}

// MarshalJSON returns the message's value as FHIR-conformant JSON.
func (p *Decimal) MarshalJSON() ([]byte, error) {
	if err := reMatch(p, []byte(p.Value)); err != nil {
		return nil, err
	}
	return []byte(p.Value), nil
}

// UnmarshalJSON populates the message's value based on the FHIR-conformant JSON
// buffer.
func (p *Decimal) UnmarshalJSON(buf []byte) error {
	if err := reMatch(p, buf); err != nil {
		return err
	}
	p.Value = string(buf)
	return nil
}

// MarshalJSON returns the message's value as FHIR-conformant JSON.
func (p *Id) MarshalJSON() ([]byte, error) {
	return JSONString(p.Value), nil
}

// UnmarshalJSON populates the message's value based on the FHIR-conformant JSON
// buffer.
func (p *Id) UnmarshalJSON(buf []byte) error {
	s, err := reMatchString(p, buf)
	if err != nil {
		return err
	}
	p.Value = s
	return nil
}

// MarshalJSON returns the message's value as FHIR-conformant JSON.
func (p *Instant) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("MarshalJSON unimplemented for %T", p)
}

// UnmarshalJSON populates the message's value based on the FHIR-conformant JSON
// buffer.
func (p *Instant) UnmarshalJSON(buf []byte) error {
	if err := reMatch(p, buf); err != nil {
		return err
	}
	return fmt.Errorf("UnmarshalJSON unimplemented for %T", p)
}

// MarshalJSON returns the message's value as FHIR-conformant JSON.
func (p *Integer) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%d", p.Value)), nil
}

// UnmarshalJSON populates the message's value based on the FHIR-conformant JSON
// buffer.
func (p *Integer) UnmarshalJSON(buf []byte) error {
	if err := reMatch(p, buf); err != nil {
		return err
	}
	i, err := strconv.ParseInt(string(buf), 10, 32)
	if err != nil {
		return fmt.Errorf("parse %s as 32-bit integer: %v", buf, err)
	}
	p.Value = int32(i)
	return nil
}

// MarshalJSON returns the message's value as FHIR-conformant JSON.
func (p *Markdown) MarshalJSON() ([]byte, error) {
	return JSONString(p.Value), nil
}

// UnmarshalJSON populates the message's value based on the FHIR-conformant JSON
// buffer.
func (p *Markdown) UnmarshalJSON(buf []byte) error {
	s, err := reMatchString(p, buf)
	if err != nil {
		return err
	}
	p.Value = s
	return nil
}

// MarshalJSON returns the message's value as FHIR-conformant JSON.
func (p *Oid) MarshalJSON() ([]byte, error) {
	return JSONString(p.Value), nil
}

// UnmarshalJSON populates the message's value based on the FHIR-conformant JSON
// buffer.
func (p *Oid) UnmarshalJSON(buf []byte) error {
	s, err := reMatchString(p, buf)
	if err != nil {
		return err
	}
	p.Value = s
	return nil
}

// MarshalJSON returns the message's value as FHIR-conformant JSON.
func (p *PositiveInt) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%d", p.Value)), nil
}

// UnmarshalJSON populates the message's value based on the FHIR-conformant JSON
// buffer.
func (p *PositiveInt) UnmarshalJSON(buf []byte) error {
	if err := reMatch(p, buf); err != nil {
		return err
	}
	i, err := strconv.ParseUint(string(buf), 10, 32)
	if err != nil {
		return fmt.Errorf("parse %s as 32-bit unsigned integer: %v", buf, err)
	}
	if i == 0 {
		return errors.New("FHIR positiveInt primitive must be >0")
	}
	p.Value = uint32(i)
	return nil
}

const noEmptyString = "FHIR string primitive can never be empty"

// MarshalJSON returns the message's value as FHIR-conformant JSON.
func (p *String) MarshalJSON() ([]byte, error) {
	if p.Value == "" {
		return nil, errors.New(noEmptyString)
	}
	return JSONString(p.Value), nil
}

// UnmarshalJSON populates the message's value based on the FHIR-conformant JSON
// buffer.
func (p *String) UnmarshalJSON(buf []byte) error {
	s, err := reMatchString(p, buf)
	if err != nil {
		return err
	}
	if s == "" {
		return errors.New(noEmptyString)
	}
	p.Value = s
	return nil
}

// MarshalJSON returns the message's value as FHIR-conformant JSON.
func (p *Time) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("MarshalJSON unimplemented for %T", p)
}

// UnmarshalJSON populates the message's value based on the FHIR-conformant JSON
// buffer.
func (p *Time) UnmarshalJSON(buf []byte) error {
	if err := reMatch(p, buf); err != nil {
		return err
	}
	return fmt.Errorf("UnmarshalJSON unimplemented for %T", p)
}

// MarshalJSON returns the message's value as FHIR-conformant JSON.
func (p *UnsignedInt) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%d", p.Value)), nil
}

// UnmarshalJSON populates the message's value based on the FHIR-conformant JSON
// buffer.
func (p *UnsignedInt) UnmarshalJSON(buf []byte) error {
	if err := reMatch(p, buf); err != nil {
		return err
	}
	i, err := strconv.ParseUint(string(buf), 10, 32)
	if err != nil {
		return fmt.Errorf("parse %s as 32-bit unsigned integer: %v", buf, err)
	}
	p.Value = uint32(i)
	return nil
}

// MarshalJSON returns the message's value as FHIR-conformant JSON.
func (p *Uri) MarshalJSON() ([]byte, error) {
	return JSONString(p.Value), nil
}

// UnmarshalJSON populates the message's value based on the FHIR-conformant JSON
// buffer.
func (p *Uri) UnmarshalJSON(buf []byte) error {
	s, err := reMatchString(p, buf)
	if err != nil {
		return err
	}
	p.Value = s
	return nil
}

// MarshalJSON returns the message's value as FHIR-conformant JSON.
func (p *Uuid) MarshalJSON() ([]byte, error) {
	return JSONString(p.Value), nil
}

// UnmarshalJSON populates the message's value based on the FHIR-conformant JSON
// buffer.
func (p *Uuid) UnmarshalJSON(buf []byte) error {
	s, err := reMatchString(p, buf)
	if err != nil {
		return err
	}
	p.Value = s
	return nil
}

// MarshalJSON returns the message's value as FHIR-conformant JSON.
func (p *Xhtml) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("MarshalJSON unimplemented for %T", p)
}

// UnmarshalJSON populates the message's value based on the FHIR-conformant JSON
// buffer.
func (p *Xhtml) UnmarshalJSON(buf []byte) error {
	if err := reMatch(p, buf); err != nil {
		return err
	}
	return fmt.Errorf("UnmarshalJSON unimplemented for %T", p)
}

// MarshalJSON returns the message as FHIR-conformant JSON.
func (p *Dosage) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("MarshalJSON unimplemented for %T", p)
}

// UnmarshalJSON populates the message based on the FHIR-conformant JSON buffer.
func (p *Dosage) UnmarshalJSON(buf []byte) error {
	if err := reMatch(p, buf); err != nil {
		return err
	}
	return fmt.Errorf("UnmarshalJSON unimplemented for %T", p)
}

// MarshalJSON returns the message as FHIR-conformant JSON.
func (p *Meta) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("MarshalJSON unimplemented for %T", p)
}

// UnmarshalJSON populates the message based on the FHIR-conformant JSON buffer.
func (p *Meta) UnmarshalJSON(buf []byte) error {
	if err := reMatch(p, buf); err != nil {
		return err
	}
	return fmt.Errorf("UnmarshalJSON unimplemented for %T", p)
}

// MarshalJSON returns the message as FHIR-conformant JSON.
func (p *Narrative) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("MarshalJSON unimplemented for %T", p)
}

// UnmarshalJSON populates the message based on the FHIR-conformant JSON buffer.
func (p *Narrative) UnmarshalJSON(buf []byte) error {
	if err := reMatch(p, buf); err != nil {
		return err
	}
	return fmt.Errorf("UnmarshalJSON unimplemented for %T", p)
}

// MarshalJSON returns the message as FHIR-conformant JSON.
func (p *Reference) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("MarshalJSON unimplemented for %T", p)
}

// UnmarshalJSON populates the message based on the FHIR-conformant JSON buffer.
func (p *Reference) UnmarshalJSON(buf []byte) error {
	if err := reMatch(p, buf); err != nil {
		return err
	}
	return fmt.Errorf("UnmarshalJSON unimplemented for %T", p)
}

// IsJSONFHIRNode is a no-op. See jsonfhir.stu3Element interface for details.
func (*Base64Binary) IsJSONFHIRNode() {}

// IsJSONFHIRNode is a no-op. See jsonfhir.stu3Element interface for details.
func (*Boolean) IsJSONFHIRNode() {}

// IsJSONFHIRNode is a no-op. See jsonfhir.stu3Element interface for details.
func (*Code) IsJSONFHIRNode() {}

// IsJSONFHIRNode is a no-op. See jsonfhir.stu3Element interface for details.
func (*Date) IsJSONFHIRNode() {}

// IsJSONFHIRNode is a no-op. See jsonfhir.stu3Element interface for details.
func (*DateTime) IsJSONFHIRNode() {}

// IsJSONFHIRNode is a no-op. See jsonfhir.stu3Element interface for details.
func (*Decimal) IsJSONFHIRNode() {}

// IsJSONFHIRNode is a no-op. See jsonfhir.stu3Element interface for details.
func (*Id) IsJSONFHIRNode() {}

// IsJSONFHIRNode is a no-op. See jsonfhir.stu3Element interface for details.
func (*Instant) IsJSONFHIRNode() {}

// IsJSONFHIRNode is a no-op. See jsonfhir.stu3Element interface for details.
func (*Integer) IsJSONFHIRNode() {}

// IsJSONFHIRNode is a no-op. See jsonfhir.stu3Element interface for details.
func (*Markdown) IsJSONFHIRNode() {}

// IsJSONFHIRNode is a no-op. See jsonfhir.stu3Element interface for details.
func (*Oid) IsJSONFHIRNode() {}

// IsJSONFHIRNode is a no-op. See jsonfhir.stu3Element interface for details.
func (*PositiveInt) IsJSONFHIRNode() {}

// IsJSONFHIRNode is a no-op. See jsonfhir.stu3Element interface for details.
func (*String) IsJSONFHIRNode() {}

// IsJSONFHIRNode is a no-op. See jsonfhir.stu3Element interface for details.
func (*Time) IsJSONFHIRNode() {}

// IsJSONFHIRNode is a no-op. See jsonfhir.stu3Element interface for details.
func (*UnsignedInt) IsJSONFHIRNode() {}

// IsJSONFHIRNode is a no-op. See jsonfhir.stu3Element interface for details.
func (*Uri) IsJSONFHIRNode() {}

// IsJSONFHIRNode is a no-op. See jsonfhir.stu3Element interface for details.
func (*Uuid) IsJSONFHIRNode() {}

// IsJSONFHIRNode is a no-op. See jsonfhir.stu3Element interface for details.
func (*Dosage) IsJSONFHIRNode() {}

// IsJSONFHIRNode is a no-op. See jsonfhir.stu3Element interface for details.
func (*Extension) IsJSONFHIRNode() {}

// IsJSONFHIRNode is a no-op. See jsonfhir.stu3Element interface for details.
func (*Meta) IsJSONFHIRNode() {}

// IsJSONFHIRNode is a no-op. See jsonfhir.stu3Element interface for details.
func (*Narrative) IsJSONFHIRNode() {}

// IsJSONFHIRNode is a no-op. See jsonfhir.stu3Element interface for details.
func (*Reference) IsJSONFHIRNode() {}
