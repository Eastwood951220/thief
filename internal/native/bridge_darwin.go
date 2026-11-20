package native

/*
#cgo CFLAGS: -mmacosx-version-min=13.0
#cgo LDFLAGS: -framework AppKit -framework CoreText -framework Carbon -mmacosx-version-min=13.0
#include <stdlib.h>
#include "bridge.h"
*/
import "C"
import "unsafe"

type Event struct { Kind, Payload string }
var handler func(Event)
func Run(onEvent func(Event)) { handler = onEvent; C.thief_run() }
func Apply(json string) { p:=C.CString(json); defer C.free(unsafe.Pointer(p)); C.thief_apply(p) }

//export thief_event
func thief_event(kind, payload *C.char) { if handler!=nil { handler(Event{C.GoString(kind),C.GoString(payload)}) } }
