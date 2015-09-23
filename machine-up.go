package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/NeowayLabs/cloud-machine/machine"
	"gopkg.in/yaml.v2"
)

func main() {
	flag.Parse()

	machineFile := flag.Arg(0)
	if machineFile == "" {
		fmt.Printf("You need to pass a machine definition file, type: %s <machine.yml>\n", os.Args[0])
		return
	}

	machineContent, err := ioutil.ReadFile(machineFile)
	if err != nil {
		panic(err.Error())
	}

	var myMachine machine.Machine
	err = yaml.Unmarshal(machineContent, &myMachine)
	if err != nil {
		panic(err.Error())
	}

	auth, err := AwsAuth()
	if err != nil {
		panic(err.Error())
	}

	err = machine.Get(&myMachine, auth)
	if err != nil {
		panic(err.Error())
	}
}
