package main

import (
 "runtime"
 "thief/internal/native"
)

func main() {
 runtime.LockOSThread()
 native.Run(func(e native.Event) {})
}
