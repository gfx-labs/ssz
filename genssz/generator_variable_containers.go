package genssz

import (
	"fmt"
	
	"github.com/dave/jennifer/jen"
	"github.com/gfx-labs/ssz"
)

// GenerateVariableContainerCode generates Go code for containers with variable-size fields
func GenerateVariableContainerCode(world *World, schema *Schema, f *jen.File) error {
	// Generate code for each type in the world
	for _, structDef := range schema.Structs {
		// Convert to ssz.Field
		sszField := structDef.ToSSZField()
		
		// Check if this is a container type
		if sszField.Type != ssz.TypeContainer {
			continue
		}
		
		// Check if it has variable-size fields
		hasVariable := false
		for _, field := range sszField.Children {
			refs := make(map[string]ssz.Field)
			for _, s := range schema.Structs {
				refs[s.Name] = s.ToSSZField()
			}
			
			isVar, err := field.IsVariable(refs)
			if err != nil {
				return fmt.Errorf("failed to check if field %s is variable: %w", field.Name, err)
			}
			if isVar {
				hasVariable = true
				break
			}
		}
		
		// Skip if no variable fields
		if !hasVariable {
			continue
		}
		
		// Generate the variable container
		if err := generateVariableContainer(f, sszField, schema); err != nil {
			return fmt.Errorf("failed to generate variable container for %s: %w", structDef.Name, err)
		}
	}
	
	return nil
}

// generateVariableContainer generates a container with fixed offset section and dynamic slices
func generateVariableContainer(f *jen.File, structDef ssz.Field, schema *Schema) error {
	typeName := structDef.Name
	refs := make(map[string]ssz.Field)
	for _, s := range schema.Structs {
		refs[s.Name] = s.ToSSZField()
	}
	
	// Separate fields into fixed and variable
	var fixedFields []ssz.Field
	var variableFields []ssz.Field
	var offsetCount int
	
	for _, field := range structDef.Children {
		isVar, err := field.IsVariable(refs)
		if err != nil {
			return err
		}
		
		if isVar {
			variableFields = append(variableFields, field)
			offsetCount++ // Each variable field needs a 4-byte offset
		} else {
			fixedFields = append(fixedFields, field)
		}
	}
	
	// Calculate fixed section size
	fixedSectionSize := 0
	
	// Add size of all fixed fields
	for _, field := range fixedFields {
		size, err := getFieldSize(field, refs)
		if err != nil {
			return err
		}
		fixedSectionSize += size
	}
	
	// Add size of offsets for variable fields
	fixedSectionSize += offsetCount * 4
	
	// Generate comprehensive struct comment with field layout
	f.Comment(fmt.Sprintf("%s is a variable-size SSZ container with the following layout:", typeName))
	f.Comment("")
	f.Comment("Fixed section layout:")
	
	// Document fixed fields with their offsets
	currentOffset := 0
	for _, field := range fixedFields {
		size, _ := getFieldSize(field, refs)
		endOffset := currentOffset + size - 1
		typeDesc := getTypeDescription(field)
		
		if currentOffset == endOffset {
			f.Comment(fmt.Sprintf("[%3d]      %s (%s)", currentOffset, field.Name, typeDesc))
		} else {
			f.Comment(fmt.Sprintf("[%3d-%3d]  %s (%s)", currentOffset, endOffset, field.Name, typeDesc))
		}
		currentOffset += size
	}
	
	// Document offset fields
	if len(variableFields) > 0 {
		f.Comment("")
		f.Comment("Offset fields (4 bytes each):")
		for _, field := range variableFields {
			endOffset := currentOffset + 3
			f.Comment(fmt.Sprintf("[%3d-%3d]  %s_offset (uint32)", currentOffset, endOffset, field.Name))
			currentOffset += 4
		}
	}
	
	f.Comment("")
	f.Comment(fmt.Sprintf("Total fixed section: %d bytes", fixedSectionSize))
	f.Comment("")
	f.Comment("Variable fields (stored after fixed section):")
	for _, field := range variableFields {
		typeDesc := getVariableTypeDescription(field, schema)
		f.Comment(fmt.Sprintf("- %s (%s)", field.Name, typeDesc))
	}
	
	// Generate the struct
	f.Type().Id(typeName).Struct(append(
		[]jen.Code{
			jen.Comment("Fixed-size section containing fixed fields and offsets"),
			jen.Id("fixed").Op("[").Lit(fixedSectionSize).Op("]").Byte(),
			jen.Line(),
			jen.Comment("Variable-size fields"),
		},
		generateVariableFieldDeclarations(variableFields, schema)...,
	)...)
	f.Line()
	
	// Generate constructor
	if err := generateVariableContainerConstructor(f, typeName, fixedSectionSize, variableFields); err != nil {
		return err
	}
	
	// Generate methods
	if err := generateVariableContainerMethods(f, typeName, structDef, fixedFields, variableFields, fixedSectionSize, schema); err != nil {
		return err
	}
	
	return nil
}

// generateVariableFieldDeclarations generates field declarations for variable fields
func generateVariableFieldDeclarations(fields []ssz.Field, schema *Schema) []jen.Code {
	var declarations []jen.Code
	
	for _, field := range fields {
		fieldName := field.Name
		
		switch field.Type {
		case ssz.TypeList:
			if len(field.Children) > 0 {
				elemType := field.Children[0]
				// Special case: []uint8 should be []byte
				if elemType.Type == ssz.TypeUint8 {
					declarations = append(declarations, 
						jen.Id(fieldName).Op("[]").Byte())
				} else {
					// Check if element is fixed-size
					refs := make(map[string]ssz.Field)
					for _, s := range schema.Structs {
						refs[s.Name] = s.ToSSZField()
					}
					
					isFixed := false
					switch elemType.Type {
					case ssz.TypeUint16, ssz.TypeUint32, ssz.TypeUint64, ssz.TypeBoolean:
						isFixed = true
					case ssz.TypeRef:
						if refField, ok := refs[elemType.Ref]; ok {
							isVar, _ := refField.IsVariable(refs)
							isFixed = !isVar
						}
					}
					
					if isFixed {
						// Fixed-size elements - store as contiguous []byte
						declarations = append(declarations, 
							jen.Id(fieldName).Op("[]").Byte())
					} else {
						// Variable-size elements - store as slice of type
						declarations = append(declarations, 
							jen.Id(fieldName).Op("[]").Add(getElementGoType(elemType, schema)))
					}
				}
			}
		case ssz.TypeBitList:
			declarations = append(declarations, jen.Id(fieldName).Op("[]").Byte())
		case ssz.TypeRef:
			// Check if the referenced type is variable
			refs := make(map[string]ssz.Field)
			for _, s := range schema.Structs {
				refs[s.Name] = s.ToSSZField()
			}
			
			if refField, ok := refs[field.Ref]; ok {
				isVar, _ := refField.IsVariable(refs)
				if isVar {
					// Variable container reference - use pointer to the struct type
					declarations = append(declarations, jen.Id(fieldName).Op("*").Id(field.Ref))
				} else {
					// Fixed-size reference - store as []byte
					declarations = append(declarations, jen.Id(fieldName).Op("[]").Byte())
				}
			} else {
				// Unknown ref - default to []byte
				declarations = append(declarations, jen.Id(fieldName).Op("[]").Byte())
			}
		default:
			// This shouldn't happen if IsVariable works correctly
			declarations = append(declarations, jen.Comment(fmt.Sprintf("// TODO: %s field type %s", fieldName, field.Type)))
		}
	}
	
	return declarations
}

// generateVariableContainerConstructor generates the constructor for a variable container
func generateVariableContainerConstructor(f *jen.File, typeName string, fixedSize int, variableFields []ssz.Field) error {
	f.Comment(fmt.Sprintf("New%s creates a new %s", typeName, typeName))
	f.Func().Id("New" + typeName).Params().Op("*").Id(typeName).Block(
		jen.Return(jen.Op("&").Id(typeName).Values()),
	)
	f.Line()
	
	return nil
}

