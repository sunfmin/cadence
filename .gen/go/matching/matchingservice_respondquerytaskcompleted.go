// Code generated by thriftrw v1.6.0. DO NOT EDIT.
// @generated

package matching

import (
	"errors"
	"fmt"
	"github.com/uber/cadence/.gen/go/shared"
	"go.uber.org/thriftrw/wire"
	"strings"
)

type MatchingService_RespondQueryTaskCompleted_Args struct {
	Request *RespondQueryTaskCompletedRequest `json:"request,omitempty"`
}

func (v *MatchingService_RespondQueryTaskCompleted_Args) ToWire() (wire.Value, error) {
	var (
		fields [1]wire.Field
		i      int = 0
		w      wire.Value
		err    error
	)
	if v.Request != nil {
		w, err = v.Request.ToWire()
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 1, Value: w}
		i++
	}
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func _RespondQueryTaskCompletedRequest_1_Read(w wire.Value) (*RespondQueryTaskCompletedRequest, error) {
	var v RespondQueryTaskCompletedRequest
	err := v.FromWire(w)
	return &v, err
}

func (v *MatchingService_RespondQueryTaskCompleted_Args) FromWire(w wire.Value) error {
	var err error
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 1:
			if field.Value.Type() == wire.TStruct {
				v.Request, err = _RespondQueryTaskCompletedRequest_1_Read(field.Value)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (v *MatchingService_RespondQueryTaskCompleted_Args) String() string {
	if v == nil {
		return "<nil>"
	}
	var fields [1]string
	i := 0
	if v.Request != nil {
		fields[i] = fmt.Sprintf("Request: %v", v.Request)
		i++
	}
	return fmt.Sprintf("MatchingService_RespondQueryTaskCompleted_Args{%v}", strings.Join(fields[:i], ", "))
}

func (v *MatchingService_RespondQueryTaskCompleted_Args) Equals(rhs *MatchingService_RespondQueryTaskCompleted_Args) bool {
	if !((v.Request == nil && rhs.Request == nil) || (v.Request != nil && rhs.Request != nil && v.Request.Equals(rhs.Request))) {
		return false
	}
	return true
}

func (v *MatchingService_RespondQueryTaskCompleted_Args) MethodName() string {
	return "RespondQueryTaskCompleted"
}

func (v *MatchingService_RespondQueryTaskCompleted_Args) EnvelopeType() wire.EnvelopeType {
	return wire.Call
}

var MatchingService_RespondQueryTaskCompleted_Helper = struct {
	Args           func(request *RespondQueryTaskCompletedRequest) *MatchingService_RespondQueryTaskCompleted_Args
	IsException    func(error) bool
	WrapResponse   func(error) (*MatchingService_RespondQueryTaskCompleted_Result, error)
	UnwrapResponse func(*MatchingService_RespondQueryTaskCompleted_Result) error
}{}

func init() {
	MatchingService_RespondQueryTaskCompleted_Helper.Args = func(request *RespondQueryTaskCompletedRequest) *MatchingService_RespondQueryTaskCompleted_Args {
		return &MatchingService_RespondQueryTaskCompleted_Args{Request: request}
	}
	MatchingService_RespondQueryTaskCompleted_Helper.IsException = func(err error) bool {
		switch err.(type) {
		case *shared.BadRequestError:
			return true
		case *shared.InternalServiceError:
			return true
		case *shared.EntityNotExistsError:
			return true
		default:
			return false
		}
	}
	MatchingService_RespondQueryTaskCompleted_Helper.WrapResponse = func(err error) (*MatchingService_RespondQueryTaskCompleted_Result, error) {
		if err == nil {
			return &MatchingService_RespondQueryTaskCompleted_Result{}, nil
		}
		switch e := err.(type) {
		case *shared.BadRequestError:
			if e == nil {
				return nil, errors.New("WrapResponse received non-nil error type with nil value for MatchingService_RespondQueryTaskCompleted_Result.BadRequestError")
			}
			return &MatchingService_RespondQueryTaskCompleted_Result{BadRequestError: e}, nil
		case *shared.InternalServiceError:
			if e == nil {
				return nil, errors.New("WrapResponse received non-nil error type with nil value for MatchingService_RespondQueryTaskCompleted_Result.InternalServiceError")
			}
			return &MatchingService_RespondQueryTaskCompleted_Result{InternalServiceError: e}, nil
		case *shared.EntityNotExistsError:
			if e == nil {
				return nil, errors.New("WrapResponse received non-nil error type with nil value for MatchingService_RespondQueryTaskCompleted_Result.EntityNotExistError")
			}
			return &MatchingService_RespondQueryTaskCompleted_Result{EntityNotExistError: e}, nil
		}
		return nil, err
	}
	MatchingService_RespondQueryTaskCompleted_Helper.UnwrapResponse = func(result *MatchingService_RespondQueryTaskCompleted_Result) (err error) {
		if result.BadRequestError != nil {
			err = result.BadRequestError
			return
		}
		if result.InternalServiceError != nil {
			err = result.InternalServiceError
			return
		}
		if result.EntityNotExistError != nil {
			err = result.EntityNotExistError
			return
		}
		return
	}
}

type MatchingService_RespondQueryTaskCompleted_Result struct {
	BadRequestError      *shared.BadRequestError      `json:"badRequestError,omitempty"`
	InternalServiceError *shared.InternalServiceError `json:"internalServiceError,omitempty"`
	EntityNotExistError  *shared.EntityNotExistsError `json:"entityNotExistError,omitempty"`
}

func (v *MatchingService_RespondQueryTaskCompleted_Result) ToWire() (wire.Value, error) {
	var (
		fields [3]wire.Field
		i      int = 0
		w      wire.Value
		err    error
	)
	if v.BadRequestError != nil {
		w, err = v.BadRequestError.ToWire()
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 1, Value: w}
		i++
	}
	if v.InternalServiceError != nil {
		w, err = v.InternalServiceError.ToWire()
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 2, Value: w}
		i++
	}
	if v.EntityNotExistError != nil {
		w, err = v.EntityNotExistError.ToWire()
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 3, Value: w}
		i++
	}
	if i > 1 {
		return wire.Value{}, fmt.Errorf("MatchingService_RespondQueryTaskCompleted_Result should have at most one field: got %v fields", i)
	}
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func (v *MatchingService_RespondQueryTaskCompleted_Result) FromWire(w wire.Value) error {
	var err error
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 1:
			if field.Value.Type() == wire.TStruct {
				v.BadRequestError, err = _BadRequestError_Read(field.Value)
				if err != nil {
					return err
				}
			}
		case 2:
			if field.Value.Type() == wire.TStruct {
				v.InternalServiceError, err = _InternalServiceError_Read(field.Value)
				if err != nil {
					return err
				}
			}
		case 3:
			if field.Value.Type() == wire.TStruct {
				v.EntityNotExistError, err = _EntityNotExistsError_Read(field.Value)
				if err != nil {
					return err
				}
			}
		}
	}
	count := 0
	if v.BadRequestError != nil {
		count++
	}
	if v.InternalServiceError != nil {
		count++
	}
	if v.EntityNotExistError != nil {
		count++
	}
	if count > 1 {
		return fmt.Errorf("MatchingService_RespondQueryTaskCompleted_Result should have at most one field: got %v fields", count)
	}
	return nil
}

