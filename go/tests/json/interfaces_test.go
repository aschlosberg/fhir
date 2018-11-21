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

package json

import (
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
