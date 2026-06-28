//go:build darwin

package sudo

/*
#cgo LDFLAGS: -framework Foundation -framework LocalAuthentication
#include <stdlib.h>
#include "touchid.h"
*/
import "C"

import "unsafe"

// Authenticate prompts for Touch ID or the device password.
func Authenticate(reason string) bool {
	if reason == "" {
		reason = "Confirm this action"
	}

	cReason := C.CString(reason)
	defer C.free(unsafe.Pointer(cReason))

	return C.ConfirmDeviceOwner(cReason, nil, 0) == 0
}
