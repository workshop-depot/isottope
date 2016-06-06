# isottope
Is an Event Loop for [otto](https://github.com/robertkrimen/otto) JavaScript engine, which itself is written in pure Go(ld).

Using *isottope* it's possible to register Go functions as events and when they get triggered - from either Go or inside JavaScript - these Go functions would get called for each subscriber.

Three JavaScript functions added; **subscribe**, **unsubscribe** and **emit**. Events also can get triggered from inside Go.

It's not a fancy one but you can even implement timers just using this pattern - there a sample usage inside test file.

Not used heavily yet (just in one side project of mine), so any bug report is most appreciated.