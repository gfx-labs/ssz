package solidsszgen

import (
	"fmt"

	"github.com/dave/jennifer/jen"
	"github.com/gfx-labs/cuessz"
)

// generateProgressiveContainer generates an immutable progressive container with builder pattern
// Progressive containers have stable merkle positions with optional fields indicated by active_fields
func generateProgressiveContainer(f *jen.File, defName string, def cuessz.Def, ctx *GeneratorContext) error {
	// Verify active_fields
	if len(def.ActiveFields) == 0 {
		return fmt.Errorf("progressive container %s missing active_fields", defName)
	}

	// Count active fields (should match number of children)
	activeCount := 0
	for _, active := range def.ActiveFields {
		if active == 1 {
			activeCount++
		}
	}
	if activeCount != len(def.Children) {
		return fmt.Errorf("progressive container %s: active field count (%d) != children count (%d)",
			defName, activeCount, len(def.Children))
	}

	// Map children to their active_fields positions
	childToPosition := make(map[int]int) // child index -> merkle position
	positionToChild := make(map[int]int) // merkle position -> child index
	childIdx := 0
	for pos, active := range def.ActiveFields {
		if active == 1 {
			childToPosition[childIdx] = pos
			positionToChild[pos] = childIdx
			childIdx++
		}
	}

	// Check if we need solidssz import
	hasBitlist := false
	for _, child := range def.Children {
		if child.Def.Type == cuessz.TypeBitList {
			hasBitlist = true
			break
		}
	}
	if hasBitlist {
		f.ImportName("github.com/gfx-labs/ssz/pkg/solidssz", "solidssz")
	}

	// Separate fields into fixed and variable
	fixedFields := make([]cuessz.Field, 0)
	variableFields := make([]cuessz.Field, 0)
	for _, field := range def.Children {
		if isFixedSize(field.Def, ctx) {
			fixedFields = append(fixedFields, field)
		} else {
			variableFields = append(variableFields, field)
		}
	}

	// Calculate offsets for fixed fields
	fixedFieldOffsets := make([]uint64, len(fixedFields))
	var currentOffset uint64 = 0
	for i, field := range fixedFields {
		fixedFieldOffsets[i] = currentOffset
		size, _ := calculateFieldSize(field.Def, ctx)
		currentOffset += size
	}
	fixedFieldsSize := currentOffset

	// Total fixed size includes offsets for variable fields
	fixedSize := fixedFieldsSize + uint64(len(variableFields)*4)

	// Generate type documentation
	f.Comment(fmt.Sprintf("%s is an immutable progressive container backed by []byte", defName))
	f.Comment("")
	f.Comment(fmt.Sprintf("Active fields (merkle positions): %v", def.ActiveFields))
	f.Comment(fmt.Sprintf("This progressive container has %d active fields out of %d total positions", activeCount, len(def.ActiveFields)))
	f.Comment("")
	f.Comment(fmt.Sprintf("Use New%sBuilder() to construct new instances.", defName))
	f.Type().Id(defName).Op("[]").Byte()
	f.Line()

	// Generate NewFromSSZ
	generateProgressiveNewFromSSZ(f, defName, fixedSize, def.ActiveFields)

	// Generate Fixed() and SizeSSZ()
	f.Func().Params(jen.Id("c").Id(defName)).Id("Fixed").Params().Bool().Block(
		jen.Return(jen.False()),
	)
	f.Line()

	f.Func().Params(jen.Id("c").Id(defName)).Id("SizeSSZ").Params().Int().Block(
		jen.Return(jen.Len(jen.Id("c"))),
	)
	f.Line()

	// Generate getters for fixed fields
	for i, field := range fixedFields {
		offset := fixedFieldOffsets[i]
		size, _ := calculateFieldSize(field.Def, ctx)
		generateImmutableFixedGetter(f, defName, field, offset, size, ctx)
	}

	// Generate getters for variable fields
	for i, field := range variableFields {
		offsetPosition := fixedFieldsSize + uint64(i*4)
		generateImmutableVariableGetter(f, defName, field, offsetPosition, i, len(variableFields), fixedSize, ctx)
	}

	// Generate Builder
	generateProgressiveBuilder(f, defName, def, fixedFields, variableFields, fixedFieldOffsets,
		fixedFieldsSize, fixedSize, childToPosition, ctx)

	return nil
}

