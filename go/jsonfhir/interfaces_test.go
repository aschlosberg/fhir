package jsonfhir

import pb "github.com/google/fhir/proto/stu3"

// Compile-time checks that all primitive-type and special-purpose elements
// satisfy the necessary interfaces for JSON marshalling.
var (
	// Primitives
	_ stu3Element = (*pb.Base64Binary)(nil)
	_ stu3Element = (*pb.Boolean)(nil)
	_ stu3Element = (*pb.Code)(nil)
	_ stu3Element = (*pb.Date)(nil)
	_ stu3Element = (*pb.DateTime)(nil)
	_ stu3Element = (*pb.Decimal)(nil)
	_ stu3Element = (*pb.Id)(nil)
	_ stu3Element = (*pb.Instant)(nil)
	_ stu3Element = (*pb.Integer)(nil)
	_ stu3Element = (*pb.Markdown)(nil)
	_ stu3Element = (*pb.Oid)(nil)
	_ stu3Element = (*pb.PositiveInt)(nil)
	_ stu3Element = (*pb.String)(nil)
	_ stu3Element = (*pb.Time)(nil)
	_ stu3Element = (*pb.UnsignedInt)(nil)
	_ stu3Element = (*pb.Uri)(nil)
	_ stu3Element = (*pb.Uuid)(nil)
	// TODO(arrans): why doesn't Xhtml have Extensions? The pb.FHIRMessage
	// interface may need to be split.
	// _ stu3Element = (*pb.Xhtml)(nil)

	// Special-purpose elements
	_ stu3Element = (*pb.Dosage)(nil)
	_ stu3Element = (*pb.Extension)(nil)
	_ stu3Element = (*pb.Meta)(nil)
	_ stu3Element = (*pb.Narrative)(nil)
	_ stu3Element = (*pb.Reference)(nil)
)
