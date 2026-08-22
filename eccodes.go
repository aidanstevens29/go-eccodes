// Package eccodes provides Go bindings to ECMWF ecCodes for reading GRIB files.
//
// A Reader streams messages from a GRIB file. Each returned Message owns an
// ecCodes handle and must be closed by the caller.
package eccodes

/*
#cgo pkg-config: eccodes
#include <errno.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <eccodes.h>

static FILE* go_eccodes_fopen(const char* path, int* err) {
	FILE* file = fopen(path, "rb");
	if (file == NULL) {
		*err = errno;
	}
	return file;
}
*/
import "C"

import (
	"errors"
	"fmt"
	"io"
	"runtime"
	"strings"
	"sync"
	"unsafe"
)

// ErrClosed is returned when an operation is attempted on a closed object.
var ErrClosed = errors.New("eccodes: object is closed")

// TargetVersion is the upstream ecCodes release this package is modeled on.
const TargetVersion = "2.48.0"

// Version describes the linked ecCodes library version.
type Version struct {
	Major int
	Minor int
	Patch int
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// RuntimeVersion returns the version of the ecCodes library linked at runtime.
func RuntimeVersion() Version {
	value := int64(C.codes_get_api_version())
	return Version{
		Major: int(value / 10000),
		Minor: int(value / 100 % 100),
		Patch: int(value % 100),
	}
}

// Error is an error returned by the ecCodes library.
type Error struct {
	Code int
	Op   string
	Key  string
}

func (e *Error) Error() string {
	message := C.GoString(C.codes_get_error_message(C.int(e.Code)))
	if e.Key != "" {
		return fmt.Sprintf("eccodes: %s %q: %s (code %d)", e.Op, e.Key, message, e.Code)
	}
	return fmt.Sprintf("eccodes: %s: %s (code %d)", e.Op, message, e.Code)
}

func codeError(code C.int, op, key string) error {
	if code == C.CODES_SUCCESS {
		return nil
	}
	return &Error{Code: int(code), Op: op, Key: key}
}

// Reader streams GRIB messages from a file.
type Reader struct {
	mu     sync.Mutex
	file   *C.FILE
	closed bool
}

// Open opens path for reading GRIB messages.
func Open(path string) (*Reader, error) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))

	var errno C.int
	file := C.go_eccodes_fopen(cpath, &errno)
	if file == nil {
		return nil, fmt.Errorf("eccodes: open %q: %s", path, C.GoString(C.strerror(errno)))
	}
	reader := &Reader{file: file}
	runtime.SetFinalizer(reader, (*Reader).finalize)
	return reader, nil
}

func (r *Reader) finalize() { _ = r.Close() }

// Next returns the next GRIB message. It returns io.EOF after the last message.
// The caller must close each returned message.
func (r *Reader) Next() (*Message, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil, ErrClosed
	}

	var code C.int
	handle := C.codes_grib_handle_new_from_file(nil, r.file, &code)
	if handle == nil {
		if code == C.CODES_SUCCESS || code == C.CODES_END_OF_FILE {
			return nil, io.EOF
		}
		return nil, codeError(code, "read message", "")
	}
	return newMessage(handle), nil
}

// Close closes the underlying file. Messages already returned by Next remain
// valid until they are closed.
func (r *Reader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil
	}
	r.closed = true
	runtime.SetFinalizer(r, nil)
	if C.fclose(r.file) != 0 {
		return errors.New("eccodes: close file failed")
	}
	r.file = nil
	return nil
}

// Message is one GRIB message backed by an ecCodes handle.
type Message struct {
	mu     sync.Mutex
	handle *C.codes_handle
	closed bool
}

func newMessage(handle *C.codes_handle) *Message {
	message := &Message{handle: handle}
	runtime.SetFinalizer(message, (*Message).finalize)
	return message
}

// NewMessage parses one encoded GRIB message from memory. ecCodes copies data,
// so the caller may reuse the input slice after this function returns.
func NewMessage(data []byte) (*Message, error) {
	if len(data) == 0 {
		return nil, errors.New("eccodes: parse message: empty input")
	}
	handle := C.codes_handle_new_from_message_copy(nil, unsafe.Pointer(&data[0]), C.size_t(len(data)))
	runtime.KeepAlive(data)
	if handle == nil {
		return nil, errors.New("eccodes: parse message: invalid GRIB message")
	}
	return newMessage(handle), nil
}

func (m *Message) finalize() { _ = m.Close() }

// Close releases the native ecCodes handle. It is safe to call more than once.
func (m *Message) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	m.closed = true
	runtime.SetFinalizer(m, nil)
	code := C.codes_handle_delete(m.handle)
	m.handle = nil
	return codeError(code, "close message", "")
}