// generateVariableContainerMethods generates methods for a variable container
func generateVariableContainerMethods(f *jen.File, typeName string, structDef ssz.Field, 
	fixedFields, variableFields []ssz.Field, fixedSectionSize int, schema *Schema) error {
	
	refs := make(map[string]ssz.Field)
	for _, s := range schema.Structs {
		refs[s.Name] = s.ToSSZField()
	}
	
	// Calculate field offsets within fixed section
	fieldOffsets := make(map[string]int)
	currentOffset := 0
	
	// Fixed fields come first
	for _, field := range fixedFields {
		fieldOffsets[field.Name] = currentOffset
		size, _ := getFieldSize(field, refs)
		currentOffset += size
	}
	
	// Offsets for variable fields come after fixed fields
	offsetsStart := currentOffset
	for i, field := range variableFields {
		fieldOffsets[field.Name+"_offset"] = offsetsStart + i*4
	}
	
	// Generate Fixed() method
	f.Comment("Fixed returns false as this container has variable-size fields")
	f.Func().Params(jen.Id("c").Op("*").Id(typeName)).Id("Fixed").Params().Bool().Block(
		jen.Return(jen.False()),
	)
	f.Line()
	
	// Generate getters and setters for fixed fields
	for _, field := range fixedFields {
		offset := fieldOffsets[field.Name]
		if err := generateFixedFieldAccessors(f, typeName, field, offset, refs); err != nil {
			return err
		}
	}
	
	// Generate offset getters/setters for variable fields
	for i, field := range variableFields {
		offsetPos := fieldOffsets[field.Name+"_offset"]
		if err := generateOffsetAccessors(f, typeName, field, offsetPos, i); err != nil {
			return err
		}
	}
	
	// Generate getters and setters for variable fields that handle entire data and adjust offsets
	for i, field := range variableFields {
		if err := generateVariableFieldAccessors(f, typeName, field, variableFields, i, fieldOffsets, schema); err != nil {
			return err
		}
	}
	
	// Generate SSZSize method
	if err := generateVariableContainerSSZSize(f, typeName, fixedFields, variableFields, schema); err != nil {
		return err
	}
	
	// Generate MarshalSSZ methods
	if err := generateVariableContainerMarshal(f, typeName, structDef, fixedFields, variableFields, 
		fixedSectionSize, fieldOffsets, refs); err != nil {
		return err
	}
	
	// Generate UnmarshalSSZ method
	if err := generateVariableContainerUnmarshal(f, typeName, structDef, fixedFields, variableFields,
		fixedSectionSize, fieldOffsets, refs); err != nil {
		return err
	}
	
	// Generate HashSSZTo and HashTreeRoot methods
	if err := generateVariableContainerHashing(f, typeName, structDef, fixedFields, variableFields,
		fieldOffsets, refs); err != nil {
		return err
	}
	
	return nil
}

// generateFixedFieldAccessors generates getters and setters for fixed fields
func generateFixedFieldAccessors(f *jen.File, typeName string, field ssz.Field, offset int, refs map[string]ssz.Field) error {
	methodName := capitalizeFirst(field.Name)
	
	switch field.Type {
	case ssz.TypeUint8:
		// Getter
		f.Comment(fmt.Sprintf("%s returns the %s field", methodName, field.Name))
		f.Func().Params(jen.Id("c").Op("*").Id(typeName)).Id(methodName).Params().Uint8().Block(
			jen.Return(jen.Id("c").Dot("fixed").Index(jen.Lit(offset))),
		)
		f.Line()
		
		// Setter
		f.Comment(fmt.Sprintf("Set%s sets the %s field", methodName, field.Name))
		f.Func().Params(jen.Id("c").Op("*").Id(typeName)).Id("Set" + methodName).Params(jen.Id("v").Uint8()).Block(
			jen.Id("c").Dot("fixed").Index(jen.Lit(offset)).Op("=").Id("v"),
		)
		f.Line()
		
	case ssz.TypeUint16:
		// Getter
		f.Comment(fmt.Sprintf("%s returns the %s field", methodName, field.Name))
		f.Func().Params(jen.Id("c").Op("*").Id(typeName)).Id(methodName).Params().Uint16().Block(
			jen.Return(jen.Qual("encoding/binary", "LittleEndian").Dot("Uint16").Call(
				jen.Id("c").Dot("fixed").Index(jen.Lit(offset).Op(":").Lit(offset+2)),
			)),
		)
		f.Line()
		
		// Setter
		f.Comment(fmt.Sprintf("Set%s sets the %s field", methodName, field.Name))
		f.Func().Params(jen.Id("c").Op("*").Id(typeName)).Id("Set" + methodName).Params(jen.Id("v").Uint16()).Block(
			jen.Qual("encoding/binary", "LittleEndian").Dot("PutUint16").Call(
				jen.Id("c").Dot("fixed").Index(jen.Lit(offset).Op(":").Lit(offset+2)),
				jen.Id("v"),
			),
		)
		f.Line()
		
	case ssz.TypeUint32:
		// Getter
		f.Comment(fmt.Sprintf("%s returns the %s field", methodName, field.Name))
		f.Func().Params(jen.Id("c").Op("*").Id(typeName)).Id(methodName).Params().Uint32().Block(
			jen.Return(jen.Qual("encoding/binary", "LittleEndian").Dot("Uint32").Call(
				jen.Id("c").Dot("fixed").Index(jen.Lit(offset).Op(":").Lit(offset+4)),
			)),
		)
		f.Line()
		
		// Setter
		f.Comment(fmt.Sprintf("Set%s sets the %s field", methodName, field.Name))
		f.Func().Params(jen.Id("c").Op("*").Id(typeName)).Id("Set" + methodName).Params(jen.Id("v").Uint32()).Block(
			jen.Qual("encoding/binary", "LittleEndian").Dot("PutUint32").Call(
				jen.Id("c").Dot("fixed").Index(jen.Lit(offset).Op(":").Lit(offset+4)),
				jen.Id("v"),
			),
		)
		f.Line()
		
	case ssz.TypeUint64:
		// Getter
		f.Comment(fmt.Sprintf("%s returns the %s field", methodName, field.Name))
		f.Func().Params(jen.Id("c").Op("*").Id(typeName)).Id(methodName).Params().Uint64().Block(
			jen.Return(jen.Qual("encoding/binary", "LittleEndian").Dot("Uint64").Call(
				jen.Id("c").Dot("fixed").Index(jen.Lit(offset).Op(":").Lit(offset+8)),
			)),
		)
		f.Line()
		
		// Setter
		f.Comment(fmt.Sprintf("Set%s sets the %s field", methodName, field.Name))
		f.Func().Params(jen.Id("c").Op("*").Id(typeName)).Id("Set" + methodName).Params(jen.Id("v").Uint64()).Block(
			jen.Qual("encoding/binary", "LittleEndian").Dot("PutUint64").Call(
				jen.Id("c").Dot("fixed").Index(jen.Lit(offset).Op(":").Lit(offset+8)),
				jen.Id("v"),
			),
		)
		f.Line()
		
	case ssz.TypeBoolean:
		// Getter
		f.Comment(fmt.Sprintf("%s returns the %s field", methodName, field.Name))
		f.Func().Params(jen.Id("c").Op("*").Id(typeName)).Id(methodName).Params().Bool().Block(
			jen.Return(jen.Id("c").Dot("fixed").Index(jen.Lit(offset)).Op("!=").Lit(0)),
		)
		f.Line()
		
		// Setter
		f.Comment(fmt.Sprintf("Set%s sets the %s field", methodName, field.Name))
		f.Func().Params(jen.Id("c").Op("*").Id(typeName)).Id("Set" + methodName).Params(jen.Id("v").Bool()).Block(
			jen.If(jen.Id("v")).Block(
				jen.Id("c").Dot("fixed").Index(jen.Lit(offset)).Op("=").Lit(1),
			).Else().Block(
				jen.Id("c").Dot("fixed").Index(jen.Lit(offset)).Op("=").Lit(0),
			),
		)
		f.Line()
		
	// TODO: Add support for other fixed types (vectors, refs, etc)
	}
	
	return nil
}

// generateOffsetAccessors generates methods to get/set offsets for variable fields
func generateOffsetAccessors(f *jen.File, typeName string, field ssz.Field, offsetPos int, fieldIndex int) error {
	// These are internal methods, not exposed to users
	// The offsets are managed automatically by marshal/unmarshal
	
	methodName := fmt.Sprintf("get%sOffset", capitalizeFirst(field.Name))
	f.Comment(fmt.Sprintf("%s returns the offset for the %s field", methodName, field.Name))
	f.Func().Params(jen.Id("c").Op("*").Id(typeName)).Id(methodName).Params().Uint32().Block(
		jen.Return(jen.Qual("encoding/binary", "LittleEndian").Dot("Uint32").Call(
			jen.Id("c").Dot("fixed").Index(jen.Lit(offsetPos).Op(":").Lit(offsetPos+4)),
		)),
	)
	f.Line()
	
	setMethodName := fmt.Sprintf("set%sOffset", capitalizeFirst(field.Name))
	f.Comment(fmt.Sprintf("%s sets the offset for the %s field", setMethodName, field.Name))
	f.Func().Params(jen.Id("c").Op("*").Id(typeName)).Id(setMethodName).Params(jen.Id("offset").Uint32()).Block(
		jen.Qual("encoding/binary", "LittleEndian").Dot("PutUint32").Call(
			jen.Id("c").Dot("fixed").Index(jen.Lit(offsetPos).Op(":").Lit(offsetPos+4)),
			jen.Id("offset"),
		),
	)
	f.Line()
	
	return nil
}

