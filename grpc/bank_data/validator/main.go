package validator

import (
	"github.com/shadiestgoat/bankDataDB/pb/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/protoadapt"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type Validation func(msg protoreflect.Message, req protoadapt.MessageV2) ([]string, *string)

func NewFieldValidation(fieldName string, required bool, f func(protoreflect.Value) *string) Validation {
	return func(msg protoreflect.Message, _ protoadapt.MessageV2) ([]string, *string) {
		fields := msg.Descriptor().Fields()
		field := fields.ByTextName(fieldName)
		isUpdate := fields.ByTextName("id") == nil

		if !msg.Has(field) {
			if !isUpdate && required {
				return []string{fieldName}, new("required")
			}
			return nil, nil
		}

		errMsg := f(msg.Get(field))
		if errMsg == nil {
			return nil, nil
		}

		return []string{fieldName}, errMsg
	}
}

func NewMessageValidation[T protoadapt.MessageV2](fields []string, f func(req T) *string) Validation {
	return func(_ protoreflect.Message, req protoadapt.MessageV2) ([]string, *string) {
		return fields, f(req.(T))
	}
}

type Validator struct {
	Validations []Validation
}

func (v Validator) Validate(val protoadapt.MessageV2) error {
	valErrors := make([]protoadapt.MessageV1, 0, len(v.Validations))

	md := val.ProtoReflect()

	for _, f := range v.Validations {
		fields, msg := f(md, val)

		if msg != nil {
			valErrors = append(valErrors, errors.ValidationError_builder{
				Fields:  fields,
				Message: msg,
			}.Build())
		}
	}

	if len(valErrors) == 0 {
		return nil
	}

	s, err := status.New(codes.InvalidArgument, "validation error").WithDetails(valErrors...)
	if err != nil {
		// Rather loud than quiet
		panic("Unable to build a validation error: " + err.Error())
	}

	return s.Err()
}
