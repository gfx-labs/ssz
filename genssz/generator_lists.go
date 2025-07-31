package genssz

import (
	"fmt"
	
	"github.com/dave/jennifer/jen"
	"github.com/gfx-labs/ssz"
)

// GenerateListCode generates Go code for list types from a World and Schema
func GenerateListCode(world *World, schema *Schema, f *jen.File) error {
	// Generate code for each type in the world
	for _, structDef := range schema.Structs {
		// Convert to ssz.Field
		sszField := structDef.ToSSZField()
		
		// Check if this is a variable-size type (list)
		refs := make(map[string]ssz.Field)
		for _, s := range schema.Structs {
			refs[s.Name] = s.ToSSZField()
		}
		isVar, err := sszField.IsVariable(refs)
		isFixed := !isVar
		if err != nil {
			return fmt.Errorf("failed to check if %s is fixed size: %w", structDef.Name, err)
		}
		
		// Skip fixed-size types (they're handled by the regular generator)
		if isFixed {
			continue
		}
		
		// Only generate code for pure list types, not containers with list fields
		if sszField.Type != ssz.TypeList {
			continue
		}
		
		// Generate the list struct
		if err := generateListStruct(f, sszField, schema); err != nil {
			return fmt.Errorf("failed to generate list struct for %s: %w", structDef.Name, err)
		}
		
		// Generate list methods
		if err := generateListMethods(f, sszField, schema); err != nil {
			return fmt.Errorf("failed to generate list methods for %s: %w", structDef.Name, err)
		}
	}
	
	return nil
}

// generateListStruct generates the struct definition for a list type
func generateListStruct(f *jen.File, structDef ssz.Field, schema *Schema) error {
	typeName := structDef.Name
	
	// For a pure list type (like type MyList []ElementType with a limit)
	if structDef.Type == ssz.TypeList {
		if len(structDef.Children) == 0 {
			return fmt.Errorf("list %s has no element type", structDef.Name)
		}
		
		elemType := structDef.Children[0]
		elemGoType := getElementGoType(elemType, schema)
		
		// Check if element type is fixed-size
		refs := make(map[string]ssz.Field)
		for _, s := range schema.Structs {
			refs[s.Name] = s.ToSSZField()
		}
		
		isElemFixed := false
		elemSize := 0
		if elemType.Type == ssz.TypeRef {
			// Check if the referenced type is fixed-size
			if refField, ok := refs[elemType.Ref]; ok {
				isVar, err := refField.IsVariable(refs)
				if err == nil && !isVar {
					isElemFixed = true
					elemSize, _ = getFieldSize(refField, refs)
				}
			}
		} else {
			// Basic types are fixed-size
			isVar, _ := elemType.IsVariable(refs)
			if !isVar {
				isElemFixed = true
				elemSize, _ = getFieldSize(elemType, refs)
			}
		}
		
		// Generate custom struct with elements field
		// For fixed-size elements, use []byte buffer; for variable-size, use []ElementType
		if isElemFixed && elemSize > 0 {
			f.Comment(fmt.Sprintf("%s is an SSZ list type with the following structure:", typeName))
			f.Comment("")
			f.Comment("List properties:")
			f.Comment(fmt.Sprintf("- Element type: %s", getTypeDescription(elemType)))
			f.Comment(fmt.Sprintf("- Element size: %d bytes (fixed)", elemSize))
			f.Comment(fmt.Sprintf("- Maximum elements: %d", structDef.Limit))
			f.Comment(fmt.Sprintf("- Maximum buffer size: %d bytes", structDef.Limit * uint64(elemSize)))
			f.Comment("")
			f.Comment("Storage layout:")
			f.Comment("- buffer: []byte - Stores elements contiguously")
			f.Comment("- count: int - Current number of elements")
			f.Comment("")
			f.Comment("Element access:")
			f.Comment(fmt.Sprintf("- Element N starts at: buffer[N * %d]", elemSize))
			f.Comment(fmt.Sprintf("- Element N ends at: buffer[(N+1) * %d - 1]", elemSize))
			f.Type().Id(typeName).Struct(
				jen.Id("buffer").Op("[]").Byte(),
				jen.Id("count").Int(),
			)
		} else {
			f.Comment(fmt.Sprintf("%s is an SSZ list type with the following structure:", typeName))
			f.Comment("")
			f.Comment("List properties:")
			f.Comment(fmt.Sprintf("- Element type: %s (variable size)", getTypeDescription(elemType)))
			f.Comment(fmt.Sprintf("- Maximum elements: %d", structDef.Limit))
			f.Comment("")
			f.Comment("Storage:")
			f.Comment("- elements: Direct slice of element type")
			// Special case: for []uint8, use []byte
			var elemFieldType jen.Code
			if elemType.Type == ssz.TypeUint8 {
				elemFieldType = jen.Op("[]").Byte()
			} else {
				elemFieldType = jen.Op("[]").Add(elemGoType)
			}
			f.Type().Id(typeName).Struct(
				jen.Id("elements").Add(elemFieldType),
			)
		}
		f.Line()
		
		return nil
	}
	
	// For containers with list fields
	// Generate struct comment
	f.Comment(fmt.Sprintf("%s is an SSZ container with variable-size fields", typeName))
	
	// Build struct fields
	var fields []jen.Code
	for _, field := range structDef.Children {
		fieldName := capitalizeFirst(field.Name)
		
		switch field.Type {
		case ssz.TypeList:
			if len(field.Children) == 0 {
				return fmt.Errorf("list field %s has no element type", field.Name)
			}
			elemType := field.Children[0]
			elemGoType := getElementGoType(elemType, schema)
			fields = append(fields, jen.Id(fieldName).Op("[]").Add(elemGoType))
			
		case ssz.TypeUint8:
			fields = append(fields, jen.Id(fieldName).Uint8())
		case ssz.TypeUint16:
			fields = append(fields, jen.Id(fieldName).Uint16())
		case ssz.TypeUint32:
			fields = append(fields, jen.Id(fieldName).Uint32())
		case ssz.TypeUint64:
			fields = append(fields, jen.Id(fieldName).Uint64())
		case ssz.TypeBoolean:
			fields = append(fields, jen.Id(fieldName).Bool())
		case ssz.TypeRef:
			fields = append(fields, jen.Id(fieldName).Id(field.Ref))
		case ssz.TypeVector:
			// Handle fixed-size vectors in variable containers
			if len(field.Children) > 0 {
				elemType := field.Children[0]
				size := int(field.Size)
				switch elemType.Type {
				case ssz.TypeUint8:
					fields = append(fields, jen.Id(fieldName).Op("[").Lit(size).Op("]").Byte())
				case ssz.TypeUint16:
					fields = append(fields, jen.Id(fieldName).Op("[").Lit(size).Op("]").Uint16())
				case ssz.TypeUint32:
					fields = append(fields, jen.Id(fieldName).Op("[").Lit(size).Op("]").Uint32())
				case ssz.TypeUint64:
					fields = append(fields, jen.Id(fieldName).Op("[").Lit(size).Op("]").Uint64())
				case ssz.TypeRef:
					fields = append(fields, jen.Id(fieldName).Op("[").Lit(size).Op("]").Id(elemType.Ref))
				}
			}
		}
	}
	
	f.Type().Id(typeName).Struct(fields...)
	f.Line()
	
	return nil
}

