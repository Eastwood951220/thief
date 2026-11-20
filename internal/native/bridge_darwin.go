package native

/*
#cgo CFLAGS: -mmacosx-version-min=13.0 -fobjc-arc
#cgo LDFLAGS: -framework AppKit -framework CoreText -framework Carbon -framework UniformTypeIdentifiers -mmacosx-version-min=13.0
#include <stdlib.h>
#include "bridge.h"
*/
import "C"
import (
	"fmt"
	"thief/internal/reader"
	"unsafe"
)

type Event struct{ Kind, Payload string }

var handler func(Event)

func Run(onEvent func(Event)) { handler = onEvent; C.thief_run() }
func Apply(json string)       { p := C.CString(json); defer C.free(unsafe.Pointer(p)); C.thief_apply(p) }

type Measurer struct{}

func (Measurer) Fit(text string, l reader.Layout) (int, error) {
	p := C.CString(text)
	defer C.free(unsafe.Pointer(p))
	n := int(C.thief_fit(p, C.long(len(text)), C.double(l.Width), C.double(l.Height), C.double(l.FontSize), C.double(l.LineSpacing), C.int(boolInt(l.SingleLine))))
	if n < 0 {
		return 0, fmt.Errorf("文字测量失败")
	}
	units := 0
	count := 0
	for _, r := range text {
		size := 1
		if r > 0xffff {
			size = 2
		}
		if units+size > n {
			break
		}
		units += size
		count++
	}
	if units != n {
		return 0, fmt.Errorf("无效 UTF-16 分页边界")
	}
	return count, nil
}
func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
func Chapters(json string) {
	p := C.CString(json)
	defer C.free(unsafe.Pointer(p))
	C.thief_chapters(p)
}
func Quit() { C.thief_quit() }
func Failure(message string) {
	p := C.CString(message)
	defer C.free(unsafe.Pointer(p))
	C.thief_failure(p)
}
func QuitError(message string) {
	p := C.CString(message)
	defer C.free(unsafe.Pointer(p))
	C.thief_quit_error(p)
}

//export thief_event
func thief_event(kind, payload *C.char) {
	if handler != nil {
		handler(Event{C.GoString(kind), C.GoString(payload)})
	}
}
