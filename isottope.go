package isottope

import "github.com/robertkrimen/otto"

// Init creates & initializes a new EventLoop;
// it starts a go routine
// TODO: stop it
func Init(vm *otto.Otto) *EventLoop {
	loop := new(EventLoop)
	loop.vm = vm
	loop.eventChannel = make(chan emitted)
	loop.events = make(map[string]*eventSubscription)
	loop.registerEvent = make(chan eventRegistration)

	loop.defineSubscribe()
	loop.defineEmit()
	loop.defineUnsubscribe()
	go loop.start()

	return loop
}

// Emit emits an event
func (loop *EventLoop) Emit(name string, arguments ...interface{}) {
	loop.eventChannel <- emitted{name, arguments}
}

//RegisterEvent registers a go func as an event
func (loop *EventLoop) RegisterEvent(name string, goFunc func(...interface{}) []interface{}) {
	loop.registerEvent <- eventRegistration{
		name,
		goFunc,
	}
}

func (loop *EventLoop) start() {
	vm := loop.vm
	for {
		select {
		case eReg := <-loop.registerEvent:
			if eReg.goFunc != nil && len(eReg.name) > 0 {
				loop.events[eReg.name] = &eventSubscription{
					eReg.goFunc,
					nil,
				}
			}
		case e := <-loop.eventChannel:
			ef, ok := loop.events[e.eventName]
			if ok {
				if len(e.eventArguments) > 0 {
					for _, v := range ef.subscribers {
						resArg := ef.goFunc(e.eventArguments...)

						chain := []interface{}{}
						chain = append(chain, v.targetArguments...)
						chain = append(chain, resArg...)

						_, err := vm.Call(`Function.call.call`, nil, chain...)
						if err != nil {
							//TODO:
						}
					}
				} else {
					for _, v := range ef.subscribers {
						resArg := ef.goFunc(v.eventArguments...)

						chain := []interface{}{}
						chain = append(chain, v.targetArguments...)
						chain = append(chain, resArg...)

						_, err := vm.Call(`Function.call.call`, nil, chain...)
						if err != nil {
							//TODO:
						}
					}
				}
			}
		}
	}
}

func (loop *EventLoop) defineUnsubscribe() {
	// unsubscribe('eventName', code)
	vm := loop.vm

	vm.Set("unsubscribe", func(call otto.FunctionCall) (result otto.Value) {
		var r actionResult
		defer func() {
			var err error

			buffer := make(map[string]interface{})
			buffer["err"] = r.err
			buffer["result"] = r.result

			result, err = vm.ToValue(buffer)
			if err != nil {
				//TODO:
			}
		}()

		eventName, err := call.Argument(0).ToString()
		if err != nil {
			r.err = err
			return
		}

		ef, ok := loop.events[eventName]
		if !ok {
			r.err = Error(`event does not exist: ` + eventName)
			return
		}

		_index, err := call.Argument(1).ToInteger()
		if err != nil {
			r.err = err
			return
		}

		index := int(_index)
		if index >= len(ef.subscribers) || index < 0 {
			r.err = Error(`subscription index out of range`)
			return
		}

		ef.subscribers = append(ef.subscribers[:index], ef.subscribers[index+1:]...)
		r.result = `unsubscribed`

		return
	})
}

func (loop *EventLoop) defineEmit() {
	// emit('sampleEvent', "Arg", 10.01, 3);
	vm := loop.vm

	vm.Set("emit", func(call otto.FunctionCall) (result otto.Value) {
		var r actionResult
		defer func() {
			var err error

			buffer := make(map[string]interface{})
			buffer["err"] = r.err
			buffer["result"] = r.result

			result, err = vm.ToValue(buffer)
			if err != nil {
				//TODO:
			}
		}()

		eventName, err := call.Argument(0).ToString()
		if err != nil {
			r.err = err
			return
		}

		_, ok := loop.events[eventName]
		if !ok {
			r.err = Error(`event does not exist: ` + eventName)
			return
		}

		eventArguments := []interface{}{}
		if len(call.ArgumentList) > 1 {
			for _, v := range call.ArgumentList[1:] {
				eventArguments = append(eventArguments, v)
			}
		}

		go func() {
			loop.eventChannel <- emitted{eventName, eventArguments}
		}()

		return
	})
}

func (loop *EventLoop) defineSubscribe() {
	//subscribe('eventName', function(a,b,c){}, eventArg1, eventArg2)
	vm := loop.vm

	vm.Set("subscribe", func(call otto.FunctionCall) (result otto.Value) {
		var r actionResult
		defer func() {
			var err error

			buffer := make(map[string]interface{})
			buffer["err"] = r.err
			buffer["result"] = r.result

			result, err = vm.ToValue(buffer)
			if err != nil {
				//TODO:
			}
		}()

		eventName, err := call.Argument(0).ToString()
		if err != nil {
			r.err = err
			return
		}

		ef, ok := loop.events[eventName]
		if !ok {
			r.err = Error(`event does not exist: ` + eventName)
			return
		}

		targetArguments := []interface{}{}
		targetArguments = append(targetArguments, call.Argument(1))
		targetArguments = append(targetArguments, nil)

		eventArguments := []interface{}{}
		if len(call.ArgumentList) > 2 {
			// eventArguments = append(eventArguments, call.ArgumentList[2:]...)
			for _, v := range call.ArgumentList[2:] {
				eventArguments = append(eventArguments, v)
			}
		}

		ef.subscribers = append(ef.subscribers, subscription{eventArguments, targetArguments})
		r.result = len(ef.subscribers) - 1

		return
	})
}

// EventLoop creates an event loop for otto
type EventLoop struct {
	registerEvent chan eventRegistration
	eventChannel  chan emitted
	events        map[string]*eventSubscription
	vm            *otto.Otto
}

type eventRegistration struct {
	name   string
	goFunc func(...interface{}) []interface{}
}

type eventSubscription struct {
	goFunc      func(...interface{}) []interface{}
	subscribers []subscription
}

type subscription struct {
	// event settings registered input values
	eventArguments []interface{}

	// js function to get called
	targetArguments []interface{}
}

type actionResult struct {
	err    error
	result interface{}
}

type emitted struct {
	eventName string

	// used if it's an event comes from `emit` in js,
	// otherwise the registered input values would be used
	eventArguments []interface{}
}

//Error constant error based on http://dave.cheney.net/2016/04/07/constant-errors
type Error string

func (e Error) Error() string { return string(e) }