func generateProgressiveNewFromSSZ(f *jen.File, defName string, fixedSize uint64, activeFields []int) {
	f.Comment(fmt.Sprintf("New%sFromSSZ creates a %s from SSZ-encoded bytes", defName, defName))
	f.Func().Id(fmt.Sprintf("New%sFromSSZ", defName)).Params(jen.Id("data").Op("[]").Byte()).Id(defName).Block(
		jen.If(jen.Len(jen.Id("data")).Op("<").Lit(int(fixedSize))).Block(
			jen.Return(jen.Nil()),
		),
		jen.Return(jen.Id(defName).Call(jen.Id("data"))),
	)
	f.Line()
}

func generateProgressiveBuilder(f *jen.File, defName string, def cuessz.Def, fixedFields, variableFields []cuessz.Field,
	fixedFieldOffsets []uint64, fixedFieldsSize, fixedSize uint64, childToPosition map[int]int, ctx *GeneratorContext) {

	builderName := defName + "Builder"

	// Generate builder struct
	f.Comment(fmt.Sprintf("%s is a mutable builder for %s", builderName, defName))
	f.Comment(fmt.Sprintf("Progressive container with active fields: %v", def.ActiveFields))
	f.Type().Id(builderName).Struct(
		func() []jen.Code {
			fields := make([]jen.Code, 0)
			for _, field := range fixedFields {
				fieldName := toPrivateFieldName(field.Name)
				fieldType := getBuilderFieldType(field.Def, ctx)
				fields = append(fields, jen.Id(fieldName).Add(fieldType))
			}
			for _, field := range variableFields {
				fieldName := toPrivateFieldName(field.Name)
				fieldType := getBuilderFieldType(field.Def, ctx)
				fields = append(fields, jen.Id(fieldName).Add(fieldType))
			}
			return fields
		}()...,
	)
	f.Line()

	// Generate constructor
	f.Comment(fmt.Sprintf("New%s creates a new builder for %s", builderName, defName))
	f.Func().Id(fmt.Sprintf("New%s", builderName)).Params().Op("*").Id(builderName).Block(
		jen.Return(jen.Op("&").Id(builderName).Values()),
	)
	f.Line()

	// Generate setters for all fields
	for i, field := range fixedFields {
		generateBuilderSetter(f, builderName, field, fixedFieldOffsets[i], ctx)
	}
	for _, field := range variableFields {
		generateBuilderSetter(f, builderName, field, 0, ctx)
	}

	// Generate Build() method
	generateProgressiveBuilderBuild(f, builderName, defName, fixedFields, variableFields,
		fixedFieldOffsets, fixedFieldsSize, fixedSize, ctx)
}

func generateProgressiveBuilderBuild(f *jen.File, builderName, typeName string, fixedFields, variableFields []cuessz.Field,
	fixedFieldOffsets []uint64, fixedFieldsSize, fixedSize uint64, ctx *GeneratorContext) {

	f.Func().Params(jen.Id("b").Op("*").Id(builderName)).Id("Build").Params().Id(typeName).Block(
		generateProgressiveBuildLogic(builderName, typeName, fixedFields, variableFields, fixedFieldOffsets, fixedFieldsSize, fixedSize, ctx)...,
	)
	f.Line()
}

