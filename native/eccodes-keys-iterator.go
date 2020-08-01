package native

/*
#include <eccodes.h>
*/
import "C"
import (
	"unsafe"

	"github.com/pkg/errors"
)

const (
	CODES_KEYS_ITERATOR_ALL_KEYS              = 0
	CODES_KEYS_ITERATOR_SKIP_READ_ONLY        = 1 << 0
	CODES_KEYS_ITERATOR_SKIP_OPTIONAL         = 1 << 1
	CODES_KEYS_ITERATOR_SKIP_EDITION_SPECIFIC = 1 << 2
	CODES_KEYS_ITERATOR_SKIP_CODED            = 1 << 3
	CODES_KEYS_ITERATOR_SKIP_COMPUTED         = 1 << 4
	CODES_KEYS_ITERATOR_SKIP_DUPLICATES       = 1 << 5
	CODES_KEYS_ITERATOR_SKIP_FUNCTION         = 1 << 6
	CODES_KEYS_ITERATOR_DUMP_ONLY             = 1 << 7
)

func Ccodes_keys_iterator_new(handle Ccodes_handle, flags int, namespace string) Ccodes_keys_iterator {
	var cNamespace *C.char

	if len(namespace) > 0 {
		cNamespace = C.CString(namespace)
		defer C.free(unsafe.Pointer(cNamespace))
	}

	return unsafe.Pointer(C.codes_keys_iterator_new((*C.codes_handle)(handle), C.ulong(Culong(flags)), cNamespace))
}

func Ccodes_keys_iterator_next(kiter Ccodes_keys_iterator) int {
	return int(C.codes_keys_iterator_next((*C.codes_keys_iterator)(kiter)))
}

func Ccodes_keys_iterator_get_name(kiter Ccodes_keys_iterator) string {
	return C.GoString(C.codes_keys_iterator_get_name((*C.codes_keys_iterator)(kiter)))
}

func Ccodes_keys_iterator_delete(kiter Ccodes_keys_iterator) error {
	err := C.codes_keys_iterator_delete((*C.codes_keys_iterator)(kiter))
	if err != 0 {
		return errors.New(Cgrib_get_error_message(int(err)))
	}
	return nil
}