// generateListMethods generates methods for a list type
func generateListMethods(f *jen.File, structDef ssz.Field, schema *Schema) error {
	typeName := structDef.Name
	
	// For a pure list type
	if structDef.Type == ssz.TypeList {
		if len(structDef.Children) == 0 {
			return fmt.Errorf("list %s has no element type", structDef.Name)
		}
		
		elemType := structDef.Children[0]
		elemGoType := getElementGoType(elemType, schema)
		
		// Check if element type is fixed-size
		refs := make(map[string]ssz.Field)
		for _, s := range schema.Structs {
			refs[s.Name] = s.ToSSZField()
		}
		
		isElemFixed := false
		elemSize := 0
		if elemType.Type == ssz.TypeRef {
			// Check if the referenced type is fixed-size
			if refField, ok := refs[elemType.Ref]; ok {
				isVar, err := refField.IsVariable(refs)
				if err == nil && !isVar {
					isElemFixed = true
					elemSize, _ = getFieldSize(refField, refs)
				}
			}
		} else {
			// Basic types are fixed-size
			isVar, _ := elemType.IsVariable(refs)
			if !isVar {
				isElemFixed = true
				elemSize, _ = getFieldSize(elemType, refs)
			}
		}
		
		// Determine elemFieldType for the list element storage
		var elemFieldType jen.Code
		if isElemFixed && elemSize > 0 {
			// For fixed-size elements, we don't need elemFieldType in constructors
			elemFieldType = nil
		} else {
			// For variable-size elements, determine the field type
			if elemType.Type == ssz.TypeUint8 {
				elemFieldType = jen.Op("[]").Byte()
			} else {
				elemFieldType = jen.Op("[]").Add(elemGoType)
			}
		}
		
		// Generate constructors and methods for custom struct
		generateListConstructors(f, typeName, elemGoType, elemFieldType, structDef, isElemFixed, elemSize)
		generateListMethodsCustom(f, typeName, elemGoType, elemFieldType, structDef, isElemFixed, elemSize)
		return nil
	}
	
	// For containers with list fields
	// Generate constructor
	f.Comment(fmt.Sprintf("New%s creates a new %s", typeName, typeName))
	f.Func().Id("New" + typeName).Params().Op("*").Id(typeName).Block(
		jen.Return(jen.Op("&").Id(typeName).Op("{}")),
	)
	f.Line()
	
	// Generate methods for each list field
	for _, field := range structDef.Children {
		if field.Type == ssz.TypeList {
			fieldName := capitalizeFirst(field.Name)
			methodBase := capitalizeFirst(field.Name)
			
			// Limit method for this field
			f.Comment(fmt.Sprintf("%sLimit returns the maximum number of elements the %s field can contain", methodBase, field.Name))
			f.Func().Params(jen.Id("c").Op("*").Id(typeName)).Id(methodBase + "Limit").Params().Uint64().Block(
				jen.Return(jen.Lit(field.Limit)),
			)
			f.Line()
			
			// Len method for this field
			f.Comment(fmt.Sprintf("%sLen returns the current number of elements in the %s field", methodBase, field.Name))
			f.Func().Params(jen.Id("c").Op("*").Id(typeName)).Id(methodBase + "Len").Params().Int().Block(
				jen.Return(jen.Len(jen.Id("c").Dot(fieldName))),
			)
			f.Line()
		}
	}
	
	// Fixed method (always returns false for containers with lists)
	f.Comment("Fixed returns false as this container has variable-size fields")
	f.Func().Params(jen.Id("c").Op("*").Id(typeName)).Id("Fixed").Params().Bool().Block(
		jen.Return(jen.False()),
	)
	f.Line()
	
	return nil
}

// generateListConstructors generates constructors for custom list struct
func generateListConstructors(f *jen.File, typeName string, elemGoType jen.Code, elemFieldType jen.Code, structDef ssz.Field, isElemFixed bool, elemSize int) {
	if isElemFixed && elemSize > 0 {
		// Basic constructor for fixed-size elements
		f.Comment(fmt.Sprintf("New%s creates a new %s", typeName, typeName))
		f.Func().Id("New" + typeName).Params().Op("*").Id(typeName).Block(
			jen.Return(jen.Op("&").Id(typeName).Op("{").
				Id("buffer").Op(":").Make(jen.Op("[]").Byte(), jen.Lit(0)).Op(",").
				Id("count").Op(":").Lit(0).
			Op("}")),
		)
		f.Line()
		
		// Constructor with initial capacity for fixed-size elements
		f.Comment(fmt.Sprintf("New%sWithCapacity creates a new %s with the specified capacity", typeName, typeName))
		f.Func().Id("New" + typeName + "WithCapacity").Params(jen.Id("capacity").Int()).Op("*").Id(typeName).Block(
			jen.Return(jen.Op("&").Id(typeName).Op("{").
				Id("buffer").Op(":").Make(jen.Op("[]").Byte(), jen.Lit(0), jen.Id("capacity").Op("*").Lit(elemSize)).Op(",").
				Id("count").Op(":").Lit(0).
			Op("}")),
		)
		f.Line()
	} else {
		// Basic constructor for variable-size elements
		f.Comment(fmt.Sprintf("New%s creates a new %s", typeName, typeName))
		f.Func().Id("New" + typeName).Params().Op("*").Id(typeName).Block(
			jen.Return(jen.Op("&").Id(typeName).Op("{").Id("elements").Op(":").Make(elemFieldType, jen.Lit(0)).Op("}")),
		)
		f.Line()
		
		// Constructor with initial capacity for variable-size elements
		f.Comment(fmt.Sprintf("New%sWithCapacity creates a new %s with the specified capacity", typeName, typeName))
		f.Func().Id("New" + typeName + "WithCapacity").Params(jen.Id("capacity").Int()).Op("*").Id(typeName).Block(
			jen.Return(jen.Op("&").Id(typeName).Op("{").Id("elements").Op(":").Make(elemFieldType, jen.Lit(0), jen.Id("capacity")).Op("}")),
		)
		f.Line()
	}
}