func generateProgressiveBuildLogic(builderName, typeName string, fixedFields, variableFields []cuessz.Field,
	fixedFieldOffsets []uint64, fixedFieldsSize, fixedSize uint64, ctx *GeneratorContext) []jen.Code {

	buildBlock := []jen.Code{
		jen.Comment("Calculate total size"),
		jen.Id("varSize").Op(":=").Lit(0),
	}

	// Calculate variable sizes
	for _, field := range variableFields {
		fieldName := toPrivateFieldName(field.Name)
		if field.Def.Type == cuessz.TypeBitList {
			buildBlock = append(buildBlock,
				jen.Id("varSize").Op("+=").Id("b").Dot(fieldName).Dot("SizeSSZ").Call(),
			)
		} else {
			buildBlock = append(buildBlock,
				jen.Id("varSize").Op("+=").Len(jen.Id("b").Dot(fieldName)),
			)
		}
	}

	buildBlock = append(buildBlock,
		jen.Line(),
		jen.Comment("Allocate result buffer"),
		jen.Id("result").Op(":=").Make(jen.Op("[]").Byte(), jen.Lit(int(fixedSize)).Op("+").Id("varSize")),
		jen.Line(),
	)

	// Write fixed fields
	buildBlock = append(buildBlock, jen.Comment("Write fixed fields"))
	for i, field := range fixedFields {
		offset := fixedFieldOffsets[i]
		size, _ := calculateFieldSize(field.Def, ctx)
		fieldName := toPrivateFieldName(field.Name)
		def := field.Def

		// Resolve refs
		if def.Type == cuessz.TypeRef {
			if refDef, ok := ctx.Defs[def.Ref]; ok {
				def = refDef
			}
		}

		switch def.Type {
		case cuessz.TypeBoolean:
			buildBlock = append(buildBlock,
				jen.If(jen.Id("b").Dot(fieldName)).Block(
					jen.Id("result").Index(jen.Lit(int(offset))).Op("=").Lit(1),
				),
			)
		case cuessz.TypeUint8:
			buildBlock = append(buildBlock,
				jen.Id("result").Index(jen.Lit(int(offset))).Op("=").Id("b").Dot(fieldName),
			)
		case cuessz.TypeUint16:
			buildBlock = append(buildBlock,
				jen.Qual("encoding/binary", "LittleEndian").Dot("PutUint16").Call(
					jen.Id("result").Index(jen.Lit(int(offset)).Op(":").Lit(int(offset+size))),
					jen.Id("b").Dot(fieldName),
				),
			)
		case cuessz.TypeUint32:
			buildBlock = append(buildBlock,
				jen.Qual("encoding/binary", "LittleEndian").Dot("PutUint32").Call(
					jen.Id("result").Index(jen.Lit(int(offset)).Op(":").Lit(int(offset+size))),
					jen.Id("b").Dot(fieldName),
				),
			)
		case cuessz.TypeUint64:
			buildBlock = append(buildBlock,
				jen.Qual("encoding/binary", "LittleEndian").Dot("PutUint64").Call(
					jen.Id("result").Index(jen.Lit(int(offset)).Op(":").Lit(int(offset+size))),
					jen.Id("b").Dot(fieldName),
				),
			)
		default:
			buildBlock = append(buildBlock,
				jen.Copy(jen.Id("result").Index(jen.Lit(int(offset)).Op(":").Lit(int(offset+size))), jen.Id("b").Dot(fieldName)),
			)
		}
	}

	buildBlock = append(buildBlock, jen.Line())

	// Write offsets and variable data
	if len(variableFields) > 0 {
		buildBlock = append(buildBlock, jen.Comment("Write offsets and variable data"))
		buildBlock = append(buildBlock, jen.Id("currentOffset").Op(":=").Uint32().Call(jen.Lit(int(fixedSize))))

		for i, field := range variableFields {
			offsetPosition := fixedFieldsSize + uint64(i*4)
			fieldName := toPrivateFieldName(field.Name)

			// Write offset
			buildBlock = append(buildBlock,
				jen.Qual("encoding/binary", "LittleEndian").Dot("PutUint32").Call(
					jen.Id("result").Index(jen.Lit(int(offsetPosition)).Op(":").Lit(int(offsetPosition+4))),
					jen.Id("currentOffset"),
				),
			)

			// Write data
			if field.Def.Type == cuessz.TypeBitList {
				buildBlock = append(buildBlock,
					jen.Id("data").Op(":=").Id("b").Dot(fieldName).Dot("SSZBytes").Call(),
					jen.Copy(jen.Id("result").Index(jen.Id("currentOffset").Op(":")), jen.Id("data")),
					jen.Id("currentOffset").Op("+=").Uint32().Call(jen.Len(jen.Id("data"))),
				)
			} else {
				buildBlock = append(buildBlock,
					jen.Copy(jen.Id("result").Index(jen.Id("currentOffset").Op(":")), jen.Id("b").Dot(fieldName)),
					jen.Id("currentOffset").Op("+=").Uint32().Call(jen.Len(jen.Id("b").Dot(fieldName))),
				)
			}
		}
	}

	buildBlock = append(buildBlock,
		jen.Line(),
		jen.Return(jen.Id(typeName).Call(jen.Id("result"))),
	)

	return buildBlock
}