// generateVariableContainerSSZSize generates the SSZSize method
func generateVariableContainerSSZSize(f *jen.File, typeName string, fixedFields, variableFields []ssz.Field, schema *Schema) error {
	f.Comment("SSZSize returns the total size when serialized")
	f.Func().Params(jen.Id("c").Op("*").Id(typeName)).Id("SSZSize").Params().Int().Block(append(
		[]jen.Code{
			jen.Comment("Fixed section size"),
			jen.Id("size").Op(":=").Len(jen.Id("c").Dot("fixed")),
			jen.Line(),
			jen.Comment("Add sizes of variable fields"),
		},
		generateVariableFieldSizes(variableFields, schema)...,
	)...)
	f.Line()
	
	return nil
}

// generateVariableFieldSizes generates code to calculate sizes of variable fields
func generateVariableFieldSizes(fields []ssz.Field, schema *Schema) []jen.Code {
	var statements []jen.Code
	
	refs := make(map[string]ssz.Field)
	for _, s := range schema.Structs {
		refs[s.Name] = s.ToSSZField()
	}
	
	for _, field := range fields {
		switch field.Type {
		case ssz.TypeList:
			if len(field.Children) > 0 && field.Children[0].Type == ssz.TypeUint8 {
				// Byte list
				statements = append(statements,
					jen.Id("size").Op("+=").Len(jen.Id("c").Dot(field.Name)))
			} else if len(field.Children) > 0 {
				// List of other types - need to calculate element sizes
				elemType := field.Children[0]
				switch elemType.Type {
				case ssz.TypeUint16:
					statements = append(statements,
						jen.Id("size").Op("+=").Len(jen.Id("c").Dot(field.Name)).Op("*").Lit(2))
				case ssz.TypeUint32:
					statements = append(statements,
						jen.Id("size").Op("+=").Len(jen.Id("c").Dot(field.Name)).Op("*").Lit(4))
				case ssz.TypeUint64:
					statements = append(statements,
						jen.Id("size").Op("+=").Len(jen.Id("c").Dot(field.Name)).Op("*").Lit(8))
				case ssz.TypeList:
					// List of lists
					if len(elemType.Children) > 0 && elemType.Children[0].Type == ssz.TypeUint8 {
						// List of bytes - each element is a []byte
						statements = append(statements,
							jen.For(jen.Id("_").Op(",").Id("elem").Op(":=").Range().Id("c").Dot(field.Name)).Block(
								jen.Id("size").Op("+=").Len(jen.Id("elem")),
							))
					} else {
						// List of other lists - need SSZSize
						statements = append(statements,
							jen.For(jen.Id("_").Op(",").Id("elem").Op(":=").Range().Id("c").Dot(field.Name)).Block(
								jen.Id("size").Op("+=").Id("elem").Dot("SSZSize").Call(),
							))
					}
				case ssz.TypeRef:
					// List of references
					if refField, ok := refs[elemType.Ref]; ok {
						isVar, _ := refField.IsVariable(refs)
						if isVar {
							// List of variable containers - use SSZSize
							statements = append(statements,
								jen.For(jen.Id("_").Op(",").Id("elem").Op(":=").Range().Id("c").Dot(field.Name)).Block(
									jen.Id("size").Op("+=").Id("elem").Dot("SSZSize").Call(),
								))
						} else {
							// List of fixed-size references - use len
							fixedSize, _ := getFieldSize(refField, refs)
							statements = append(statements,
								jen.Id("size").Op("+=").Len(jen.Id("c").Dot(field.Name)).Op("*").Lit(fixedSize))
						}
					} else {
						// Unknown ref - assume []byte and count bytes
						statements = append(statements,
							jen.For(jen.Id("_").Op(",").Id("elem").Op(":=").Range().Id("c").Dot(field.Name)).Block(
								jen.Id("size").Op("+=").Len(jen.Id("elem")),
							))
					}
				default:
					// Variable size elements
					statements = append(statements,
						jen.For(jen.Id("_").Op(",").Id("elem").Op(":=").Range().Id("c").Dot(field.Name)).Block(
							jen.Id("size").Op("+=").Id("elem").Dot("SSZSize").Call(),
						))
				}
			}
		case ssz.TypeBitList:
			statements = append(statements,
				jen.Id("size").Op("+=").Len(jen.Id("c").Dot(field.Name)))
		case ssz.TypeRef:
			// Check if the referenced type is variable
			if refField, ok := refs[field.Ref]; ok {
				isVar, _ := refField.IsVariable(refs)
				if isVar {
					// Variable container reference - use SSZSize
					statements = append(statements,
						jen.Id("size").Op("+=").Id("c").Dot(field.Name).Dot("SSZSize").Call())
				} else {
					// Fixed-size reference - stored as []byte
					statements = append(statements,
						jen.Id("size").Op("+=").Len(jen.Id("c").Dot(field.Name)))
				}
			} else {
				// Unknown ref - assume []byte
				statements = append(statements,
					jen.Id("size").Op("+=").Len(jen.Id("c").Dot(field.Name)))
			}
		}
	}
	
	statements = append(statements, jen.Return(jen.Id("size")))
	return statements
}

// generateVariableContainerMarshal generates marshal methods
func generateVariableContainerMarshal(f *jen.File, typeName string, structDef ssz.Field,
	fixedFields, variableFields []ssz.Field, fixedSectionSize int, 
	fieldOffsets map[string]int, refs map[string]ssz.Field) error {
	
	// MarshalSSZ
	f.Comment("MarshalSSZ marshals the container to SSZ format")
	f.Func().Params(jen.Id("c").Op("*").Id(typeName)).Id("MarshalSSZ").Params().Params(jen.Op("[]").Byte(), jen.Error()).Block(
		jen.Id("buf").Op(":=").Make(jen.Op("[]").Byte(), jen.Id("c").Dot("SSZSize").Call()),
		jen.Id("_").Op(",").Id("err").Op(":=").Id("c").Dot("MarshalSSZTo").Call(jen.Id("buf")),
		jen.Return(jen.Id("buf"), jen.Id("err")),
	)
	f.Line()
	
	// MarshalSSZTo
	f.Comment("MarshalSSZTo marshals the container to the provided buffer")
	f.Func().Params(jen.Id("c").Op("*").Id(typeName)).Id("MarshalSSZTo").Params(jen.Id("buf").Op("[]").Byte()).Params(jen.Op("[]").Byte(), jen.Error()).Block(append(
		[]jen.Code{
			jen.Comment("Check buffer size"),
			jen.If(jen.Len(jen.Id("buf")).Op("<").Id("c").Dot("SSZSize").Call()).Block(
				jen.Return(jen.Nil(), jen.Qual("fmt", "Errorf").Call(
					jen.Lit("buffer too small: expected at least %d bytes, got %d"),
					jen.Id("c").Dot("SSZSize").Call(),
					jen.Len(jen.Id("buf")),
				)),
			),
			jen.Line(),
			jen.Comment("Initialize offset tracking"),
			jen.Id("currentOffset").Op(":=").Uint32().Call(jen.Len(jen.Id("c").Dot("fixed"))),
			jen.Line(),
		},
		append(
			generateVariableFieldMarshaling(variableFields, fieldOffsets, refs),
			jen.Line(),
			jen.Comment("Copy fixed section (with updated offsets)"),
			jen.Copy(jen.Id("buf").Op("[:").Len(jen.Id("c").Dot("fixed")).Op("]"), jen.Id("c").Dot("fixed").Op("[:]")),
			jen.Return(jen.Id("buf"), jen.Nil()),
		)...,
	)...)
	f.Line()
	
	return nil
}