// generateListMethodsCustom generates all methods for custom list struct
func generateListMethodsCustom(f *jen.File, typeName string, elemGoType jen.Code, elemFieldType jen.Code, structDef ssz.Field, isElemFixed bool, elemSize int) {
	// Limit method
	f.Comment(fmt.Sprintf("Limit returns the maximum number of elements this list can contain"))
	f.Func().Params(jen.Id("l").Op("*").Id(typeName)).Id("Limit").Params().Uint64().Block(
		jen.Return(jen.Uint64().Call(jen.Lit(structDef.Limit))),
	)
	f.Line()
	
	// Len method
	f.Comment(fmt.Sprintf("Len returns the current number of elements in the list"))
	f.Func().Params(jen.Id("l").Op("*").Id(typeName)).Id("Len").Params().Int().Block(
		func() jen.Code {
			if isElemFixed && elemSize > 0 {
				return jen.Return(jen.Id("l").Dot("count"))
			} else {
				return jen.Return(jen.Len(jen.Id("l").Dot("elements")))
			}
		}(),
	)
	f.Line()
	
	// Get method - different implementation for fixed vs variable elements
	if isElemFixed && elemSize > 0 {
		// For fixed-size elements stored as []byte buffer
		f.Comment(fmt.Sprintf("Get returns the element at the given index"))
		f.Comment(fmt.Sprintf("Panics if index is out of bounds"))
		f.Func().Params(jen.Id("l").Op("*").Id(typeName)).Id("Get").Params(jen.Id("index").Int()).Add(elemGoType).Block(
			jen.If(jen.Id("index").Op(">=").Id("l").Dot("count")).Block(
				jen.Panic(jen.Lit("index out of bounds")),
			),
			jen.Id("start").Op(":=").Id("index").Op("*").Lit(elemSize),
			jen.Id("end").Op(":=").Id("start").Op("+").Lit(elemSize),
			jen.Return(jen.Add(elemGoType).Call(jen.Id("l").Dot("buffer").Op("[").Id("start").Op(":").Id("end").Op("]"))),
		)
		f.Line()
		
		// Set method for fixed-size elements
		f.Comment(fmt.Sprintf("Set updates the element at the given index"))
		f.Comment(fmt.Sprintf("Panics if index is out of bounds"))
		f.Func().Params(jen.Id("l").Op("*").Id(typeName)).Id("Set").Params(jen.Id("index").Int(), jen.Id("value").Add(elemGoType)).Block(
			jen.If(jen.Id("index").Op(">=").Id("l").Dot("count")).Block(
				jen.Panic(jen.Lit("index out of bounds")),
			),
			jen.Id("start").Op(":=").Id("index").Op("*").Lit(elemSize),
			jen.Copy(jen.Id("l").Dot("buffer").Op("[").Id("start").Op(":").Id("start").Op("+").Lit(elemSize).Op("]"), jen.Op("[]").Byte().Call(jen.Id("value"))),
		)
		f.Line()
		
		// Append method for fixed-size elements
		f.Comment(fmt.Sprintf("Append adds an element to the end of the list"))
		f.Comment(fmt.Sprintf("Returns an error if the list would exceed its limit"))
		f.Func().Params(jen.Id("l").Op("*").Id(typeName)).Id("Append").Params(jen.Id("value").Add(elemGoType)).Error().Block(
			jen.If(jen.Uint64().Call(jen.Id("l").Dot("count")).Op(">=").Id("l").Dot("Limit").Call()).Block(
				jen.Return(jen.Qual("fmt", "Errorf").Call(
					jen.Lit("list exceeds limit of %d elements"),
					jen.Id("l").Dot("Limit").Call(),
				)),
			),
			jen.Comment("Grow buffer if needed"),
			jen.Id("requiredSize").Op(":=").Parens(jen.Id("l").Dot("count").Op("+").Lit(1)).Op("*").Lit(elemSize),
			jen.If(jen.Len(jen.Id("l").Dot("buffer")).Op("<").Id("requiredSize")).Block(
				jen.Id("newBuffer").Op(":=").Make(jen.Op("[]").Byte(), jen.Id("requiredSize"), jen.Id("requiredSize").Op("*").Lit(2)),
				jen.Copy(jen.Id("newBuffer"), jen.Id("l").Dot("buffer")),
				jen.Id("l").Dot("buffer").Op("=").Id("newBuffer"),
			),
			jen.Comment("Append the element"),
			jen.Id("start").Op(":=").Id("l").Dot("count").Op("*").Lit(elemSize),
			jen.Copy(jen.Id("l").Dot("buffer").Op("[").Id("start").Op(":").Id("start").Op("+").Lit(elemSize).Op("]"), jen.Op("[]").Byte().Call(jen.Id("value"))),
			jen.Id("l").Dot("count").Op("++"),
			jen.Return(jen.Nil()),
		)
		f.Line()
	} else {
		// For variable-size elements, use direct access
		f.Comment(fmt.Sprintf("Get returns the element at the given index"))
		f.Comment(fmt.Sprintf("Panics if index is out of bounds"))
		f.Func().Params(jen.Id("l").Op("*").Id(typeName)).Id("Get").Params(jen.Id("index").Int()).Add(elemGoType).Block(
			jen.Return(jen.Id("l").Dot("elements").Op("[").Id("index").Op("]")),
		)
		f.Line()
		
		// Set method for variable-size elements
		f.Comment(fmt.Sprintf("Set updates the element at the given index"))
		f.Comment(fmt.Sprintf("Panics if index is out of bounds"))
		f.Func().Params(jen.Id("l").Op("*").Id(typeName)).Id("Set").Params(jen.Id("index").Int(), jen.Id("value").Add(elemGoType)).Block(
			jen.Id("l").Dot("elements").Op("[").Id("index").Op("]").Op("=").Id("value"),
		)
		f.Line()
		
		// Append method for variable-size elements
		f.Comment(fmt.Sprintf("Append adds an element to the end of the list"))
		f.Comment(fmt.Sprintf("Returns an error if the list would exceed its limit"))
		f.Func().Params(jen.Id("l").Op("*").Id(typeName)).Id("Append").Params(jen.Id("value").Add(elemGoType)).Error().Block(
			jen.If(jen.Uint64().Call(jen.Len(jen.Id("l").Dot("elements"))).Op(">=").Id("l").Dot("Limit").Call()).Block(
				jen.Return(jen.Qual("fmt", "Errorf").Call(
					jen.Lit("list exceeds limit of %d elements"),
					jen.Id("l").Dot("Limit").Call(),
				)),
			),
			jen.Id("l").Dot("elements").Op("=").Append(jen.Id("l").Dot("elements"), jen.Id("value")),
			jen.Return(jen.Nil()),
		)
		f.Line()
	}
	
	// Clear method
	f.Comment(fmt.Sprintf("Clear removes all elements from the list"))
	f.Func().Params(jen.Id("l").Op("*").Id(typeName)).Id("Clear").Params().Block(
		func() jen.Code {
			if isElemFixed && elemSize > 0 {
				return jen.Block(
					jen.Id("l").Dot("buffer").Op("=").Id("l").Dot("buffer").Op("[:0]"),
					jen.Id("l").Dot("count").Op("=").Lit(0),
				)
			} else {
				return jen.Id("l").Dot("elements").Op("=").Id("l").Dot("elements").Op("[:0]")
			}
		}(),
	)
	f.Line()
	
	// Resize method - different implementation for fixed vs variable elements
	f.Comment(fmt.Sprintf("Resize changes the size of the list, truncating or extending with zero values"))
	f.Comment(fmt.Sprintf("Returns an error if the new size would exceed the list limit"))
	f.Func().Params(jen.Id("l").Op("*").Id(typeName)).Id("Resize").Params(jen.Id("newSize").Int()).Error().Block(
		jen.If(jen.Uint64().Call(jen.Id("newSize")).Op(">").Id("l").Dot("Limit").Call()).Block(
			jen.Return(jen.Qual("fmt", "Errorf").Call(
				jen.Lit("new size %d exceeds limit %d"),
				jen.Id("newSize"),
				jen.Id("l").Dot("Limit").Call(),
			)),
		),
		jen.Line(),
		func() jen.Code {
			if isElemFixed && elemSize > 0 {
				return jen.Block(
					jen.Id("currentSize").Op(":=").Id("l").Dot("count"),
					jen.If(jen.Id("newSize").Op(">").Id("currentSize")).Block(
						jen.Comment("Extend the buffer if needed"),
						jen.Id("requiredSize").Op(":=").Id("newSize").Op("*").Lit(elemSize),
						jen.If(jen.Len(jen.Id("l").Dot("buffer")).Op("<").Id("requiredSize")).Block(
							jen.Id("newBuffer").Op(":=").Make(jen.Op("[]").Byte(), jen.Id("requiredSize")),
							jen.Copy(jen.Id("newBuffer"), jen.Id("l").Dot("buffer").Op("[:").Id("currentSize").Op("*").Lit(elemSize).Op("]")),
							jen.Id("l").Dot("buffer").Op("=").Id("newBuffer"),
						),
						jen.Comment("Zero out new elements"),
						jen.Id("start").Op(":=").Id("currentSize").Op("*").Lit(elemSize),
						jen.Id("end").Op(":=").Id("newSize").Op("*").Lit(elemSize),
						jen.For(jen.Id("i").Op(":=").Id("start"), jen.Id("i").Op("<").Id("end"), jen.Id("i").Op("++")).Block(
							jen.Id("l").Dot("buffer").Op("[").Id("i").Op("]").Op("=").Lit(0),
						),
					).Else().If(jen.Id("newSize").Op("<").Id("currentSize")).Block(
						jen.Comment("Truncate the buffer"),
						jen.Id("l").Dot("buffer").Op("=").Id("l").Dot("buffer").Op("[:").Id("newSize").Op("*").Lit(elemSize).Op("]"),
					),
					jen.Id("l").Dot("count").Op("=").Id("newSize"),
				)
			} else {
				return jen.Block(
					jen.Id("currentSize").Op(":=").Len(jen.Id("l").Dot("elements")),
					jen.If(jen.Id("newSize").Op(">").Id("currentSize")).Block(
						jen.Comment("Extend the list with zero values"),
						jen.Id("toAdd").Op(":=").Id("newSize").Op("-").Id("currentSize"),
						jen.For(jen.Id("i").Op(":=").Lit(0), jen.Id("i").Op("<").Id("toAdd"), jen.Id("i").Op("++")).Block(
							func() jen.Code {
								// For byte slices, append a zero byte
								if structDef.Children[0].Type == ssz.TypeUint8 {
									return jen.Id("l").Dot("elements").Op("=").Append(jen.Id("l").Dot("elements"), jen.Lit(0))
								} else {
									return jen.Id("l").Dot("elements").Op("=").Append(jen.Id("l").Dot("elements"), jen.Var().Add(elemGoType))
								}
							}(),
						),
					).Else().If(jen.Id("newSize").Op("<").Id("currentSize")).Block(
						jen.Comment("Truncate the list"),
						jen.Id("l").Dot("elements").Op("=").Id("l").Dot("elements").Op("[:").Id("newSize").Op("]"),
					),
				)
			}
		}(),
		jen.Return(jen.Nil()),
	)
	f.Line()
	
	// Elements method - different return type for fixed vs variable elements
	if isElemFixed && elemSize > 0 {
		// For fixed-size elements, return the raw buffer
		f.Comment(fmt.Sprintf("Buffer returns the underlying byte buffer"))
		f.Func().Params(jen.Id("l").Op("*").Id(typeName)).Id("Buffer").Params().Op("[]").Byte().Block(
			jen.Return(jen.Id("l").Dot("buffer")),
		)
		f.Line()
	} else {
		// For variable-size elements, return the typed slice
		f.Comment(fmt.Sprintf("Elements returns the underlying slice of elements"))
		f.Func().Params(jen.Id("l").Op("*").Id(typeName)).Id("Elements").Params().Add(elemFieldType).Block(
			jen.Return(jen.Id("l").Dot("elements")),
		)
		f.Line()
	}
	
	// Fixed method
	f.Comment(fmt.Sprintf("Fixed returns false as lists are variable-size"))
	f.Func().Params(jen.Id("l").Op("*").Id(typeName)).Id("Fixed").Params().Bool().Block(
		jen.Return(jen.False()),
	)
	f.Line()
}

