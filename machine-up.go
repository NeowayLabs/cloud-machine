package main

import (
	"flag"
	"io/ioutil"
	"os"

	"github.com/NeowayLabs/cloud-machine/machine"
	"github.com/NeowayLabs/logger"
	"gopkg.in/yaml.v2"
)

func main() {
	flag.Parse()

	machineFile := flag.Arg(0)
	if machineFile == "" {
		logger.Fatal("You need to pass a machine definition file, type: %s <machine.yml>\n", os.Args[0])
	}

	machineContent, err := ioutil.ReadFile(machineFile)
	if err != nil {
		logger.Fatal("Error open machine file: %s", err.Error())
	}

	var myMachine machine.Machine
	err = yaml.Unmarshal(machineContent, &myMachine)
	if err != nil {
		logger.Fatal("Error reading machine file: %s", err.Error())
	}

	auth, err := AwsAuth()
	if err != nil {
		logger.Fatal("Error reading aws credentials: %s", err.Error())
	}

	err = machine.Get(&myMachine, auth)
	if err != nil {
		logger.Fatal("Error getting machine: %s", err.Error())
	}
}
