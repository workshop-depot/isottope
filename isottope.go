package isottope

import (
	"time"

	"github.com/robertkrimen/otto"
	"github.com/rs/xid"
)

//SimpleEvent is a default implementation of Event, with no Pre or Post functionality, serves as a normal event
type SimpleEvent string

//GetID implements Event
func (e SimpleEvent) GetID() string {
	return string(e)
}

//Pre implements Event
func (e SimpleEvent) Pre(bool) func(...interface{}) []interface{} { return nil }

//Post implements Event
func (e SimpleEvent) Post(bool) func(...interface{}) bool { return nil }

//nop provides a default no operation Go func for the Pre part of the action of the event
func nop(args ...interface{}) []interface{} {
	return args
}

func (loop *EventLoop) start(started chan struct{}) {
	close(started)

	vm := loop.vm
MAIN_LOOP:
	for {
		select {
		case evr := <-loop.registerEvent:
			eid := evr.event.GetID()
			if len(eid) == 0 {
				continue MAIN_LOOP
			}
			_, ok := loop.events[eid]
			if !ok {
				loop.events[eid] = &eventSubscription{
					evr.event,
					nil,
				}
			}
			if evr.done != nil {
				close(evr.done)
			}
		case e := <-loop.eventChannel:
			event, ok := loop.events[e.eventName]
			if !ok {
				continue MAIN_LOOP
			}

			_event := event.event
			shouldRemoveEvent := false
			if len(event.subscribers) == 0 {
				shouldRemoveEvent = execEvent(vm, true, nil, _event, nil)
			} else {
				if len(e.arguments) > 0 {
					for _, v := range event.subscribers {
						_args := e.arguments

						shouldRemoveEvent = execEvent(vm, false, &v, _event, _args)
					}
				} else {
					for _, v := range event.subscribers {
						_args := v.arguments

						shouldRemoveEvent = execEvent(vm, false, &v, _event, _args)
					}
				}
			}

			if shouldRemoveEvent {
				delete(loop.events, e.eventName)
			}
		}
	}
}

func execEvent(vm *otto.Otto, subscribersDepleted bool, v *subscription, event Event, args []interface{}) (unregisterEvent bool) {
	unregisterEvent = false
	_pre := event.Pre(subscribersDepleted)
	if _pre == nil {
		_pre = nop
	}
	resArg := _pre(args...)
	defer func() {
		_post := event.Post(subscribersDepleted)
		if _post != nil {
			unregisterEvent = _post(args...)
		}
	}()

	if v == nil {
		return
	}

	if len(v.calleeJS) > 0 {
		chain := []interface{}{}
		chain = append(chain, v.calleeJS...)
		chain = append(chain, resArg...)

		_, err := vm.Call(`Function.call.call`, nil, chain...)
		if err != nil {
			//TODO:
		}
	}

	if v.calleeGo != nil {
		v.calleeGo(resArg...)
	}

	return
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

		ef.subscribers = append(ef.subscribers, subscription{eventArguments, targetArguments, nil})
		r.result = len(ef.subscribers) - 1

		return
	})
}

type actionResult struct {
	err    error
	result interface{}
}

// Init creates & initializes a new EventLoop;
// it starts a go routine
// TODO: stop it
func Init(vm *otto.Otto) *EventLoop {
	loop := new(EventLoop)
	loop.vm = vm
	loop.eventChannel = make(chan emitted)
	loop.events = make(map[string]*eventSubscription)
	loop.registerEvent = make(chan *eventRegistration)

	loop.defineSubscribe()
	loop.defineEmit()
	loop.defineUnsubscribe()
	loop.defineTimer()
	started := make(chan struct{})
	go loop.start(started)
	<-started

	return loop
}

// Emit emits an event
func (loop *EventLoop) Emit(name string, arguments ...interface{}) {
	loop.eventChannel <- emitted{name, arguments}
}

//RegisterEvent registers an event
func (loop *EventLoop) RegisterEvent(e Event, ensure bool) {
	ereg := new(eventRegistration)
	ereg.event = e
	if ensure {
		ereg.done = make(chan struct{})
	}
	loop.registerEvent <- ereg
	if ensure {
		<-ereg.done
	}
}

// EventLoop creates an event loop for otto
type EventLoop struct {
	registerEvent chan *eventRegistration
	eventChannel  chan emitted
	events        map[string]*eventSubscription
	vm            *otto.Otto
}

type eventRegistration struct {
	event Event
	done  chan struct{}
}

type emitted struct {
	eventName string

	arguments []interface{}
}

type eventSubscription struct {
	event       Event
	subscribers []subscription
}

type subscription struct {
	arguments []interface{}

	// js function to get called
	calleeJS []interface{}
	calleeGo func(...interface{})
}

//Event is a protocol that an event context must follow
type Event interface {
	GetID() string
	Pre(bool) func(...interface{}) []interface{}
	Post(bool) func(...interface{}) bool
}

//Error constant error based on http://dave.cheney.net/2016/04/07/constant-errors
type Error string

func (e Error) Error() string { return string(e) }

//-----------------------------------------------------------------------------
// timer stuff

func (loop *EventLoop) defineTimer() {
	vm := loop.vm

	vm.Set("createTimeouts", func(call otto.FunctionCall) (result otto.Value) {
		isInterval, err := call.Argument(0).ToBoolean()
		if err != nil {
			isInterval = false
		}
		delay, err := call.Argument(1).ToInteger()
		if err != nil {
			delay = 1000
		}

		t := &timerEvent{}
		t.duration = time.Duration(delay) * time.Millisecond
		t.id = xid.New().String()
		t.timer = time.AfterFunc(time.Second, func() {
			loop.Emit(t.id)
		})
		t.interval = isInterval

		loop.RegisterEvent(t, true)

		_id, _ := otto.ToValue(t.id)
		return _id
	})

	vm.Run(`
		function __interval__() {				
			var delay = arguments[0];
			if (!delay) {
				delay = 1000;
			}
			var isInterval = arguments[1] | false;			
			
			var eventID = createTimeouts(isInterval, delay);
			return eventID;			
		}

		function setTimeout(f, delay) {
			var __id = __interval__(delay, false);
			
			var args = [];
			args.push(__id);
			args.push(f)
			var i = 0;
			for (i = 2; i < arguments.length; i++) {
				args.push(arguments[i]);
			}
			
			var rsub = subscribe.apply(null, args);
			return {eventID: __id, result: rsub.result};
		}

		function setInterval(f, delay) {
			var __id = __interval__(delay, true);
			
			var args = [];
			args.push(__id);
			args.push(f)
			var i = 0;
			for (i = 2; i < arguments.length; i++) {
				args.push(arguments[i]);
			}
			
			var rsub = subscribe.apply(null, args);
			return {eventID: __id, result: rsub.result};
		}

		function clearTimeout(timer) {
			return unsubscribe(timer.eventID, timer.result);
		}
		
		function clearInterval(timer) {
			return unsubscribe(timer.eventID, timer.result);
		}
	`)
}

type timerEvent struct {
	id       string
	timer    *time.Timer
	duration time.Duration
	interval bool
}

//GetID implements Event
func (e *timerEvent) GetID() string {
	return e.id
}

//Pre implements Event
func (e *timerEvent) Pre(bool) func(...interface{}) []interface{} { return nil }

//Post implements Event
func (e *timerEvent) Post(subscribersDepleted bool) func(...interface{}) bool {
	return func(...interface{}) bool {
		if subscribersDepleted {
			return true
		}

		if e.interval {
			e.timer.Reset(e.duration)
			return false
		}

		return true
	}
}