// getElementGoType returns the Go type for a list element
func getElementGoType(elemType ssz.Field, schema *Schema) jen.Code {
	switch elemType.Type {
	case ssz.TypeUint8:
		return jen.Uint8()
	case ssz.TypeUint16:
		return jen.Uint16()
	case ssz.TypeUint32:
		return jen.Uint32()
	case ssz.TypeUint64:
		return jen.Uint64()
	case ssz.TypeBoolean:
		return jen.Bool()
	case ssz.TypeRef:
		// Check if the reference is to a fixed-size type or a struct
		for _, s := range schema.Structs {
			if s.Name == elemType.Ref {
				refs2 := make(map[string]ssz.Field)
				for _, st := range schema.Structs {
					refs2[st.Name] = st.ToSSZField()
				}
				field := s.ToSSZField()
				isVar, err := field.IsVariable(refs2)
				isFixed := !isVar
				if err == nil && isFixed {
					// Fixed-size types are byte slices
					return jen.Id(elemType.Ref)
				}
				// Variable-size types are structs
				return jen.Op("*").Id(elemType.Ref)
			}
		}
		// Default to assuming it's a fixed-size type
		return jen.Id(elemType.Ref)
	case ssz.TypeVector:
		// Vector elements in a list
		if len(elemType.Children) > 0 && elemType.Children[0].Type == ssz.TypeUint8 {
			return jen.Op("[").Lit(int(elemType.Size)).Op("]").Byte()
		}
		// TODO: Handle other vector types
		return jen.Op("[]").Byte()
	case ssz.TypeList:
		// List elements in a list (nested lists)
		if len(elemType.Children) > 0 && elemType.Children[0].Type == ssz.TypeUint8 {
			// List of bytes = []byte
			return jen.Op("[]").Byte()
		}
		// TODO: Handle other nested list types
		return jen.Op("[]").Interface()
	default:
		return jen.Interface()
	}
}


// generateListSSZSize generates the SSZSize method for a list type (no longer needed with generic ListOf)
// Keeping for backwards compatibility with containers that have list fields
func generateListSSZSize(f *jen.File, typeName string, elemType ssz.Field, schema *Schema) error {
	refs := make(map[string]ssz.Field)
	for _, s := range schema.Structs {
		refs[s.Name] = s.ToSSZField()
	}
	
	elemSize, err := getFieldSize(elemType, refs)
	if err != nil {
		// Variable-size elements
		f.Comment("SSZSize returns the serialized size of the list")
		f.Func().Params(jen.Id("l").Op("*").Id(typeName)).Id("SSZSize").Params().Int().Block(
			jen.Comment("4 bytes for length + sum of element sizes"),
			jen.Id("size").Op(":=").Lit(4),
			jen.For(jen.Id("_").Op(",").Id("elem").Op(":=").Range().Id("l").Dot("elements")).Block(
				jen.Id("size").Op("+=").Id("elem").Dot("SSZSize").Call(),
			),
			jen.Return(jen.Id("size")),
		)
		return nil
	}
	
	// Fixed-size elements
	f.Comment("SSZSize returns the serialized size of the list")
	f.Func().Params(jen.Id("l").Op("*").Id(typeName)).Id("SSZSize").Params().Int().Block(
		jen.Comment(fmt.Sprintf("4 bytes for length + %d bytes per element", elemSize)),
		jen.Return(jen.Lit(4).Op("+").Len(jen.Id("l").Dot("elements")).Op("*").Lit(elemSize)),
	)
	f.Line()
	
	return nil
}