// generateVariableContainerWithBuilder generates an immutable type backed by []byte
// and a separate Builder for construction
func generateVariableContainerWithBuilder(f *jen.File, defName string, def cuessz.Def, ctx *GeneratorContext) error {
	// Check if we need to import solidssz for bitlists
	hasBitlist := false
	for _, child := range def.Children {
		if child.Def.Type == cuessz.TypeBitList {
			hasBitlist = true
			break
		}
	}
	if hasBitlist {
		f.ImportName("github.com/gfx-labs/ssz/pkg/solidssz", "solidssz")
	}

	// Separate fixed and variable fields
	var fixedFields []cuessz.Field
	var variableFields []cuessz.Field

	for _, child := range def.Children {
		if isFixedSize(child.Def, ctx) {
			fixedFields = append(fixedFields, child)
		} else {
			variableFields = append(variableFields, child)
		}
	}

	// Calculate fixed portion size and offsets
	fixedFieldOffsets := make([]uint64, len(fixedFields))
	currentOffset := uint64(0)

	for i, field := range fixedFields {
		fixedFieldOffsets[i] = currentOffset
		size, err := calculateFieldSize(field.Def, ctx)
		if err != nil {
			return fmt.Errorf("failed to calculate size of fixed field %s: %w", field.Name, err)
		}
		currentOffset += size
	}

	fixedFieldsSize := currentOffset
	// Add 4 bytes per variable field for offsets
	fixedSize := fixedFieldsSize + uint64(len(variableFields)*4)

	// Generate immutable type as []byte
	f.Comment(fmt.Sprintf("%s is an immutable SSZ container backed by []byte", defName))
	f.Comment("")
	f.Comment("Use New" + defName + "Builder() to construct new instances.")
	f.Type().Id(defName).Op("[]").Byte()
	f.Line()

	// Generate NewFromSSZ
	generateImmutableNewFromSSZ(f, defName, fixedSize)

	// Generate Fixed() method
	f.Func().Params(jen.Id("c").Id(defName)).Id("Fixed").Params().Bool().Block(
		jen.Return(jen.False()),
	)
	f.Line()

	// Generate SizeSSZ() method
	f.Func().Params(jen.Id("c").Id(defName)).Id("SizeSSZ").Params().Int().Block(
		jen.Return(jen.Len(jen.Id("c"))),
	)
	f.Line()

	// Generate getters for fixed fields
	for i, field := range fixedFields {
		offset := fixedFieldOffsets[i]
		size, _ := calculateFieldSize(field.Def, ctx)
		generateImmutableFixedGetter(f, defName, field, offset, size, ctx)
	}

	// Generate getters for variable fields
	for i, field := range variableFields {
		offsetPosition := fixedFieldsSize + uint64(i*4)
		generateImmutableVariableGetter(f, defName, field, offsetPosition, i, len(variableFields), fixedSize, ctx)
	}

	// Generate Builder type
	generateBuilder(f, defName, fixedFields, variableFields, fixedFieldOffsets, fixedFieldsSize, fixedSize, ctx)

	return nil
}

