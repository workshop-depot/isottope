package main

import (
	"dc0d-github/isottope"
	"time"

	"github.com/robertkrimen/otto"
)

func main() {
	vm := otto.New()
	loop := isottope.Init(vm)

	timer(vm)
	customEvent(vm, loop)

	//the timer itself is implemented using isottope.Event interface

	<-time.After(time.Second * 5)
}

func customEvent(vm *otto.Otto, loop *isottope.EventLoop) {
	loop.RegisterEvent(isottope.SimpleEvent(`TEST_OK`), true)

	vm.Run(`
		function target() {
			var args = ['input args:'];
			if (arguments.length > 0) {
				for (i = 0; i < arguments.length; i++) {
					args.push(arguments[i]);
				}
			}
			var msg = args.join(' ');
			console.log(msg);
		}

		subscribe('TEST_OK', target, 'DEFAULT-ARG-1', 'DEFAULT-ARG-2');

		emit('TEST_OK', "Custome Arg");	
		emit('TEST_OK');	
	`)

	loop.Emit(`TEST_OK`, `Another Custome Arg`)
	loop.Emit(`TEST_OK`)
}

func timer(vm *otto.Otto) {
	vm.Run(`
		var R = {};
		var c = 0;
		function target() {
			c++;
			console.log('registrar #' + c);
			if (c < 5) {
				return;
			}
			clearTimeout(R);
		}

		try {
			R = setInterval(target, 200);
		}
		catch(e) {
			console.error(e);
		}		
	`)
}