// generateListMarshalSSZ generates the MarshalSSZ method for a list type (no longer needed with generic ListOf)
// Keeping for backwards compatibility with containers that have list fields
func generateListMarshalSSZ(f *jen.File, typeName string, elemType ssz.Field, schema *Schema) error {
	refs := make(map[string]ssz.Field)
	for _, s := range schema.Structs {
		refs[s.Name] = s.ToSSZField()
	}
	
	f.Comment("MarshalSSZ marshals the list into SSZ format")
	f.Func().Params(jen.Id("l").Op("*").Id(typeName)).Id("MarshalSSZ").Params().Params(jen.Op("[]").Byte(), jen.Error()).Block(
		jen.Id("buf").Op(":=").Make(jen.Op("[]").Byte(), jen.Id("l").Dot("SSZSize").Call()),
		jen.Id("_").Op(",").Id("err").Op(":=").Id("l").Dot("MarshalSSZTo").Call(jen.Id("buf")),
		jen.Return(jen.Id("buf"), jen.Id("err")),
	)
	f.Line()
	
	f.Comment("MarshalSSZTo marshals the list to the provided buffer")
	f.Func().Params(jen.Id("l").Op("*").Id(typeName)).Id("MarshalSSZTo").Params(jen.Id("buf").Op("[]").Byte()).Params(jen.Op("[]").Byte(), jen.Error()).Block(
		jen.Comment("Check buffer size"),
		jen.If(jen.Len(jen.Id("buf")).Op("<").Id("l").Dot("SSZSize").Call()).Block(
			jen.Return(jen.Nil(), jen.Qual("fmt", "Errorf").Call(
				jen.Lit("buffer too small: expected at least %d bytes, got %d"),
				jen.Id("l").Dot("SSZSize").Call(),
				jen.Len(jen.Id("buf")),
			)),
		),
		jen.Line(),
		
		jen.Comment("Write list length"),
		jen.Qual("encoding/binary", "LittleEndian").Dot("PutUint32").Call(
			jen.Id("buf").Op("[").Lit(0).Op(":").Lit(4).Op("]"),
			jen.Uint32().Call(jen.Len(jen.Id("l").Dot("elements"))),
		),
		jen.Id("offset").Op(":=").Lit(4),
		jen.Line(),
		
		jen.Comment("Write elements"),
		generateListMarshalElements(elemType, refs),
		
		jen.Return(jen.Id("buf"), jen.Nil()),
	)
	f.Line()
	
	return nil
}

// generateListMarshalElements generates code to marshal list elements
func generateListMarshalElements(elemType ssz.Field, refs map[string]ssz.Field) jen.Code {
	switch elemType.Type {
	case ssz.TypeUint8:
		return jen.Block(
			jen.Copy(jen.Id("buf").Op("[").Id("offset").Op(":"), jen.Id("l").Dot("elements")),
		)
	case ssz.TypeUint16:
		return jen.Block(
			jen.For(jen.Id("_").Op(",").Id("v").Op(":=").Range().Id("l").Dot("elements")).Block(
				jen.Qual("encoding/binary", "LittleEndian").Dot("PutUint16").Call(
					jen.Id("buf").Op("[").Id("offset").Op(":").Id("offset").Op("+").Lit(2).Op("]"),
					jen.Id("v"),
				),
				jen.Id("offset").Op("+=").Lit(2),
			),
		)
	case ssz.TypeUint32:
		return jen.Block(
			jen.For(jen.Id("_").Op(",").Id("v").Op(":=").Range().Id("l").Dot("elements")).Block(
				jen.Qual("encoding/binary", "LittleEndian").Dot("PutUint32").Call(
					jen.Id("buf").Op("[").Id("offset").Op(":").Id("offset").Op("+").Lit(4).Op("]"),
					jen.Id("v"),
				),
				jen.Id("offset").Op("+=").Lit(4),
			),
		)
	case ssz.TypeUint64:
		return jen.Block(
			jen.For(jen.Id("_").Op(",").Id("v").Op(":=").Range().Id("l").Dot("elements")).Block(
				jen.Qual("encoding/binary", "LittleEndian").Dot("PutUint64").Call(
					jen.Id("buf").Op("[").Id("offset").Op(":").Id("offset").Op("+").Lit(8).Op("]"),
					jen.Id("v"),
				),
				jen.Id("offset").Op("+=").Lit(8),
			),
		)
	case ssz.TypeBoolean:
		return jen.Block(
			jen.For(jen.Id("_").Op(",").Id("v").Op(":=").Range().Id("l").Dot("elements")).Block(
				jen.If(jen.Id("v")).Block(
					jen.Id("buf").Op("[").Id("offset").Op("]").Op("=").Lit(1),
				).Else().Block(
					jen.Id("buf").Op("[").Id("offset").Op("]").Op("=").Lit(0),
				),
				jen.Id("offset").Op("++"),
			),
		)
	case ssz.TypeRef:
		// Check if fixed or variable size
		if ref, ok := refs[elemType.Ref]; ok {
			isVar, _ := ref.IsVariable(refs)
			if !isVar {
				// Fixed-size reference types
				size, _ := getFieldSize(elemType, refs)
				return jen.Block(
					jen.For(jen.Id("i").Op(":=").Range().Id("l").Dot("elements")).Block(
						jen.Copy(
							jen.Id("buf").Op("[").Id("offset").Op(":").Id("offset").Op("+").Lit(size).Op("]"),
							jen.Id("l").Dot("elements").Op("[").Id("i").Op("]"),
						),
						jen.Id("offset").Op("+=").Lit(size),
					),
				)
			}
		}
		// Variable-size reference types
		return jen.Block(
			jen.For(jen.Id("_").Op(",").Id("elem").Op(":=").Range().Id("l").Dot("elements")).Block(
				jen.Block(
					jen.Id("_").Op(",").Id("err").Op(":=").Id("elem").Dot("MarshalSSZTo").Call(
						jen.Id("buf").Op("[").Id("offset").Op(":"),
					),
					jen.If(jen.Id("err").Op("!=").Nil()).Block(
						jen.Return(jen.Nil(), jen.Id("err")),
					),
					jen.Id("offset").Op("+=").Id("elem").Dot("SSZSize").Call(),
				),
			),
		)
	default:
		// Generic case
		return jen.Block(
			jen.Comment("TODO: Handle other element types"),
		)
	}
}

// generateListUnmarshalSSZ generates the UnmarshalSSZ method for a list type
func generateListUnmarshalSSZ(f *jen.File, typeName string, elemType ssz.Field, schema *Schema) error {
	refs := make(map[string]ssz.Field)
	for _, s := range schema.Structs {
		refs[s.Name] = s.ToSSZField()
	}
	
	f.Comment("UnmarshalSSZ unmarshals the list from SSZ format")
	f.Func().Params(jen.Id("l").Op("*").Id(typeName)).Id("UnmarshalSSZ").Params(jen.Id("buf").Op("[]").Byte()).Error().Block(
		jen.Comment("Check minimum size"),
		jen.If(jen.Len(jen.Id("buf")).Op("<").Lit(4)).Block(
			jen.Return(jen.Qual("fmt", "Errorf").Call(
				jen.Lit("buffer too small: expected at least 4 bytes for length, got %d"),
				jen.Len(jen.Id("buf")),
			)),
		),
		jen.Line(),
		
		jen.Comment("Read list length"),
		jen.Id("length").Op(":=").Qual("encoding/binary", "LittleEndian").Dot("Uint32").Call(
			jen.Id("buf").Op("[").Lit(0).Op(":").Lit(4).Op("]"),
		),
		jen.Line(),
		
		jen.Comment("Check limit"),
		jen.If(jen.Uint64().Call(jen.Id("length")).Op(">").Id("l").Dot("Limit").Call()).Block(
			jen.Return(jen.Qual("fmt", "Errorf").Call(
				jen.Lit("list length %d exceeds limit %d"),
				jen.Id("length"),
				jen.Id("l").Dot("Limit").Call(),
			)),
		),
		jen.Line(),
		
		generateListUnmarshalElements(elemType, refs),
		
		jen.Return(jen.Nil()),
	)
	f.Line()
	
	return nil
}