// generateVariableFieldMarshaling generates code to marshal variable fields
func generateVariableFieldMarshaling(fields []ssz.Field, fieldOffsets map[string]int, refs map[string]ssz.Field) []jen.Code {
	var statements []jen.Code
	
	for _, field := range fields {
		// offsetPos := fieldOffsets[field.Name+"_offset"]  // unused for now
		setOffsetMethod := fmt.Sprintf("set%sOffset", capitalizeFirst(field.Name))
		
		statements = append(statements,
			jen.Comment(fmt.Sprintf("Set offset for %s", field.Name)),
			jen.Id("c").Dot(setOffsetMethod).Call(jen.Id("currentOffset")),
			jen.Line())
		
		switch field.Type {
		case ssz.TypeList:
			if len(field.Children) > 0 && field.Children[0].Type == ssz.TypeUint8 {
				// Byte list - simple copy
				statements = append(statements,
					jen.Comment(fmt.Sprintf("Copy %s data", field.Name)),
					jen.Copy(
						jen.Id("buf").Op("[").Id("currentOffset").Op(":]"),
						jen.Id("c").Dot(field.Name),
					),
					jen.Id("currentOffset").Op("+=").Uint32().Call(jen.Len(jen.Id("c").Dot(field.Name))),
					jen.Line())
			} else if len(field.Children) > 0 {
				// List of other types
				elemType := field.Children[0]
				switch elemType.Type {
				case ssz.TypeUint16:
					statements = append(statements,
						jen.Comment(fmt.Sprintf("Marshal %s elements", field.Name)),
						jen.For(jen.Id("_").Op(",").Id("v").Op(":=").Range().Id("c").Dot(field.Name)).Block(
							jen.Qual("encoding/binary", "LittleEndian").Dot("PutUint16").Call(
								jen.Id("buf").Op("[").Id("currentOffset").Op(":").Id("currentOffset").Op("+").Lit(2).Op("]"),
								jen.Id("v"),
							),
							jen.Id("currentOffset").Op("+=").Lit(2),
						),
						jen.Line())
				case ssz.TypeUint32:
					statements = append(statements,
						jen.Comment(fmt.Sprintf("Marshal %s elements", field.Name)),
						jen.For(jen.Id("_").Op(",").Id("v").Op(":=").Range().Id("c").Dot(field.Name)).Block(
							jen.Qual("encoding/binary", "LittleEndian").Dot("PutUint32").Call(
								jen.Id("buf").Op("[").Id("currentOffset").Op(":").Id("currentOffset").Op("+").Lit(4).Op("]"),
								jen.Id("v"),
							),
							jen.Id("currentOffset").Op("+=").Lit(4),
						),
						jen.Line())
				case ssz.TypeUint64:
					statements = append(statements,
						jen.Comment(fmt.Sprintf("Marshal %s elements", field.Name)),
						jen.For(jen.Id("_").Op(",").Id("v").Op(":=").Range().Id("c").Dot(field.Name)).Block(
							jen.Qual("encoding/binary", "LittleEndian").Dot("PutUint64").Call(
								jen.Id("buf").Op("[").Id("currentOffset").Op(":").Id("currentOffset").Op("+").Lit(8).Op("]"),
								jen.Id("v"),
							),
							jen.Id("currentOffset").Op("+=").Lit(8),
						),
						jen.Line())
				case ssz.TypeList:
					// List of lists - check if it's a list of bytes
					if len(elemType.Children) > 0 && elemType.Children[0].Type == ssz.TypeUint8 {
						// List of bytes - each element is []byte
						statements = append(statements,
							jen.Comment(fmt.Sprintf("Marshal %s elements (list of bytes)", field.Name)),
							jen.For(jen.Id("_").Op(",").Id("elem").Op(":=").Range().Id("c").Dot(field.Name)).Block(
								jen.Copy(
									jen.Id("buf").Op("[").Id("currentOffset").Op(":]"),
									jen.Id("elem"),
								),
								jen.Id("currentOffset").Op("+=").Uint32().Call(jen.Len(jen.Id("elem"))),
							),
							jen.Line())
					} else {
						// List of other types - need SSZSize
						statements = append(statements,
							jen.Comment(fmt.Sprintf("Marshal %s elements (list of other types)", field.Name)),
							jen.For(jen.Id("_").Op(",").Id("elem").Op(":=").Range().Id("c").Dot(field.Name)).Block(
								jen.Block(
									jen.Id("_").Op(",").Id("err").Op(":=").Id("elem").Dot("MarshalSSZTo").Call(
										jen.Id("buf").Op("[").Id("currentOffset").Op(":]"),
									),
									jen.If(jen.Id("err").Op("!=").Nil()).Block(
										jen.Return(jen.Nil(), jen.Id("err")),
									),
									jen.Id("currentOffset").Op("+=").Uint32().Call(jen.Id("elem").Dot("SSZSize").Call()),
								),
							),
							jen.Line())
					}
				case ssz.TypeRef:
					// List of references
					if refField, ok := refs[elemType.Ref]; ok {
						isVar, _ := refField.IsVariable(refs)
						if isVar {
							// List of variable containers - marshal each element
							statements = append(statements,
								jen.Comment(fmt.Sprintf("Marshal %s elements (variable refs)", field.Name)),
								jen.For(jen.Id("_").Op(",").Id("elem").Op(":=").Range().Id("c").Dot(field.Name)).Block(
									jen.Block(
										jen.Id("_").Op(",").Id("err").Op(":=").Id("elem").Dot("MarshalSSZTo").Call(
											jen.Id("buf").Op("[").Id("currentOffset").Op(":]"),
										),
										jen.If(jen.Id("err").Op("!=").Nil()).Block(
											jen.Return(jen.Nil(), jen.Id("err")),
										),
										jen.Id("currentOffset").Op("+=").Uint32().Call(jen.Id("elem").Dot("SSZSize").Call()),
									),
								),
								jen.Line())
						} else {
							// List of fixed-size references - treat as [][]byte
							statements = append(statements,
								jen.Comment(fmt.Sprintf("Marshal %s elements (fixed refs)", field.Name)),
								jen.For(jen.Id("_").Op(",").Id("elem").Op(":=").Range().Id("c").Dot(field.Name)).Block(
									jen.Copy(
										jen.Id("buf").Op("[").Id("currentOffset").Op(":]"),
										jen.Id("elem"),
									),
									jen.Id("currentOffset").Op("+=").Uint32().Call(jen.Len(jen.Id("elem"))),
								),
								jen.Line())
						}
					} else {
						// Unknown ref - assume []byte
						statements = append(statements,
							jen.Comment(fmt.Sprintf("Marshal %s elements (unknown refs)", field.Name)),
							jen.For(jen.Id("_").Op(",").Id("elem").Op(":=").Range().Id("c").Dot(field.Name)).Block(
								jen.Copy(
									jen.Id("buf").Op("[").Id("currentOffset").Op(":]"),
									jen.Id("elem"),
								),
								jen.Id("currentOffset").Op("+=").Uint32().Call(jen.Len(jen.Id("elem"))),
							),
							jen.Line())
					}
				default:
					// Other variable-size list elements
					statements = append(statements,
						jen.Comment(fmt.Sprintf("Marshal %s elements (variable size)", field.Name)),
						jen.For(jen.Id("_").Op(",").Id("elem").Op(":=").Range().Id("c").Dot(field.Name)).Block(
							jen.Block(
								jen.Id("_").Op(",").Id("err").Op(":=").Id("elem").Dot("MarshalSSZTo").Call(
									jen.Id("buf").Op("[").Id("currentOffset").Op(":]"),
								),
								jen.If(jen.Id("err").Op("!=").Nil()).Block(
									jen.Return(jen.Nil(), jen.Id("err")),
								),
								jen.Id("currentOffset").Op("+=").Uint32().Call(jen.Id("elem").Dot("SSZSize").Call()),
							),
						),
						jen.Line())
				}
			}
		case ssz.TypeBitList:
			// BitList - simple copy like byte list
			statements = append(statements,
				jen.Comment(fmt.Sprintf("Copy %s data", field.Name)),
				jen.Copy(
					jen.Id("buf").Op("[").Id("currentOffset").Op(":]"),
					jen.Id("c").Dot(field.Name),
				),
				jen.Id("currentOffset").Op("+=").Uint32().Call(jen.Len(jen.Id("c").Dot(field.Name))),
				jen.Line())
		case ssz.TypeRef:
			// Check if the referenced type is variable
			if refField, ok := refs[field.Ref]; ok {
				isVar, _ := refField.IsVariable(refs)
				if isVar {
					// Variable container reference - marshal using MarshalSSZTo
					statements = append(statements,
						jen.Comment(fmt.Sprintf("Marshal %s (variable ref)", field.Name)),
						jen.Block(
							jen.Id("_").Op(",").Id("err").Op(":=").Id("c").Dot(field.Name).Dot("MarshalSSZTo").Call(
								jen.Id("buf").Op("[").Id("currentOffset").Op(":]"),
							),
							jen.If(jen.Id("err").Op("!=").Nil()).Block(
								jen.Return(jen.Nil(), jen.Id("err")),
							),
							jen.Id("currentOffset").Op("+=").Uint32().Call(jen.Id("c").Dot(field.Name).Dot("SSZSize").Call()),
						),
						jen.Line())
				} else {
					// Fixed-size reference - stored as []byte
					statements = append(statements,
						jen.Comment(fmt.Sprintf("Copy %s data (fixed ref)", field.Name)),
						jen.Copy(
							jen.Id("buf").Op("[").Id("currentOffset").Op(":]"),
							jen.Id("c").Dot(field.Name),
						),
						jen.Id("currentOffset").Op("+=").Uint32().Call(jen.Len(jen.Id("c").Dot(field.Name))),
						jen.Line())
				}
			} else {
				// Unknown ref - assume []byte
				statements = append(statements,
					jen.Comment(fmt.Sprintf("Copy %s data (unknown ref)", field.Name)),
					jen.Copy(
						jen.Id("buf").Op("[").Id("currentOffset").Op(":]"),
						jen.Id("c").Dot(field.Name),
					),
					jen.Id("currentOffset").Op("+=").Uint32().Call(jen.Len(jen.Id("c").Dot(field.Name))),
					jen.Line())
			}
		}
	}
	
	return statements
}