func generateImmutableNewFromSSZ(f *jen.File, defName string, fixedSize uint64) {
	f.Comment(fmt.Sprintf("New%sFromSSZ creates a %s from SSZ-encoded bytes", defName, defName))
	f.Func().Id(fmt.Sprintf("New%sFromSSZ", defName)).Params(jen.Id("data").Op("[]").Byte()).Id(defName).Block(
		jen.If(jen.Len(jen.Id("data")).Op("<").Lit(int(fixedSize))).Block(
			jen.Return(jen.Nil()),
		),
		jen.Return(jen.Id(defName).Call(jen.Id("data"))),
	)
	f.Line()
}

func generateImmutableFixedGetter(f *jen.File, typeName string, field cuessz.Field, offset, size uint64, ctx *GeneratorContext) {
	methodName := snakeToCamel(field.Name)
	def := field.Def

	// Resolve refs
	if def.Type == cuessz.TypeRef {
		if refDef, ok := ctx.Defs[def.Ref]; ok {
			def = refDef
		}
	}

	var returnType *jen.Statement
	var bodyStatements []jen.Code

	switch def.Type {
	case cuessz.TypeBoolean:
		returnType = jen.Bool()
		bodyStatements = []jen.Code{
			jen.Return(jen.Id("c").Index(jen.Lit(int(offset))).Op("!=").Lit(0)),
		}
	case cuessz.TypeUint8:
		returnType = jen.Uint8()
		bodyStatements = []jen.Code{
			jen.Return(jen.Id("c").Index(jen.Lit(int(offset)))),
		}
	case cuessz.TypeUint16:
		returnType = jen.Uint16()
		bodyStatements = []jen.Code{
			jen.Return(jen.Qual("encoding/binary", "LittleEndian").Dot("Uint16").Call(
				jen.Id("c").Index(jen.Lit(int(offset)).Op(":").Lit(int(offset+size))),
			)),
		}
	case cuessz.TypeUint32:
		returnType = jen.Uint32()
		bodyStatements = []jen.Code{
			jen.Return(jen.Qual("encoding/binary", "LittleEndian").Dot("Uint32").Call(
				jen.Id("c").Index(jen.Lit(int(offset)).Op(":").Lit(int(offset+size))),
			)),
		}
	case cuessz.TypeUint64:
		returnType = jen.Uint64()
		bodyStatements = []jen.Code{
			jen.Return(jen.Qual("encoding/binary", "LittleEndian").Dot("Uint64").Call(
				jen.Id("c").Index(jen.Lit(int(offset)).Op(":").Lit(int(offset+size))),
			)),
		}
	default:
		returnType = jen.Index(jen.Lit(int(size))).Byte()
		bodyStatements = []jen.Code{
			jen.Var().Id("result").Index(jen.Lit(int(size))).Byte(),
			jen.Copy(jen.Id("result").Index(jen.Op(":")), jen.Id("c").Index(jen.Lit(int(offset)).Op(":").Lit(int(offset+size)))),
			jen.Return(jen.Id("result")),
		}
	}

	// Add documentation
	comment := fmt.Sprintf("%s returns the %s field", methodName, field.Name)
	if field.Description != nil && *field.Description != "" {
		comment = fmt.Sprintf("%s returns the %s field - %s", methodName, field.Name, *field.Description)
	}
	f.Comment(comment)
	f.Func().Params(jen.Id("c").Id(typeName)).Id(methodName).Params().Add(returnType).Block(bodyStatements...)
	f.Line()
}

