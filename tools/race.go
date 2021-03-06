package main

import (
	"fmt"
	"github.com/aryszka/cast"
)

func main() {
	r := cast.NewRace(1000)
	r.Dispatcher.Subscribe(&cast.Listener{cast.HandlerFunc(func(m *cast.Message) {
		fmt.Printf("%s = %s\n", m.Key, m.Content)
	})})
	r.Start()

	<-make(chan struct{})
}
