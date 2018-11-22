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

// Package jsonfhir provides marshaling between JSON and FHIR protocol buffers.
package jsonfhir

import (
	"encoding/json"

	"github.com/golang/protobuf/proto"
	pb "github.com/google/fhir/proto/stu3"
)

// An STU3Message is any proto.Message that has both stu3.String ID and repeated
// stu3.Extension fields.
type STU3Message interface {
	proto.Message
	GetId() *pb.String
	GetExtension() []*pb.Extension
}

// An STU3Element is a STU3Message that has additional methods for JSON
// (un)marshalling of its value. All of the primitives are STU3Elements.
type STU3Element interface {
	STU3Message
	json.Marshaler
	json.Unmarshaler
}
