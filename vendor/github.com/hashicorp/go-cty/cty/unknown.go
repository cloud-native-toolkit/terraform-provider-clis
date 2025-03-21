// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MIT

package cty

// unknownType is the placeholder type used for the sigil value representing
// "Unknown", to make it unambigiously distinct from any other possible value.
type unknownType struct {
}

// Unknown is a special value that can be
var unknown interface{} = &unknownType{}

// UnknownVal returns an Value that represents an unknown value of the given
// type. Unknown values can be used to represent a value that is
// not yet known. Its meaning is undefined in cty, but it could be used by
// an calling application to allow partial evaluation.
//
// Unknown values of any type can be created of any type. All operations on
// Unknown values themselves return Unknown.
func UnknownVal(t Type) Value {
	return Value{
		ty: t,
		v:  unknown,
	}
}

func (t unknownType) GoString() string {
	// This is the stringification of our internal unknown marker. The
	// stringification of the public representation of unknowns is in
	// Value.GoString.
	return "cty.unknown"
}

type pseudoTypeDynamic struct {
	typeImplSigil
}

// DynamicPseudoType represents the dynamic pseudo-type.
//
// This type can represent situations where a type is not yet known. Its
// meaning is undefined in cty, but it could be used by a calling
// application to allow expression type checking with some types not yet known.
// For example, the application might optimistically permit any operation on
// values of this type in type checking, allowing a partial type-check result,
// and then repeat the check when more information is known to get the
// final, concrete type.
//
// It is a pseudo-type because it is used only as a sigil to the calling
// application. "Unknown" is the only valid value of this pseudo-type, so
// operations on values of this type will always short-circuit as per
// the rules for that special value.
var DynamicPseudoType Type

func (t pseudoTypeDynamic) Equals(other Type) bool {
	_, ok := other.typeImpl.(pseudoTypeDynamic)
	return ok
}

func (t pseudoTypeDynamic) FriendlyName(mode friendlyTypeNameMode) string {
	switch mode {
	case friendlyTypeConstraintName:
		return "any type"
	default:
		return "dynamic"
	}
}

func (t pseudoTypeDynamic) GoString() string {
	return "cty.DynamicPseudoType"
}

// DynamicVal is the only valid value of the pseudo-type dynamic.
// This value can be used as a placeholder where a value or expression's
// type and value are both unknown, thus allowing partial evaluation. See
// the docs for DynamicPseudoType for more information.
var DynamicVal Value

func init() {
	DynamicPseudoType = Type{
		pseudoTypeDynamic{},
	}
	DynamicVal = Value{
		ty: DynamicPseudoType,
		v:  unknown,
	}
}