// generateVariableContainerUnmarshal generates the unmarshal method
func generateVariableContainerUnmarshal(f *jen.File, typeName string, structDef ssz.Field,
	fixedFields, variableFields []ssz.Field, fixedSectionSize int,
	fieldOffsets map[string]int, refs map[string]ssz.Field) error {
	
	f.Comment("UnmarshalSSZ unmarshals the container from SSZ format")
	f.Func().Params(jen.Id("c").Op("*").Id(typeName)).Id("UnmarshalSSZ").Params(jen.Id("buf").Op("[]").Byte()).Error().Block(append(
		[]jen.Code{
			jen.Comment("Check minimum size"),
			jen.If(jen.Len(jen.Id("buf")).Op("<").Lit(fixedSectionSize)).Block(
				jen.Return(jen.Qual("fmt", "Errorf").Call(
					jen.Lit("buffer too small: expected at least %d bytes for fixed section, got %d"),
					jen.Lit(fixedSectionSize),
					jen.Len(jen.Id("buf")),
				)),
			),
			jen.Line(),
			jen.Comment("Copy fixed section"),
			jen.Copy(jen.Id("c").Dot("fixed").Op("[:]"), jen.Id("buf").Op("[:").Lit(fixedSectionSize).Op("]")),
			jen.Line(),
			jen.Comment("Unmarshal variable fields"),
		},
		append(
			generateVariableFieldUnmarshaling(variableFields, fieldOffsets, len(variableFields), refs),
			jen.Return(jen.Nil()),
		)...,
	)...)
	f.Line()
	
	return nil
}

// generateVariableFieldUnmarshaling generates code to unmarshal variable fields
func generateVariableFieldUnmarshaling(fields []ssz.Field, fieldOffsets map[string]int, numFields int, refs map[string]ssz.Field) []jen.Code {
	var statements []jen.Code
	
	for i, field := range fields {
		getOffsetMethod := fmt.Sprintf("get%sOffset", capitalizeFirst(field.Name))
		
		// Calculate field boundaries
		statements = append(statements,
			jen.Comment(fmt.Sprintf("Get boundaries for %s", field.Name)),
			jen.Id(fmt.Sprintf("start%d", i)).Op(":=").Id("c").Dot(getOffsetMethod).Call())
		
		// End offset is either the next field's offset or end of buffer
		if i < numFields-1 {
			nextGetOffsetMethod := fmt.Sprintf("get%sOffset", capitalizeFirst(fields[i+1].Name))
			statements = append(statements,
				jen.Id(fmt.Sprintf("end%d", i)).Op(":=").Id("c").Dot(nextGetOffsetMethod).Call())
		} else {
			statements = append(statements,
				jen.Id(fmt.Sprintf("end%d", i)).Op(":=").Uint32().Call(jen.Len(jen.Id("buf"))))
		}
		
		statements = append(statements, jen.Line())
		
		// Validate boundaries
		statements = append(statements,
			jen.If(jen.Id(fmt.Sprintf("start%d", i)).Op(">").Id(fmt.Sprintf("end%d", i)).Op("||").
				Id(fmt.Sprintf("end%d", i)).Op(">").Uint32().Call(jen.Len(jen.Id("buf")))).Block(
				jen.Return(jen.Qual("fmt", "Errorf").Call(
					jen.Lit(fmt.Sprintf("invalid offsets for field %s: start=%%d, end=%%d, buflen=%%d", field.Name)),
					jen.Id(fmt.Sprintf("start%d", i)),
					jen.Id(fmt.Sprintf("end%d", i)),
					jen.Len(jen.Id("buf")),
				)),
			))
		
		// Unmarshal the field
		switch field.Type {
		case ssz.TypeList:
			if len(field.Children) > 0 && field.Children[0].Type == ssz.TypeUint8 {
				// Byte list
				statements = append(statements,
					jen.Id("c").Dot(field.Name).Op("=").Make(
						jen.Op("[]").Byte(),
						jen.Id(fmt.Sprintf("end%d", i)).Op("-").Id(fmt.Sprintf("start%d", i)),
					),
					jen.Copy(
						jen.Id("c").Dot(field.Name),
						jen.Id("buf").Op("[").Id(fmt.Sprintf("start%d", i)).Op(":").Id(fmt.Sprintf("end%d", i)).Op("]"),
					),
					jen.Line())
			} else if len(field.Children) > 0 {
				// List of other types
				elemType := field.Children[0]
				switch elemType.Type {
				case ssz.TypeUint16:
					statements = append(statements,
						jen.Comment(fmt.Sprintf("Unmarshal %s elements", field.Name)),
						jen.Id(fmt.Sprintf("numElems%d", i)).Op(":=").Parens(
							jen.Id(fmt.Sprintf("end%d", i)).Op("-").Id(fmt.Sprintf("start%d", i)),
						).Op("/").Lit(2),
						jen.Id("c").Dot(field.Name).Op("=").Make(jen.Op("[]").Uint16(), jen.Id(fmt.Sprintf("numElems%d", i))),
						jen.For(
							jen.Id("j").Op(":=").Uint32().Call(jen.Lit(0)),
							jen.Id("j").Op("<").Id(fmt.Sprintf("numElems%d", i)),
							jen.Id("j").Op("++"),
						).Block(
							jen.Id("offset").Op(":=").Id(fmt.Sprintf("start%d", i)).Op("+").Id("j").Op("*").Lit(2),
							jen.Id("c").Dot(field.Name).Op("[").Id("j").Op("]").Op("=").
								Qual("encoding/binary", "LittleEndian").Dot("Uint16").Call(
									jen.Id("buf").Op("[").Id("offset").Op(":").Id("offset").Op("+").Lit(2).Op("]"),
								),
						),
						jen.Line())
				case ssz.TypeUint32:
					statements = append(statements,
						jen.Comment(fmt.Sprintf("Unmarshal %s elements", field.Name)),
						jen.Id(fmt.Sprintf("numElems%d", i)).Op(":=").Parens(
							jen.Id(fmt.Sprintf("end%d", i)).Op("-").Id(fmt.Sprintf("start%d", i)),
						).Op("/").Lit(4),
						jen.Id("c").Dot(field.Name).Op("=").Make(jen.Op("[]").Uint32(), jen.Id(fmt.Sprintf("numElems%d", i))),
						jen.For(
							jen.Id("j").Op(":=").Uint32().Call(jen.Lit(0)),
							jen.Id("j").Op("<").Id(fmt.Sprintf("numElems%d", i)),
							jen.Id("j").Op("++"),
						).Block(
							jen.Id("offset").Op(":=").Id(fmt.Sprintf("start%d", i)).Op("+").Id("j").Op("*").Lit(4),
							jen.Id("c").Dot(field.Name).Op("[").Id("j").Op("]").Op("=").
								Qual("encoding/binary", "LittleEndian").Dot("Uint32").Call(
									jen.Id("buf").Op("[").Id("offset").Op(":").Id("offset").Op("+").Lit(4).Op("]"),
								),
						),
						jen.Line())
				case ssz.TypeUint64:
					statements = append(statements,
						jen.Comment(fmt.Sprintf("Unmarshal %s elements", field.Name)),
						jen.Id(fmt.Sprintf("numElems%d", i)).Op(":=").Parens(
							jen.Id(fmt.Sprintf("end%d", i)).Op("-").Id(fmt.Sprintf("start%d", i)),
						).Op("/").Lit(8),
						jen.Id("c").Dot(field.Name).Op("=").Make(jen.Op("[]").Uint64(), jen.Id(fmt.Sprintf("numElems%d", i))),
						jen.For(
							jen.Id("j").Op(":=").Uint32().Call(jen.Lit(0)),
							jen.Id("j").Op("<").Id(fmt.Sprintf("numElems%d", i)),
							jen.Id("j").Op("++"),
						).Block(
							jen.Id("offset").Op(":=").Id(fmt.Sprintf("start%d", i)).Op("+").Id("j").Op("*").Lit(8),
							jen.Id("c").Dot(field.Name).Op("[").Id("j").Op("]").Op("=").
								Qual("encoding/binary", "LittleEndian").Dot("Uint64").Call(
									jen.Id("buf").Op("[").Id("offset").Op(":").Id("offset").Op("+").Lit(8).Op("]"),
								),
						),
						jen.Line())
				case ssz.TypeList:
					// List of lists - for now just handle list of bytes
					if len(elemType.Children) > 0 && elemType.Children[0].Type == ssz.TypeUint8 {
						statements = append(statements,
							jen.Comment(fmt.Sprintf("TODO: Unmarshal list of lists for %s", field.Name)),
							jen.Comment("For now, treating as empty list"),
							jen.Id("c").Dot(field.Name).Op("=").Make(jen.Op("[][]").Byte(), jen.Lit(0)),
							jen.Line())
					}
				default:
					statements = append(statements,
						jen.Comment(fmt.Sprintf("TODO: Unmarshal %s with element type %s", field.Name, elemType.Type)),
						jen.Line())
				}
			}
		case ssz.TypeBitList:
			// BitList - simple byte copy
			statements = append(statements,
				jen.Id("c").Dot(field.Name).Op("=").Make(
					jen.Op("[]").Byte(),
					jen.Id(fmt.Sprintf("end%d", i)).Op("-").Id(fmt.Sprintf("start%d", i)),
				),
				jen.Copy(
					jen.Id("c").Dot(field.Name),
					jen.Id("buf").Op("[").Id(fmt.Sprintf("start%d", i)).Op(":").Id(fmt.Sprintf("end%d", i)).Op("]"),
				),
				jen.Line())
		case ssz.TypeRef:
			// Check if the referenced type is variable
			if refField, ok := refs[field.Ref]; ok {
				isVar, _ := refField.IsVariable(refs)
				if isVar {
					// Variable container reference - unmarshal using UnmarshalSSZ
					statements = append(statements,
						jen.Comment(fmt.Sprintf("Unmarshal %s (variable ref)", field.Name)),
						jen.Id("c").Dot(field.Name).Op("=").New(jen.Id(field.Ref)),
						jen.If(jen.Id("err").Op(":=").Id("c").Dot(field.Name).Dot("UnmarshalSSZ").Call(
							jen.Id("buf").Op("[").Id(fmt.Sprintf("start%d", i)).Op(":").Id(fmt.Sprintf("end%d", i)).Op("]"),
						), jen.Id("err").Op("!=").Nil()).Block(
							jen.Return(jen.Id("err")),
						),
						jen.Line())
				} else {
					// Fixed-size reference - stored as []byte
					statements = append(statements,
						jen.Comment(fmt.Sprintf("Copy %s data (fixed ref)", field.Name)),
						jen.Id("c").Dot(field.Name).Op("=").Make(
							jen.Op("[]").Byte(),
							jen.Id(fmt.Sprintf("end%d", i)).Op("-").Id(fmt.Sprintf("start%d", i)),
						),
						jen.Copy(
							jen.Id("c").Dot(field.Name),
							jen.Id("buf").Op("[").Id(fmt.Sprintf("start%d", i)).Op(":").Id(fmt.Sprintf("end%d", i)).Op("]"),
						),
						jen.Line())
				}
			} else {
				// Unknown ref - assume []byte
				statements = append(statements,
					jen.Comment(fmt.Sprintf("Copy %s data (unknown ref)", field.Name)),
					jen.Id("c").Dot(field.Name).Op("=").Make(
						jen.Op("[]").Byte(),
						jen.Id(fmt.Sprintf("end%d", i)).Op("-").Id(fmt.Sprintf("start%d", i)),
					),
					jen.Copy(
						jen.Id("c").Dot(field.Name),
						jen.Id("buf").Op("[").Id(fmt.Sprintf("start%d", i)).Op(":").Id(fmt.Sprintf("end%d", i)).Op("]"),
					),
					jen.Line())
			}
		}
	}
	
	return statements
}

