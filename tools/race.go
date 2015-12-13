package main

import (
	"fmt"
	"github.com/aryszka/cast"
)

func main() {
	r := cast.NewRace(10000)
	r.Dispatcher.Subscribe(&cast.Listener{func(m *cast.Message) {
		fmt.Printf("%s = %s\n", m.Key, m.Content)
	}})
	r.Start()
}