// generateListUnmarshalElements generates code to unmarshal list elements
func generateListUnmarshalElements(elemType ssz.Field, refs map[string]ssz.Field) jen.Code {
	// Use a dummy schema for getElementGoType
	schema := &Schema{Structs: []Field{}}
	for name, ref := range refs {
		children := make([]Field, len(ref.Children))
		for i, c := range ref.Children {
			children[i] = Field{
				Name: c.Name,
				Type: c.Type,
				Size: c.Size,
				Limit: c.Limit,
				Ref: c.Ref,
			}
		}
		schema.Structs = append(schema.Structs, Field{
			Name: name,
			Type: ref.Type,
			Size: ref.Size,
			Limit: ref.Limit,
			Ref: ref.Ref,
			Children: children,
		})
	}
	elemGoType := getElementGoType(elemType, schema)
	
	switch elemType.Type {
	case ssz.TypeUint8:
		return jen.Block(
			jen.Comment("Check buffer size for uint8 elements"),
			jen.If(jen.Len(jen.Id("buf")).Op("<").Lit(4).Op("+").Int().Call(jen.Id("length"))).Block(
				jen.Return(jen.Qual("fmt", "Errorf").Call(
					jen.Lit("buffer too small for %d uint8 elements"),
					jen.Id("length"),
				)),
			),
			jen.Id("l").Dot("elements").Op("=").Make(jen.Op("[]").Add(elemGoType), jen.Id("length")),
			jen.Copy(jen.Id("l").Dot("elements"), jen.Id("buf").Op("[").Lit(4).Op(":").Lit(4).Op("+").Id("length").Op("]")),
		)
	case ssz.TypeUint16:
		return jen.Block(
			jen.Comment("Check buffer size for uint16 elements"),
			jen.If(jen.Len(jen.Id("buf")).Op("<").Lit(4).Op("+").Int().Call(jen.Id("length")).Op("*").Lit(2)).Block(
				jen.Return(jen.Qual("fmt", "Errorf").Call(
					jen.Lit("buffer too small for %d uint16 elements"),
					jen.Id("length"),
				)),
			),
			jen.Id("l").Dot("elements").Op("=").Make(jen.Op("[]").Add(elemGoType), jen.Id("length")),
			jen.Id("offset").Op(":=").Lit(4),
			jen.For(jen.Id("i").Op(":=").Uint32().Call(jen.Lit(0)), jen.Id("i").Op("<").Id("length"), jen.Id("i").Op("++")).Block(
				jen.Id("l").Dot("elements").Op("[").Id("i").Op("]").Op("=").Qual("encoding/binary", "LittleEndian").Dot("Uint16").Call(
					jen.Id("buf").Op("[").Id("offset").Op(":").Id("offset").Op("+").Lit(2).Op("]"),
				),
				jen.Id("offset").Op("+=").Lit(2),
			),
		)
	case ssz.TypeUint32:
		return jen.Block(
			jen.Comment("Check buffer size for uint32 elements"),
			jen.If(jen.Len(jen.Id("buf")).Op("<").Lit(4).Op("+").Int().Call(jen.Id("length")).Op("*").Lit(4)).Block(
				jen.Return(jen.Qual("fmt", "Errorf").Call(
					jen.Lit("buffer too small for %d uint32 elements"),
					jen.Id("length"),
				)),
			),
			jen.Id("l").Dot("elements").Op("=").Make(jen.Op("[]").Add(elemGoType), jen.Id("length")),
			jen.Id("offset").Op(":=").Lit(4),
			jen.For(jen.Id("i").Op(":=").Uint32().Call(jen.Lit(0)), jen.Id("i").Op("<").Id("length"), jen.Id("i").Op("++")).Block(
				jen.Id("l").Dot("elements").Op("[").Id("i").Op("]").Op("=").Qual("encoding/binary", "LittleEndian").Dot("Uint32").Call(
					jen.Id("buf").Op("[").Id("offset").Op(":").Id("offset").Op("+").Lit(4).Op("]"),
				),
				jen.Id("offset").Op("+=").Lit(4),
			),
		)
	case ssz.TypeUint64:
		return jen.Block(
			jen.Comment("Check buffer size for uint64 elements"),
			jen.If(jen.Len(jen.Id("buf")).Op("<").Lit(4).Op("+").Int().Call(jen.Id("length")).Op("*").Lit(8)).Block(
				jen.Return(jen.Qual("fmt", "Errorf").Call(
					jen.Lit("buffer too small for %d uint64 elements"),
					jen.Id("length"),
				)),
			),
			jen.Id("l").Dot("elements").Op("=").Make(jen.Op("[]").Add(elemGoType), jen.Id("length")),
			jen.Id("offset").Op(":=").Lit(4),
			jen.For(jen.Id("i").Op(":=").Uint32().Call(jen.Lit(0)), jen.Id("i").Op("<").Id("length"), jen.Id("i").Op("++")).Block(
				jen.Id("l").Dot("elements").Op("[").Id("i").Op("]").Op("=").Qual("encoding/binary", "LittleEndian").Dot("Uint64").Call(
					jen.Id("buf").Op("[").Id("offset").Op(":").Id("offset").Op("+").Lit(8).Op("]"),
				),
				jen.Id("offset").Op("+=").Lit(8),
			),
		)
	case ssz.TypeBoolean:
		return jen.Block(
			jen.Comment("Check buffer size for boolean elements"),
			jen.If(jen.Len(jen.Id("buf")).Op("<").Lit(4).Op("+").Int().Call(jen.Id("length"))).Block(
				jen.Return(jen.Qual("fmt", "Errorf").Call(
					jen.Lit("buffer too small for %d boolean elements"),
					jen.Id("length"),
				)),
			),
			jen.Id("l").Dot("elements").Op("=").Make(jen.Op("[]").Add(elemGoType), jen.Id("length")),
			jen.Id("offset").Op(":=").Lit(4),
			jen.For(jen.Id("i").Op(":=").Uint32().Call(jen.Lit(0)), jen.Id("i").Op("<").Id("length"), jen.Id("i").Op("++")).Block(
				jen.Id("l").Dot("elements").Op("[").Id("i").Op("]").Op("=").Id("buf").Op("[").Id("offset").Op("]").Op("!=").Lit(0),
				jen.Id("offset").Op("++"),
			),
		)
	case ssz.TypeRef:
		// Check if fixed or variable size
		if ref, ok := refs[elemType.Ref]; ok {
			isVar, _ := ref.IsVariable(refs)
			if !isVar {
				// Fixed-size reference types
				size, _ := getFieldSize(elemType, refs)
				return jen.Block(
					jen.Comment("Check buffer size for fixed-size reference elements"),
					jen.If(jen.Len(jen.Id("buf")).Op("<").Lit(4).Op("+").Int().Call(jen.Id("length")).Op("*").Lit(size)).Block(
						jen.Return(jen.Qual("fmt", "Errorf").Call(
							jen.Lit("buffer too small for %d elements of size %d"),
							jen.Id("length"),
							jen.Lit(size),
						)),
					),
					jen.Id("l").Dot("elements").Op("=").Make(jen.Op("[]").Add(elemGoType), jen.Id("length")),
					jen.Id("offset").Op(":=").Lit(4),
					jen.For(jen.Id("i").Op(":=").Uint32().Call(jen.Lit(0)), jen.Id("i").Op("<").Id("length"), jen.Id("i").Op("++")).Block(
						jen.Id("l").Dot("elements").Op("[").Id("i").Op("]").Op("=").Make(jen.Id(elemType.Ref), jen.Lit(size)),
						jen.Copy(
							jen.Id("l").Dot("elements").Op("[").Id("i").Op("]"),
							jen.Id("buf").Op("[").Id("offset").Op(":").Id("offset").Op("+").Lit(size).Op("]"),
						),
						jen.Id("offset").Op("+=").Lit(size),
					),
				)
			}
		}
		// Variable-size reference types
		return jen.Block(
			jen.Comment("Variable-size reference elements"),
			jen.Id("l").Dot("elements").Op("=").Make(jen.Op("[]").Add(elemGoType), jen.Id("length")),
			jen.Id("offset").Op(":=").Lit(4),
			jen.For(jen.Id("i").Op(":=").Uint32().Call(jen.Lit(0)), jen.Id("i").Op("<").Id("length"), jen.Id("i").Op("++")).Block(
				jen.Id("l").Dot("elements").Op("[").Id("i").Op("]").Op("=").Op("&").Id(elemType.Ref).Op("{}"),
				jen.If(jen.Id("err").Op(":=").Id("l").Dot("elements").Op("[").Id("i").Op("]").Dot("UnmarshalSSZ").Call(
					jen.Id("buf").Op("[").Id("offset").Op(":"),
				), jen.Id("err").Op("!=").Nil()).Block(
					jen.Return(jen.Id("err")),
				),
				jen.Id("offset").Op("+=").Id("l").Dot("elements").Op("[").Id("i").Op("]").Dot("SSZSize").Call(),
			),
		)
	default:
		// Generic case
		return jen.Block(
			jen.Comment("TODO: Handle other element types"),
			jen.Return(jen.Qual("fmt", "Errorf").Call(jen.Lit("unsupported element type"))),
		)
	}
}