func generateImmutableVariableGetter(f *jen.File, typeName string, field cuessz.Field, offsetPosition uint64, index, numVarFields int, fixedSize uint64, ctx *GeneratorContext) {
	methodName := snakeToCamel(field.Name)

	// Determine return type
	var returnType *jen.Statement
	if field.Def.Type == cuessz.TypeBitList {
		returnType = jen.Qual("github.com/gfx-labs/ssz/pkg/solidssz", "Bitlist")
	} else {
		returnType = jen.Op("[]").Byte()
	}

	// Add documentation
	comment := fmt.Sprintf("%s returns the %s field", methodName, field.Name)
	if field.Description != nil && *field.Description != "" {
		comment = fmt.Sprintf("%s returns the %s field - %s", methodName, field.Name, *field.Description)
	}
	f.Comment(comment)

	// Generate method body
	body := []jen.Code{
		jen.Comment("Read offset"),
		jen.Id("offset").Op(":=").Qual("encoding/binary", "LittleEndian").Dot("Uint32").Call(
			jen.Id("c").Index(jen.Lit(int(offsetPosition)).Op(":").Lit(int(offsetPosition+4))),
		),
	}

	// Determine end position
	if index == numVarFields-1 {
		// Last variable field - extends to end
		body = append(body,
			jen.Id("data").Op(":=").Id("c").Index(jen.Id("offset").Op(":")),
		)
	} else {
		// Not last - read next offset
		nextOffsetPos := offsetPosition + 4
		body = append(body,
			jen.Id("nextOffset").Op(":=").Qual("encoding/binary", "LittleEndian").Dot("Uint32").Call(
				jen.Id("c").Index(jen.Lit(int(nextOffsetPos)).Op(":").Lit(int(nextOffsetPos+4))),
			),
			jen.Id("data").Op(":=").Id("c").Index(jen.Id("offset").Op(":").Id("nextOffset")),
		)
	}

	// Return based on type
	if field.Def.Type == cuessz.TypeBitList {
		body = append(body,
			jen.Return(jen.Qual("github.com/gfx-labs/ssz/pkg/solidssz", "NewBitlistFromSSZ").Call(jen.Id("data"))),
		)
	} else {
		body = append(body,
			jen.Return(jen.Id("data")),
		)
	}

	f.Func().Params(jen.Id("c").Id(typeName)).Id(methodName).Params().Add(returnType).Block(body...)
	f.Line()
}

func generateBuilder(f *jen.File, typeName string, fixedFields, variableFields []cuessz.Field, fixedFieldOffsets []uint64, fixedFieldsSize, fixedSize uint64, ctx *GeneratorContext) {
	builderName := typeName + "Builder"

	// Generate builder struct
	f.Comment(fmt.Sprintf("%s is a mutable builder for %s", builderName, typeName))
	f.Type().Id(builderName).Struct(
		func() []jen.Code {
			fields := make([]jen.Code, 0)
			for _, field := range fixedFields {
				fieldName := toPrivateFieldName(field.Name)
				fieldType := getBuilderFieldType(field.Def, ctx)
				fields = append(fields, jen.Id(fieldName).Add(fieldType))
			}
			for _, field := range variableFields {
				fieldName := toPrivateFieldName(field.Name)
				fieldType := getBuilderFieldType(field.Def, ctx)
				fields = append(fields, jen.Id(fieldName).Add(fieldType))
			}
			return fields
		}()...,
	)
	f.Line()

	// Generate constructor
	f.Comment(fmt.Sprintf("New%s creates a new builder for %s", builderName, typeName))
	f.Func().Id(fmt.Sprintf("New%s", builderName)).Params().Op("*").Id(builderName).Block(
		jen.Return(jen.Op("&").Id(builderName).Values()),
	)
	f.Line()

	// Generate setters for all fields
	for i, field := range fixedFields {
		generateBuilderSetter(f, builderName, field, fixedFieldOffsets[i], ctx)
	}
	for _, field := range variableFields {
		generateBuilderSetter(f, builderName, field, 0, ctx)
	}

	// Generate Build() method
	generateBuilderBuild(f, builderName, typeName, fixedFields, variableFields, fixedFieldOffsets, fixedFieldsSize, fixedSize, ctx)
}