// generateVariableContainerHashing generates hashing methods
func generateVariableContainerHashing(f *jen.File, typeName string, structDef ssz.Field,
	fixedFields, variableFields []ssz.Field, fieldOffsets map[string]int, refs map[string]ssz.Field) error {
	
	// HashSSZTo
	f.Comment("HashSSZTo computes the SSZ hash of the container")
	f.Func().Params(jen.Id("c").Op("*").Id(typeName)).Id("HashSSZTo").Params(jen.Id("buf").Op("[]").Byte()).Params(jen.Op("[]").Byte(), jen.Error()).Block(append(
		[]jen.Code{
			jen.Comment("Ensure buffer has at least 32 bytes"),
			jen.If(jen.Len(jen.Id("buf")).Op("<").Lit(32)).Block(
				jen.Return(jen.Nil(), jen.Qual("fmt", "Errorf").Call(
					jen.Lit("buffer too small: need at least 32 bytes, got %d"),
					jen.Len(jen.Id("buf")),
				)),
			),
			jen.Line(),
			jen.Comment("Get buffer for field hashes"),
			jen.Id("numFields").Op(":=").Lit(len(fixedFields) + len(variableFields)),
			jen.Id("poolBuf").Op(":=").Qual("github.com/gfx-labs/ssz/merkle_tree/bufpool", "Get").Call(
				jen.Id("numFields").Op("*").Lit(32),
			),
			jen.Defer().Qual("github.com/gfx-labs/ssz/merkle_tree/bufpool", "Put").Call(jen.Id("poolBuf")),
			jen.Id("hashBuf").Op(":=").Id("poolBuf").Dot("B").Op("[:").Id("numFields").Op("*").Lit(32).Op("]"),
			jen.Line(),
			jen.Comment("Hash each field"),
			jen.Line(),
		},
		generateVariableContainerFieldHashing(fixedFields, variableFields, fieldOffsets, refs)...,
	)...)
	f.Line()
	
	// HashTreeRoot
	f.Comment("HashTreeRoot returns the SSZ hash tree root")
	f.Func().Params(jen.Id("c").Op("*").Id(typeName)).Id("HashTreeRoot").Params().Params(jen.Op("[").Lit(32).Op("]").Byte(), jen.Error()).Block(
		jen.Id("hash").Op(":=").Op("[").Lit(32).Op("]").Byte().Op("{}"),
		jen.Id("_").Op(",").Id("err").Op(":=").Id("c").Dot("HashSSZTo").Call(jen.Id("hash").Op("[:]")),
		jen.Return(jen.Id("hash"), jen.Id("err")),
	)
	f.Line()
	
	return nil
}

