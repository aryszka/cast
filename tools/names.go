package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	fmt.Print(`
        package cast

        // generated names
        var randomNames []string = []string{
        `)

	n := "120"
	if len(os.Args) > 1 {
		n = os.Args[1]
	}

	rsp, err := http.Get("http://api.uinames.com/?amount=" + n)
	if err != nil {
		log.Fatal(err)
	}

	defer rsp.Body.Close()

	d := json.NewDecoder(rsp.Body)
	var names []map[string]string
	err = d.Decode(&names)
	if err != nil {
		log.Fatal(err)
	}

	for _, n := range names {
		fmt.Printf("\"%s %s\",\n", n["surname"], n["name"])
	}

	fmt.Print("}")
}