// generateListHashSSZTo generates the HashSSZTo method for a list type
func generateListHashSSZTo(f *jen.File, typeName string, elemType ssz.Field, schema *Schema) error {
	refs := make(map[string]ssz.Field)
	for _, s := range schema.Structs {
		refs[s.Name] = s.ToSSZField()
	}
	
	// For lists, we need to implement proper hashing according to SSZ spec
	// Lists are hashed as: hash_tree_root(List) = mix_in_length(merkle_root(pack(List)), len(List))
	
	f.Comment("HashSSZTo writes the merkle tree hash of the list to the provided buffer")
	f.Func().Params(jen.Id("l").Op("*").Id(typeName)).Id("HashSSZTo").Params(jen.Id("buf").Op("[]").Byte()).Params(jen.Op("[]").Byte(), jen.Error()).Block(
		jen.Comment("Ensure buffer has at least 32 bytes"),
		jen.If(jen.Len(jen.Id("buf")).Op("<").Lit(32)).Block(
			jen.Return(jen.Nil(), jen.Qual("fmt", "Errorf").Call(
				jen.Lit("buffer too small: need at least 32 bytes, got %d"),
				jen.Len(jen.Id("buf")),
			)),
		),
		jen.Line(),
		
		generateListHashTreeRootCode(elemType, refs),
		
		jen.Return(jen.Id("buf").Op("[:32]"), jen.Nil()),
	)
	f.Line()
	
	return nil
}

// generateListHashTreeRootCode generates the code to compute hash tree root for list elements
func generateListHashTreeRootCode(elemType ssz.Field, refs map[string]ssz.Field) jen.Code {
	elemSize, err := getFieldSize(elemType, refs)
	isFixedSize := err == nil
	
	if isFixedSize && elemSize <= 32 {
		// For small fixed-size elements, pack them into chunks
		elementsPerChunk := 32 / elemSize
		
		return jen.Block(
			jen.Comment(fmt.Sprintf("Pack elements into chunks (%d elements per chunk)", elementsPerChunk)),
			jen.Id("numElements").Op(":=").Len(jen.Id("l").Dot("elements")),
			jen.Id("numChunks").Op(":=").Parens(jen.Id("numElements").Op("+").Lit(elementsPerChunk-1)).Op("/").Lit(elementsPerChunk),
			jen.If(jen.Id("numChunks").Op("==").Lit(0)).Block(
				jen.Id("numChunks").Op("=").Lit(1),
			),
			jen.Line(),
			
			jen.Comment("Get buffer for packed chunks"),
			jen.Id("poolBuf").Op(":=").Qual("github.com/gfx-labs/ssz/merkle_tree/bufpool", "Get").Call(jen.Id("numChunks").Op("*").Lit(32)),
			jen.Defer().Qual("github.com/gfx-labs/ssz/merkle_tree/bufpool", "Put").Call(jen.Id("poolBuf")),
			jen.Id("chunkBuf").Op(":=").Id("poolBuf").Dot("B").Op("[:").Id("numChunks").Op("*").Lit(32).Op("]"),
			jen.Line(),
			
			jen.Comment("Clear the buffer"),
			jen.For(jen.Id("i").Op(":=").Range().Id("chunkBuf")).Block(
				jen.Id("chunkBuf").Op("[").Id("i").Op("]").Op("=").Lit(0),
			),
			jen.Line(),
			
			generateListPackingCode(elemType, elemSize, elementsPerChunk),
			jen.Line(),
			
			jen.Comment("Compute merkle root of packed chunks"),
			jen.If(jen.Id("numChunks").Op("==").Lit(1)).Block(
				jen.Comment("Single chunk, just copy it"),
				jen.Copy(jen.Id("buf"), jen.Id("chunkBuf")),
			).Else().Block(
				jen.If(jen.Err().Op(":=").Qual("github.com/gfx-labs/ssz/merkle_tree", "ComputeMerkleRoot").Call(
					jen.Id("chunkBuf"),
					jen.Id("chunkBuf"),
				), jen.Err().Op("!=").Nil()).Block(
					jen.Return(jen.Nil(), jen.Err()),
				),
				jen.Copy(jen.Id("buf"), jen.Id("chunkBuf").Op("[:32]")),
			),
			jen.Line(),
			
			jen.Comment("Mix in the length"),
			mixInLengthCode(),
		)
	} else {
		// For larger or variable-size elements, hash each individually
		return jen.Block(
			jen.Comment("Hash each element individually"),
			jen.Id("numElements").Op(":=").Len(jen.Id("l").Dot("elements")),
			jen.If(jen.Id("numElements").Op("==").Lit(0)).Block(
				jen.Comment("Empty list - return zero hash with mixed in length"),
				jen.For(jen.Id("i").Op(":=").Lit(0), jen.Id("i").Op("<").Lit(32), jen.Id("i").Op("++")).Block(
					jen.Id("buf").Op("[").Id("i").Op("]").Op("=").Lit(0),
				),
				mixInLengthCode(),
				jen.Return(jen.Id("buf").Op("[:32]"), jen.Nil()),
			),
			jen.Line(),
			
			jen.Comment("Get buffer for element hashes"),
			jen.Id("poolBuf").Op(":=").Qual("github.com/gfx-labs/ssz/merkle_tree/bufpool", "Get").Call(jen.Id("numElements").Op("*").Lit(32)),
			jen.Defer().Qual("github.com/gfx-labs/ssz/merkle_tree/bufpool", "Put").Call(jen.Id("poolBuf")),
			jen.Id("hashBuf").Op(":=").Id("poolBuf").Dot("B").Op("[:").Id("numElements").Op("*").Lit(32).Op("]"),
			jen.Line(),
			
			generateElementHashingCode(elemType, refs),
			jen.Line(),
			
			jen.Comment("Compute merkle root"),
			jen.If(jen.Err().Op(":=").Qual("github.com/gfx-labs/ssz/merkle_tree", "ComputeMerkleRoot").Call(
				jen.Id("hashBuf"),
				jen.Id("hashBuf"),
			), jen.Err().Op("!=").Nil()).Block(
				jen.Return(jen.Nil(), jen.Err()),
			),
			jen.Copy(jen.Id("buf"), jen.Id("hashBuf").Op("[:32]")),
			jen.Line(),
			
			jen.Comment("Mix in the length"),
			mixInLengthCode(),
		)
	}
}