// generateVariableContainerFieldHashing generates code to hash each field in a variable container
func generateVariableContainerFieldHashing(fixedFields, variableFields []ssz.Field, fieldOffsets map[string]int, refs map[string]ssz.Field) []jen.Code {
	var statements []jen.Code
	fieldIdx := 0
	
	// Hash fixed fields
	for _, field := range fixedFields {
		offset := fieldOffsets[field.Name]
		
		switch field.Type {
		case ssz.TypeUint8, ssz.TypeBoolean:
			statements = append(statements,
				jen.Comment(fmt.Sprintf("Hash %s (1 byte) - field %d", field.Name, fieldIdx)),
				jen.Id("hashBuf").Op("[").Lit(fieldIdx * 32).Op("]").Op("=").
					Id("c").Dot("fixed").Op("[").Lit(offset).Op("]"),
				jen.For(jen.Id("i").Op(":=").Lit(1), jen.Id("i").Op("<").Lit(32), jen.Id("i").Op("++")).Block(
					jen.Id("hashBuf").Op("[").Lit(fieldIdx * 32).Op("+").Id("i").Op("]").Op("=").Lit(0),
				),
				jen.Line())
		case ssz.TypeUint16:
			statements = append(statements,
				jen.Comment(fmt.Sprintf("Hash %s (2 bytes) - field %d", field.Name, fieldIdx)),
				jen.Copy(
					jen.Id("hashBuf").Op("[").Lit(fieldIdx * 32).Op(":").Lit(fieldIdx * 32 + 2).Op("]"),
					jen.Id("c").Dot("fixed").Op("[").Lit(offset).Op(":").Lit(offset+2).Op("]"),
				),
				jen.For(jen.Id("i").Op(":=").Lit(2), jen.Id("i").Op("<").Lit(32), jen.Id("i").Op("++")).Block(
					jen.Id("hashBuf").Op("[").Lit(fieldIdx * 32).Op("+").Id("i").Op("]").Op("=").Lit(0),
				),
				jen.Line())
		case ssz.TypeUint32:
			statements = append(statements,
				jen.Comment(fmt.Sprintf("Hash %s (4 bytes) - field %d", field.Name, fieldIdx)),
				jen.Copy(
					jen.Id("hashBuf").Op("[").Lit(fieldIdx * 32).Op(":").Lit(fieldIdx * 32 + 4).Op("]"),
					jen.Id("c").Dot("fixed").Op("[").Lit(offset).Op(":").Lit(offset+4).Op("]"),
				),
				jen.For(jen.Id("i").Op(":=").Lit(4), jen.Id("i").Op("<").Lit(32), jen.Id("i").Op("++")).Block(
					jen.Id("hashBuf").Op("[").Lit(fieldIdx * 32).Op("+").Id("i").Op("]").Op("=").Lit(0),
				),
				jen.Line())
		case ssz.TypeUint64:
			statements = append(statements,
				jen.Comment(fmt.Sprintf("Hash %s (8 bytes) - field %d", field.Name, fieldIdx)),
				jen.Copy(
					jen.Id("hashBuf").Op("[").Lit(fieldIdx * 32).Op(":").Lit(fieldIdx * 32 + 8).Op("]"),
					jen.Id("c").Dot("fixed").Op("[").Lit(offset).Op(":").Lit(offset+8).Op("]"),
				),
				jen.For(jen.Id("i").Op(":=").Lit(8), jen.Id("i").Op("<").Lit(32), jen.Id("i").Op("++")).Block(
					jen.Id("hashBuf").Op("[").Lit(fieldIdx * 32).Op("+").Id("i").Op("]").Op("=").Lit(0),
				),
				jen.Line())
		}
		fieldIdx++
	}
	
	// Hash variable fields
	for _, field := range variableFields {
		switch field.Type {
		case ssz.TypeList:
			// For variable fields, we need to compute their merkle root
			statements = append(statements,
				jen.Comment(fmt.Sprintf("Hash %s (variable field) - field %d", field.Name, fieldIdx)),
				jen.Block(
					generateVariableFieldHash(field, fieldIdx),
				),
				jen.Line())
		}
		fieldIdx++
	}
	
	// Compute final merkle root
	statements = append(statements,
		jen.Comment("Compute merkle root of all fields"),
		jen.If(jen.Err().Op(":=").Qual("github.com/gfx-labs/ssz/merkle_tree", "ComputeMerkleRoot").Call(
			jen.Id("hashBuf"),
			jen.Id("hashBuf"),
		), jen.Err().Op("!=").Nil()).Block(
			jen.Return(jen.Nil(), jen.Err()),
		),
		jen.Copy(jen.Id("buf"), jen.Id("hashBuf").Op("[:32]")),
		jen.Return(jen.Id("buf").Op("[:32]"), jen.Nil()),
	)
	
	return statements
}

// generateVariableFieldHash generates code to hash a variable field
func generateVariableFieldHash(field ssz.Field, fieldIdx int) jen.Code {
	// This is a simplified version - in reality, we'd need to handle each type properly
	return jen.Block(
		jen.Comment("TODO: Implement proper hashing for variable field"),
		jen.Comment("For now, just fill with zeros"),
		jen.For(jen.Id("i").Op(":=").Lit(0), jen.Id("i").Op("<").Lit(32), jen.Id("i").Op("++")).Block(
			jen.Id("hashBuf").Op("[").Lit(fieldIdx * 32).Op("+").Id("i").Op("]").Op("=").Lit(0),
		),
	)
}

// getVariableTypeDescription returns a string description of a variable field type
func getVariableTypeDescription(field ssz.Field, schema *Schema) string {
	switch field.Type {
	case ssz.TypeList:
		if len(field.Children) > 0 {
			elemType := field.Children[0]
			elemDesc := getTypeDescription(elemType)
			return fmt.Sprintf("[]%s, limit %d", elemDesc, field.Limit)
		}
		return fmt.Sprintf("list, limit %d", field.Limit)
	case ssz.TypeBitList:
		return fmt.Sprintf("bitlist, limit %d", field.Limit)
	default:
		return fmt.Sprintf("%s (variable)", field.Type)
	}
}

// generateVariableFieldAccessors generates getter and setter methods for a variable field
func generateVariableFieldAccessors(f *jen.File, typeName string, field ssz.Field, 
	variableFields []ssz.Field, fieldIndex int, fieldOffsets map[string]int, schema *Schema) error {
	
	// Generate getter first
	getterName := capitalizeFirst(field.Name)
	fieldName := field.Name
	
	// Determine the return type
	var returnType jen.Code
	switch field.Type {
	case ssz.TypeList:
		if len(field.Children) > 0 && field.Children[0].Type == ssz.TypeUint8 {
			returnType = jen.Op("[]").Byte()
		} else {
			returnType = jen.Op("[]").Add(getElementGoType(field.Children[0], schema))
		}
	case ssz.TypeBitList:
		returnType = jen.Op("[]").Byte()
	case ssz.TypeRef:
		// Check if the referenced type is variable
		refs := make(map[string]ssz.Field)
		for _, s := range schema.Structs {
			refs[s.Name] = s.ToSSZField()
		}
		
		if refField, ok := refs[field.Ref]; ok {
			isVar, _ := refField.IsVariable(refs)
			if isVar {
				// Variable container reference - return pointer to the struct type
				returnType = jen.Op("*").Id(field.Ref)
			} else {
				// Fixed-size reference - return []byte
				returnType = jen.Op("[]").Byte()
			}
		} else {
			// Unknown ref - default to []byte
			returnType = jen.Op("[]").Byte()
		}
	default:
		return fmt.Errorf("unsupported variable field type %s", field.Type)
	}
	
	f.Comment(fmt.Sprintf("%s returns the %s field", getterName, field.Name))
	f.Func().Params(jen.Id("c").Op("*").Id(typeName)).Id(getterName).Params().Add(returnType).Block(
		jen.Return(jen.Id("c").Dot(fieldName)),
	)
	f.Line()
	
	// Now generate the setter
	return generateVariableFieldSetter(f, typeName, field, variableFields, fieldIndex, fieldOffsets, schema)
}