func getBuilderFieldType(def cuessz.Def, ctx *GeneratorContext) *jen.Statement {
	// Resolve refs
	if def.Type == cuessz.TypeRef {
		if refDef, ok := ctx.Defs[def.Ref]; ok {
			def = refDef
		}
	}

	switch def.Type {
	case cuessz.TypeBoolean:
		return jen.Bool()
	case cuessz.TypeUint8:
		return jen.Uint8()
	case cuessz.TypeUint16:
		return jen.Uint16()
	case cuessz.TypeUint32:
		return jen.Uint32()
	case cuessz.TypeUint64:
		return jen.Uint64()
	case cuessz.TypeBitList:
		return jen.Qual("github.com/gfx-labs/ssz/pkg/solidssz", "Bitlist")
	default:
		return jen.Op("[]").Byte()
	}
}

func generateBuilderSetter(f *jen.File, builderName string, field cuessz.Field, offset uint64, ctx *GeneratorContext) {
	methodName := "Set" + snakeToCamel(field.Name)
	fieldName := toPrivateFieldName(field.Name)
	fieldType := getBuilderFieldType(field.Def, ctx)

	comment := fmt.Sprintf("%s sets the %s field", methodName, field.Name)
	if field.Description != nil && *field.Description != "" {
		comment = fmt.Sprintf("%s sets the %s field - %s", methodName, field.Name, *field.Description)
	}
	f.Comment(comment)

	f.Func().Params(jen.Id("b").Op("*").Id(builderName)).Id(methodName).Params(
		jen.Id("v").Add(fieldType),
	).Op("*").Id(builderName).Block(
		jen.Id("b").Dot(fieldName).Op("=").Id("v"),
		jen.Return(jen.Id("b")),
	)
	f.Line()
}

