package isottope

import (
	"testing"
	"time"

	"github.com/robertkrimen/otto"
)

func TestSmokeTest(t *testing.T) {
	vm := otto.New()
	loop := Init(vm)
	loop.RegisterEvent("sampleEvent", func(args ...interface{}) []interface{} {
		var result []interface{}
		result = append(result, `INSIDE GO`)
		result = append(result, args...)
		return result
	})

	vm.Run(`
		var r1 = subscribe('sampleEvent', function() {
			var name = 'FIRST CALLBACK';
			console.log(name);
			if (arguments.length > 0) {
				console.log("<<");
				for (i = 0; i < arguments.length; i++) {
					console.log(arguments[i]);
				}
				console.log(">>");
			}
			return "JS Called: " + name;
		}, 'DEFAULT ARG 1');

		var c = -1;
		var r2 = {};
		r2 = subscribe('sampleEvent', function() {
			var name = 'SECOND CALLBACK';
			console.log(name);
			if (arguments.length > 0) {
				console.log("<<");
				for (i = 0; i < arguments.length; i++) {
					console.log(arguments[i]);
				}
				console.log(">>");
			}
			
			c++;
			console.log('<<CALL COUNTER: ' + c + '>>')
			if (r2.result && c > 0) {			
				var ru = unsubscribe('sampleEvent', r2.result);
				console.log('GOT UNSUBSCRIBED:' + JSON.stringify(ru));
				emit('sampleEvent', "UNSUBSCRIBED", name);
			}
			
			return "JS Called: " + name;
		}, 'DEFAULT ARG 1');
	
		emit('sampleEvent', "Custome Arg");
	`)

	loop.Emit("sampleEvent", `Emit From Go`)
	loop.Emit("sampleEvent")
	go func() {
		tick := time.Tick(time.Second)
		for {
			t := <-tick
			loop.Emit("sampleEvent", `AS A TIMER`, t.Second())
		}
	}()

	<-time.After(time.Second * 3)
}