// generateVariableFieldSetter generates a setter method for a variable field that handles offsets
func generateVariableFieldSetter(f *jen.File, typeName string, field ssz.Field, 
	variableFields []ssz.Field, fieldIndex int, fieldOffsets map[string]int, schema *Schema) error {
	
	methodName := "Set" + capitalizeFirst(field.Name)
	fieldName := field.Name
	
	// Determine the parameter type
	var paramType jen.Code
	refs := make(map[string]ssz.Field)
	for _, s := range schema.Structs {
		refs[s.Name] = s.ToSSZField()
	}
	
	switch field.Type {
	case ssz.TypeList:
		if len(field.Children) > 0 && field.Children[0].Type == ssz.TypeUint8 {
			paramType = jen.Op("[]").Byte()
		} else {
			paramType = jen.Op("[]").Add(getElementGoType(field.Children[0], schema))
		}
	case ssz.TypeBitList:
		paramType = jen.Op("[]").Byte()
	case ssz.TypeRef:
		// Check if the referenced type is variable
		if refField, ok := refs[field.Ref]; ok {
			isVar, _ := refField.IsVariable(refs)
			if isVar {
				// Variable container reference - use pointer to the struct type
				paramType = jen.Op("*").Id(field.Ref)
			} else {
				// Fixed-size reference - use []byte
				paramType = jen.Op("[]").Byte()
			}
		} else {
			// Unknown ref - default to []byte
			paramType = jen.Op("[]").Byte()
		}
	default:
		return fmt.Errorf("unsupported variable field type %s", field.Type)
	}
	
	f.Comment(fmt.Sprintf("%s sets the %s field and updates all affected offsets", methodName, field.Name))
	f.Comment("This method automatically adjusts the offsets of subsequent variable fields")
	f.Func().Params(jen.Id("c").Op("*").Id(typeName)).Id(methodName).Params(jen.Id("value").Add(paramType)).BlockFunc(func(g *jen.Group) {
		// Only calculate sizes if there are subsequent fields to update
		hasSubsequentFields := fieldIndex + 1 < len(variableFields)
		
		// Always set the value first
		g.Comment("Set the new value")
		g.Id("c").Dot(fieldName).Op("=").Id("value")
		g.Line()
		
		// Always update this field's offset if it's the first variable field
		if fieldIndex == 0 {
			setOffsetMethod := fmt.Sprintf("set%sOffset", capitalizeFirst(field.Name))
			g.Comment("Update this field's offset (first variable field)")
			g.Id("c").Dot(setOffsetMethod).Call(jen.Uint32().Call(jen.Len(jen.Id("c").Dot("fixed"))))
			g.Line()
		}
		
		if !hasSubsequentFields {
			// No subsequent fields to update
			g.Comment("No subsequent fields to update offsets")
			return
		}
		
		// Calculate old size of this field
		g.Comment("Calculate the size difference")
		g.Id("oldSize").Op(":=").Lit(0)
		
		switch field.Type {
		case ssz.TypeList:
			if len(field.Children) > 0 && field.Children[0].Type == ssz.TypeUint8 {
				// Byte list
				g.Id("oldSize").Op("=").Len(jen.Id("c").Dot(fieldName))
			} else if len(field.Children) > 0 {
				// List of other types
				elemType := field.Children[0]
				refs := make(map[string]ssz.Field)
				for _, s := range schema.Structs {
					refs[s.Name] = s.ToSSZField()
				}
				
				isElemFixed := false
				elemSize := 0
				if elemType.Type == ssz.TypeRef {
					if refField, ok := refs[elemType.Ref]; ok {
						isVar, err := refField.IsVariable(refs)
						if err == nil && !isVar {
							isElemFixed = true
							elemSize, _ = getFieldSize(refField, refs)
						}
					}
				} else {
					isVar, _ := elemType.IsVariable(refs)
					if !isVar {
						isElemFixed = true
						elemSize, _ = getFieldSize(elemType, refs)
					}
				}
				
				if isElemFixed && elemSize > 0 {
					// Fixed-size elements
					g.Id("oldSize").Op("=").Len(jen.Id("c").Dot(fieldName)).Op("*").Lit(elemSize)
				} else {
					// Variable-size elements
					// Special case for [][]byte (list of byte lists)
					if elemType.Type == ssz.TypeList && len(elemType.Children) > 0 && elemType.Children[0].Type == ssz.TypeUint8 {
						// List of byte lists
						g.For(jen.Id("_").Op(",").Id("elem").Op(":=").Range().Id("c").Dot(fieldName)).Block(
							jen.Id("oldSize").Op("+=").Len(jen.Id("elem")),
						)
					} else {
						// Other variable-size elements
						g.For(jen.Id("_").Op(",").Id("elem").Op(":=").Range().Id("c").Dot(fieldName)).Block(
							jen.Id("oldSize").Op("+=").Id("elem").Dot("SSZSize").Call(),
						)
					}
				}
			}
		case ssz.TypeBitList:
			g.Id("oldSize").Op("=").Len(jen.Id("c").Dot(fieldName))
		case ssz.TypeRef:
			// Check if the referenced type is variable
			if refField, ok := refs[field.Ref]; ok {
				isVar, _ := refField.IsVariable(refs)
				if isVar {
					// Variable container reference - get SSZSize
					g.Id("oldSize").Op("=").Id("c").Dot(fieldName).Dot("SSZSize").Call()
				} else {
					// Fixed-size reference - stored as []byte
					g.Id("oldSize").Op("=").Len(jen.Id("c").Dot(fieldName))
				}
			} else {
				// Unknown ref - assume []byte
				g.Id("oldSize").Op("=").Len(jen.Id("c").Dot(fieldName))
			}
		}
		
		g.Line()
		
		// Calculate new size
		g.Id("newSize").Op(":=").Lit(0)
		switch field.Type {
		case ssz.TypeList:
			if len(field.Children) > 0 && field.Children[0].Type == ssz.TypeUint8 {
				// Byte list
				g.Id("newSize").Op("=").Len(jen.Id("value"))
			} else if len(field.Children) > 0 {
				// List of other types
				elemType := field.Children[0]
				refs := make(map[string]ssz.Field)
				for _, s := range schema.Structs {
					refs[s.Name] = s.ToSSZField()
				}
				
				isElemFixed := false
				elemSize := 0
				if elemType.Type == ssz.TypeRef {
					if refField, ok := refs[elemType.Ref]; ok {
						isVar, err := refField.IsVariable(refs)
						if err == nil && !isVar {
							isElemFixed = true
							elemSize, _ = getFieldSize(refField, refs)
						}
					}
				} else {
					isVar, _ := elemType.IsVariable(refs)
					if !isVar {
						isElemFixed = true
						elemSize, _ = getFieldSize(elemType, refs)
					}
				}
				
				if isElemFixed && elemSize > 0 {
					// Fixed-size elements
					g.Id("newSize").Op("=").Len(jen.Id("value")).Op("*").Lit(elemSize)
				} else {
					// Variable-size elements
					// Special case for [][]byte (list of byte lists)
					if elemType.Type == ssz.TypeList && len(elemType.Children) > 0 && elemType.Children[0].Type == ssz.TypeUint8 {
						// List of byte lists
						g.For(jen.Id("_").Op(",").Id("elem").Op(":=").Range().Id("value")).Block(
							jen.Id("newSize").Op("+=").Len(jen.Id("elem")),
						)
					} else {
						// Other variable-size elements
						g.For(jen.Id("_").Op(",").Id("elem").Op(":=").Range().Id("value")).Block(
							jen.Id("newSize").Op("+=").Id("elem").Dot("SSZSize").Call(),
						)
					}
				}
			}
		case ssz.TypeBitList:
			g.Id("newSize").Op("=").Len(jen.Id("value"))
		case ssz.TypeRef:
			// Check if the referenced type is variable
			if refField, ok := refs[field.Ref]; ok {
				isVar, _ := refField.IsVariable(refs)
				if isVar {
					// Variable container reference - get SSZSize
					g.Id("newSize").Op("=").Id("value").Dot("SSZSize").Call()
				} else {
					// Fixed-size reference - stored as []byte
					g.Id("newSize").Op("=").Len(jen.Id("value"))
				}
			} else {
				// Unknown ref - assume []byte
				g.Id("newSize").Op("=").Len(jen.Id("value"))
			}
		}
		
		g.Line()
		
		// Set the new value
		g.Comment("Set the new value")
		g.Id("c").Dot(fieldName).Op("=").Id("value")
		g.Line()
		
		// If size changed, update offsets of subsequent fields
		g.Comment("Update offsets of subsequent variable fields if size changed")
		g.If(jen.Id("oldSize").Op("!=").Id("newSize")).Block(
			jen.Id("sizeDiff").Op(":=").Id("newSize").Op("-").Id("oldSize"),
			jen.Line(),
			jen.Comment("Update offsets for all subsequent variable fields"),
			func() jen.Code {
				var updates []jen.Code
				for i := fieldIndex + 1; i < len(variableFields); i++ {
					nextField := variableFields[i]
					getOffsetMethod := fmt.Sprintf("get%sOffset", capitalizeFirst(nextField.Name))
					setOffsetMethod := fmt.Sprintf("set%sOffset", capitalizeFirst(nextField.Name))
					updates = append(updates,
						jen.Id("c").Dot(setOffsetMethod).Call(
							jen.Uint32().Call(jen.Int().Call(jen.Id("c").Dot(getOffsetMethod).Call()).Op("+").Id("sizeDiff")),
						),
					)
				}
				return jen.Block(updates...)
			}(),
		)
	})
	f.Line()
	
	return nil
}