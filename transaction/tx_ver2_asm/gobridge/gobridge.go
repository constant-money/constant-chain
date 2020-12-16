//+ build js,wasm

package gobridge

import (
	"syscall/js"
)

var bridgeRoot js.Value

const (
	bridgeJavaScriptName = "__gobridge__"
)

func registrationWrapper(fn func(this js.Value, args []js.Value) (interface{}, error)) func(this js.Value, args []js.Value) interface{} {
	return func(this js.Value, args []js.Value) interface{} {
		cb := args[len(args)-1]

		ret, err := fn(this, args[:len(args)-1])

		if err != nil {
			cb.Invoke(err.Error(), js.Null())
		} else {
			cb.Invoke(js.Null(), ret)
		}

		return ret
	}
}

// RegisterCallback registers a Go function to be a callback used in JavaScript
func RegisterCallback(name string, callback func(this js.Value, args []js.Value) (interface{}, error)) {
	bridgeRoot.Set(name, js.FuncOf(registrationWrapper(callback)))
}

// RegisterValue registers a static value output from Go for access in JavaScript
func RegisterValue(name string, value interface{}) {
	bridgeRoot.Set(name, value)
}

func init() {
	global := js.Global()

	bridgeRoot = global.Get(bridgeJavaScriptName)
}
