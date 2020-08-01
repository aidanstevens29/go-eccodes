package codes

import (
	"fmt"
	"github.com/pkg/errors"
	"math"
	"runtime"

	"github.com/aidanstevens29/go-eccodes/debug"
	"github.com/aidanstevens29/go-eccodes/native"
)

// error codes
var (
	ErrUnknownTypeCode = errors.New("unknown type code")
	ErrNotImplemented  = errors.New("not implemented")
)

// type names constants
const (
	TypeInt64   = "int64"
	TypeFloat64 = "float64"
	TypeString  = "string"
)

type Message interface {
	isOpen() bool

	GetInterface(key string) (interface{}, error)

	GetString(key string) (string, error)

	GetLong(key string) (int64, error)
	SetLong(key string, value int64) error
	GetLongArray(key string) ([]int64, error)

	GetDouble(key string) (float64, error)
	SetDouble(key string, value float64) error
	GetDoubleArray(key string) ([]float64, error)

	Data() (latitudes []float64, longitudes []float64, values []float64, err error)
	DataUnsafe() (latitudes *Float64ArrayUnsafe, longitudes *Float64ArrayUnsafe, values *Float64ArrayUnsafe, err error)

	GetSize(key string) (int64, error)
	GetType(key string) (string, error)
	Keys() []string

	Close() error
}

type message struct {
	handle native.Ccodes_handle
}

func newMessage(h native.Ccodes_handle) Message {
	m := &message{handle: h}
	runtime.SetFinalizer(m, messageFinalizer)

	// set missing value to NaN
	m.SetDouble(parameterMissingValue, math.NaN())

	return m
}

func (m *message) isOpen() bool {
	return m.handle != nil
}

func (m *message) GetInterface(key string) (interface{}, error) {
	typeString, err := m.GetType(key)
	if err != nil {
		return nil, err
	}

	size, err := m.GetSize(key)
	if err != nil {
		return nil, err
	}

	if size == 1 {
		switch typeString {
		case TypeInt64:
			return m.GetLong(key)
		case TypeFloat64:
			return m.GetDouble(key)
		case TypeString:
			return m.GetString(key)
		default:
			return nil, ErrUnknownTypeCode
		}
	}

	// array
	switch typeString {
	case TypeInt64:
		return m.GetLongArray(key)
	case TypeFloat64:
		return m.GetDoubleArray(key)
	case TypeString:
		return nil, ErrNotImplemented
	default:
		return nil, ErrUnknownTypeCode
	}
}

func (m *message) GetString(key string) (string, error) {
	return native.Ccodes_get_string(m.handle, key)
}

func (m *message) GetLong(key string) (int64, error) {
	return native.Ccodes_get_long(m.handle, key)
}

func (m *message) SetLong(key string, value int64) error {
	return native.Ccodes_set_long(m.handle, key, value)
}

func (m *message) GetLongArray(key string) ([]int64, error) {
	return native.Ccodes_get_long_array(m.handle, key)
}

func (m *message) GetDouble(key string) (float64, error) {
	return native.Ccodes_get_double(m.handle, key)
}

func (m *message) SetDouble(key string, value float64) error {
	return native.Ccodes_set_double(m.handle, key, value)
}

func (m *message) GetDoubleArray(key string) ([]float64, error) {
	return native.Ccodes_get_double_array(m.handle, key)
}

func (m *message) Data() (latitudes []float64, longitudes []float64, values []float64, err error) {
	return native.Ccodes_grib_get_data(m.handle)
}

func (m *message) DataUnsafe() (latitudes *Float64ArrayUnsafe, longitudes *Float64ArrayUnsafe, values *Float64ArrayUnsafe, err error) {
	lats, lons, vals, err := native.Ccodes_grib_get_data_unsafe(m.handle)
	if err != nil {
		return nil, nil, nil, err
	}
	return newFloat64ArrayUnsafe(lats), newFloat64ArrayUnsafe(lons), newFloat64ArrayUnsafe(vals), nil
}

func (m *message) GetSize(key string) (int64, error) {
	return native.Ccodes_get_size(m.handle, key)
}

func (m *message) GetType(key string) (string, error) {
	typeCode, err := native.Ccodes_get_native_type(m.handle, key)
	if err != nil {
		return "", err
	}
	switch typeCode {
	case 1:
		return TypeInt64, nil
	case 2:
		return TypeFloat64, nil
	case 3:
		return TypeString, nil
	default:
		return fmt.Sprintf("type %d", typeCode), ErrUnknownTypeCode
	}
}

func (m *message) Keys() []string {
	iter := native.Ccodes_keys_iterator_new(m.handle, native.CODES_KEYS_ITERATOR_ALL_KEYS, "")
	defer native.Ccodes_keys_iterator_delete(iter)

	result := make([]string, 0)
	for native.Ccodes_keys_iterator_next(iter) == 1 {
		result = append(result, native.Ccodes_keys_iterator_get_name(iter))
	}

	return result
}

func (m *message) Close() error {
	defer func() { m.handle = nil }()
	return native.Ccodes_handle_delete(m.handle)
}

func messageFinalizer(m *message) {
	if m.isOpen() {
		debug.MemoryLeakLogger.Print("message is not closed")
		m.Close()
	}
}