func (v *MatchingService_RespondQueryTaskCompleted_Result) String() string {
	if v == nil {
		return "<nil>"
	}
	var fields [3]string
	i := 0
	if v.BadRequestError != nil {
		fields[i] = fmt.Sprintf("BadRequestError: %v", v.BadRequestError)
		i++
	}
	if v.InternalServiceError != nil {
		fields[i] = fmt.Sprintf("InternalServiceError: %v", v.InternalServiceError)
		i++
	}
	if v.EntityNotExistError != nil {
		fields[i] = fmt.Sprintf("EntityNotExistError: %v", v.EntityNotExistError)
		i++
	}
	return fmt.Sprintf("MatchingService_RespondQueryTaskCompleted_Result{%v}", strings.Join(fields[:i], ", "))
}

func (v *MatchingService_RespondQueryTaskCompleted_Result) Equals(rhs *MatchingService_RespondQueryTaskCompleted_Result) bool {
	if !((v.BadRequestError == nil && rhs.BadRequestError == nil) || (v.BadRequestError != nil && rhs.BadRequestError != nil && v.BadRequestError.Equals(rhs.BadRequestError))) {
		return false
	}
	if !((v.InternalServiceError == nil && rhs.InternalServiceError == nil) || (v.InternalServiceError != nil && rhs.InternalServiceError != nil && v.InternalServiceError.Equals(rhs.InternalServiceError))) {
		return false
	}
	if !((v.EntityNotExistError == nil && rhs.EntityNotExistError == nil) || (v.EntityNotExistError != nil && rhs.EntityNotExistError != nil && v.EntityNotExistError.Equals(rhs.EntityNotExistError))) {
		return false
	}
	return true
}

func (v *MatchingService_RespondQueryTaskCompleted_Result) MethodName() string {
	return "RespondQueryTaskCompleted"
}

func (v *MatchingService_RespondQueryTaskCompleted_Result) EnvelopeType() wire.EnvelopeType {
	return wire.Reply
}