// generateListPackingCode generates code to pack list elements into chunks
func generateListPackingCode(elemType ssz.Field, elemSize int, elementsPerChunk int) jen.Code {
	switch elemType.Type {
	case ssz.TypeUint8:
		return jen.Block(
			jen.Comment("Pack uint8 elements"),
			jen.Copy(jen.Id("chunkBuf"), jen.Id("l").Dot("elements")),
		)
	case ssz.TypeUint16:
		return jen.Block(
			jen.Comment("Pack uint16 elements"),
			jen.Id("offset").Op(":=").Lit(0),
			jen.For(jen.Id("_").Op(",").Id("v").Op(":=").Range().Id("l").Dot("elements")).Block(
				jen.Qual("encoding/binary", "LittleEndian").Dot("PutUint16").Call(
					jen.Id("chunkBuf").Op("[").Id("offset").Op(":").Id("offset").Op("+").Lit(2).Op("]"),
					jen.Id("v"),
				),
				jen.Id("offset").Op("+=").Lit(2),
			),
		)
	case ssz.TypeUint32:
		return jen.Block(
			jen.Comment("Pack uint32 elements"),
			jen.Id("offset").Op(":=").Lit(0),
			jen.For(jen.Id("_").Op(",").Id("v").Op(":=").Range().Id("l").Dot("elements")).Block(
				jen.Qual("encoding/binary", "LittleEndian").Dot("PutUint32").Call(
					jen.Id("chunkBuf").Op("[").Id("offset").Op(":").Id("offset").Op("+").Lit(4).Op("]"),
					jen.Id("v"),
				),
				jen.Id("offset").Op("+=").Lit(4),
			),
		)
	case ssz.TypeUint64:
		return jen.Block(
			jen.Comment("Pack uint64 elements"),
			jen.Id("offset").Op(":=").Lit(0),
			jen.For(jen.Id("_").Op(",").Id("v").Op(":=").Range().Id("l").Dot("elements")).Block(
				jen.Qual("encoding/binary", "LittleEndian").Dot("PutUint64").Call(
					jen.Id("chunkBuf").Op("[").Id("offset").Op(":").Id("offset").Op("+").Lit(8).Op("]"),
					jen.Id("v"),
				),
				jen.Id("offset").Op("+=").Lit(8),
			),
		)
	case ssz.TypeBoolean:
		return jen.Block(
			jen.Comment("Pack boolean elements"),
			jen.For(jen.Id("_").Op(",").Id("v").Op(":=").Range().Id("l").Dot("elements")).Block(
				jen.If(jen.Id("v")).Block(
					jen.Id("chunkBuf").Op("[").Id("i").Op("]").Op("=").Lit(1),
				).Else().Block(
					jen.Id("chunkBuf").Op("[").Id("i").Op("]").Op("=").Lit(0),
				),
			),
		)
	default:
		return jen.Comment("TODO: Handle other element types")
	}
}

// generateElementHashingCode generates code to hash individual elements
func generateElementHashingCode(elemType ssz.Field, refs map[string]ssz.Field) jen.Code {
	switch elemType.Type {
	case ssz.TypeRef:
		// Check if fixed or variable size
		if ref, ok := refs[elemType.Ref]; ok {
			isVar, _ := ref.IsVariable(refs)
			if !isVar {
				// Fixed-size reference types
				return jen.Block(
					jen.Comment("Hash fixed-size reference elements"),
					jen.For(jen.Id("i").Op(",").Id("elem").Op(":=").Range().Id("l").Dot("elements")).Block(
						jen.Id("_").Op(",").Id("err").Op(":=").Id("elem").Dot("HashSSZTo").Call(
							jen.Id("hashBuf").Op("[").Id("i").Op("*").Lit(32).Op(":").Parens(jen.Id("i").Op("+").Lit(1)).Op("*").Lit(32).Op("]"),
						),
						jen.If(jen.Id("err").Op("!=").Nil()).Block(
							jen.Return(jen.Nil(), jen.Id("err")),
						),
					),
				)
			}
		}
		// Variable-size types - need to implement differently
		return jen.Comment("TODO: Handle variable-size reference types")
	default:
		return jen.Comment("TODO: Handle other element types for hashing")
	}
}

// mixInLengthCode generates the code to mix in the list length
func mixInLengthCode() jen.Code {
	return jen.Block(
		jen.Id("lengthBuf").Op(":=").Make(jen.Op("[]").Byte(), jen.Lit(32)),
		jen.Qual("encoding/binary", "LittleEndian").Dot("PutUint64").Call(
			jen.Id("lengthBuf").Op("[0:8]"),
			jen.Uint64().Call(jen.Len(jen.Id("l").Dot("elements"))),
		),
		jen.For(jen.Id("i").Op(":=").Lit(0), jen.Id("i").Op("<").Lit(32), jen.Id("i").Op("++")).Block(
			jen.Id("buf").Op("[").Id("i").Op("]").Op("^=").Id("lengthBuf").Op("[").Id("i").Op("]"),
		),
	)
}

// generateListHashTreeRoot generates the HashTreeRoot method for a list type
func generateListHashTreeRoot(f *jen.File, typeName string, elemType ssz.Field, schema *Schema) error {
	f.Comment("HashTreeRoot returns the Merkle root of the list")
	f.Func().Params(jen.Id("l").Op("*").Id(typeName)).Id("HashTreeRoot").Params().Params(jen.Op("[").Lit(32).Op("]").Byte(), jen.Error()).Block(
		jen.Id("hash").Op(":=").Op("[").Lit(32).Op("]").Byte().Op("{}"),
		jen.Id("_").Op(",").Id("err").Op(":=").Id("l").Dot("HashSSZTo").Call(jen.Id("hash").Op("[:]")),
		jen.Return(jen.Id("hash"), jen.Id("err")),
	)
	f.Line()
	
	return nil
}