func generateBuilderBuild(f *jen.File, builderName, typeName string, fixedFields, variableFields []cuessz.Field, fixedFieldOffsets []uint64, fixedFieldsSize, fixedSize uint64, ctx *GeneratorContext) {
	f.Comment(fmt.Sprintf("Build constructs an immutable %s from the builder", typeName))

	buildBlock := []jen.Code{
		jen.Comment("Calculate total size"),
		jen.Id("varSize").Op(":=").Lit(0),
	}

	// Calculate variable sizes
	for _, field := range variableFields {
		fieldName := toPrivateFieldName(field.Name)
		if field.Def.Type == cuessz.TypeBitList {
			buildBlock = append(buildBlock,
				jen.If(jen.Id("b").Dot(fieldName).Op("!=").Nil()).Block(
					jen.Id("varSize").Op("+=").Id("b").Dot(fieldName).Dot("SizeSSZ").Call(),
				),
			)
		} else {
			buildBlock = append(buildBlock,
				jen.Id("varSize").Op("+=").Len(jen.Id("b").Dot(fieldName)),
			)
		}
	}

	buildBlock = append(buildBlock,
		jen.Line(),
		jen.Comment("Allocate result buffer"),
		jen.Id("result").Op(":=").Make(jen.Op("[]").Byte(), jen.Lit(int(fixedSize)).Op("+").Id("varSize")),
		jen.Line(),
	)

	// Write fixed fields
	buildBlock = append(buildBlock, jen.Comment("Write fixed fields"))
	for i, field := range fixedFields {
		offset := fixedFieldOffsets[i]
		size, _ := calculateFieldSize(field.Def, ctx)
		fieldName := toPrivateFieldName(field.Name)
		def := field.Def

		// Resolve refs
		if def.Type == cuessz.TypeRef {
			if refDef, ok := ctx.Defs[def.Ref]; ok {
				def = refDef
			}
		}

		switch def.Type {
		case cuessz.TypeBoolean:
			buildBlock = append(buildBlock,
				jen.If(jen.Id("b").Dot(fieldName)).Block(
					jen.Id("result").Index(jen.Lit(int(offset))).Op("=").Lit(1),
				),
			)
		case cuessz.TypeUint8:
			buildBlock = append(buildBlock,
				jen.Id("result").Index(jen.Lit(int(offset))).Op("=").Id("b").Dot(fieldName),
			)
		case cuessz.TypeUint16:
			buildBlock = append(buildBlock,
				jen.Qual("encoding/binary", "LittleEndian").Dot("PutUint16").Call(
					jen.Id("result").Index(jen.Lit(int(offset)).Op(":").Lit(int(offset+size))),
					jen.Id("b").Dot(fieldName),
				),
			)
		case cuessz.TypeUint32:
			buildBlock = append(buildBlock,
				jen.Qual("encoding/binary", "LittleEndian").Dot("PutUint32").Call(
					jen.Id("result").Index(jen.Lit(int(offset)).Op(":").Lit(int(offset+size))),
					jen.Id("b").Dot(fieldName),
				),
			)
		case cuessz.TypeUint64:
			buildBlock = append(buildBlock,
				jen.Qual("encoding/binary", "LittleEndian").Dot("PutUint64").Call(
					jen.Id("result").Index(jen.Lit(int(offset)).Op(":").Lit(int(offset+size))),
					jen.Id("b").Dot(fieldName),
				),
			)
		default:
			buildBlock = append(buildBlock,
				jen.Copy(jen.Id("result").Index(jen.Lit(int(offset)).Op(":").Lit(int(offset+size))), jen.Id("b").Dot(fieldName)),
			)
		}
	}

	buildBlock = append(buildBlock, jen.Line())

	// Write offsets and variable data
	if len(variableFields) > 0 {
		buildBlock = append(buildBlock, jen.Comment("Write offsets and variable data"))
		buildBlock = append(buildBlock, jen.Id("currentOffset").Op(":=").Uint32().Call(jen.Lit(int(fixedSize))))

		for i, field := range variableFields {
			offsetPosition := fixedFieldsSize + uint64(i*4)
			fieldName := toPrivateFieldName(field.Name)

			// Write offset
			buildBlock = append(buildBlock,
				jen.Qual("encoding/binary", "LittleEndian").Dot("PutUint32").Call(
					jen.Id("result").Index(jen.Lit(int(offsetPosition)).Op(":").Lit(int(offsetPosition+4))),
					jen.Id("currentOffset"),
				),
			)

			// Write data
			if field.Def.Type == cuessz.TypeBitList {
				buildBlock = append(buildBlock,
					jen.If(jen.Id("b").Dot(fieldName).Op("!=").Nil()).Block(
						jen.Id("data").Op(":=").Id("b").Dot(fieldName).Dot("SSZBytes").Call(),
						jen.Copy(jen.Id("result").Index(jen.Id("currentOffset").Op(":")), jen.Id("data")),
						jen.Id("currentOffset").Op("+=").Uint32().Call(jen.Len(jen.Id("data"))),
					),
				)
			} else {
				buildBlock = append(buildBlock,
					jen.Copy(jen.Id("result").Index(jen.Id("currentOffset").Op(":")), jen.Id("b").Dot(fieldName)),
					jen.Id("currentOffset").Op("+=").Uint32().Call(jen.Len(jen.Id("b").Dot(fieldName))),
				)
			}
		}
	}

	buildBlock = append(buildBlock,
		jen.Line(),
		jen.Return(jen.Id(typeName).Call(jen.Id("result"))),
	)

	f.Func().Params(jen.Id("b").Op("*").Id(builderName)).Id("Build").Params().Id(typeName).Block(buildBlock...)
	f.Line()
}