func (m *Message) withKey(key string, fn func(*C.codes_handle, *C.char) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return ErrClosed
	}
	ckey := C.CString(key)
	defer C.free(unsafe.Pointer(ckey))
	return fn(m.handle, ckey)
}

// Long returns a signed integer key.
func (m *Message) Long(key string) (int64, error) {
	var value C.long
	err := m.withKey(key, func(handle *C.codes_handle, ckey *C.char) error {
		return codeError(C.codes_get_long(handle, ckey, &value), "get long", key)
	})
	return int64(value), err
}

// Double returns a floating-point key.
func (m *Message) Double(key string) (float64, error) {
	var value C.double
	err := m.withKey(key, func(handle *C.codes_handle, ckey *C.char) error {
		return codeError(C.codes_get_double(handle, ckey, &value), "get double", key)
	})
	return float64(value), err
}

// String returns a string key.
func (m *Message) String(key string) (string, error) {
	var value string
	err := m.withKey(key, func(handle *C.codes_handle, ckey *C.char) error {
		size := C.size_t(256)
		for {
			if uint64(size) > uint64(maxInt()) {
				return fmt.Errorf("eccodes: get string %q: value is too large", key)
			}
			buffer := make([]byte, int(size))
			code := C.codes_get_string(handle, ckey, (*C.char)(unsafe.Pointer(&buffer[0])), &size)
			if code == C.CODES_BUFFER_TOO_SMALL {
				continue
			}
			if err := codeError(code, "get string", key); err != nil {
				return err
			}
			value = strings.TrimRight(string(buffer[:int(size)]), "\x00")
			return nil
		}
	})
	return value, err
}

// Size returns the number of values stored under key.
func (m *Message) Size(key string) (int, error) {
	var size C.size_t
	err := m.withKey(key, func(handle *C.codes_handle, ckey *C.char) error {
		return codeError(C.codes_get_size(handle, ckey, &size), "get size", key)
	})
	if uint64(size) > uint64(maxInt()) {
		return 0, fmt.Errorf("eccodes: get size %q: value is too large", key)
	}
	return int(size), err
}

// Doubles returns all floating-point values stored under key. For a decoded
// GRIB field, Doubles("values") returns the grid-point values.
func (m *Message) Doubles(key string) ([]float64, error) {
	var values []float64
	err := m.withKey(key, func(handle *C.codes_handle, ckey *C.char) error {
		var size C.size_t
		if err := codeError(C.codes_get_size(handle, ckey, &size), "get size", key); err != nil {
			return err
		}
		if uint64(size) > uint64(maxInt()) {
			return fmt.Errorf("eccodes: get doubles %q: value is too large", key)
		}
		values = make([]float64, int(size))
		if size == 0 {
			return nil
		}
		return codeError(C.codes_get_double_array(handle, ckey, (*C.double)(unsafe.Pointer(&values[0])), &size), "get doubles", key)
	})
	return values, err
}

// Longs returns all signed integer values stored under key.
func (m *Message) Longs(key string) ([]int64, error) {
	var values []int64
	err := m.withKey(key, func(handle *C.codes_handle, ckey *C.char) error {
		var size C.size_t
		if err := codeError(C.codes_get_size(handle, ckey, &size), "get size", key); err != nil {
			return err
		}
		if uint64(size) > uint64(maxInt()) {
			return fmt.Errorf("eccodes: get longs %q: value is too large", key)
		}
		native := make([]C.long, int(size))
		if size > 0 {
			if err := codeError(C.codes_get_long_array(handle, ckey, &native[0], &size), "get longs", key); err != nil {
				return err
			}
		}
		values = make([]int64, len(native))
		for i, value := range native {
			values[i] = int64(value)
		}
		return nil
	})
	return values, err
}

// Bytes returns the encoded GRIB message. The returned slice is an independent
// copy and remains valid after the Message is closed.
func (m *Message) Bytes() ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil, ErrClosed
	}
	var size C.size_t
	if err := codeError(C.codes_get_message_size(m.handle, &size), "get message size", ""); err != nil {
		return nil, err
	}
	if uint64(size) > uint64(maxInt()) {
		return nil, errors.New("eccodes: message is too large")
	}
	data := make([]byte, int(size))
	if size == 0 {
		return data, nil
	}
	if err := codeError(C.codes_get_message_copy(m.handle, unsafe.Pointer(&data[0]), &size), "copy message", ""); err != nil {
		return nil, err
	}
	return data[:int(size)], nil
}

func maxInt() int { return int(^uint(0) >> 1) }
