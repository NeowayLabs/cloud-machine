package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/NeowayLabs/cloud-machine/machine"
	"github.com/NeowayLabs/cloud-machine/volume"
	"github.com/NeowayLabs/logger"
	"gopkg.in/yaml.v2"
)

type (
	// Clusters ...
	Clusters struct {
		Default  Default
		Clusters []struct {
			Machine string
			Nodes   int
		}
	}

	// Cluster ...
	Cluster struct {
		Machine machine.Machine
		Nodes   int
	}

	// Default ...
	Default struct {
		ImageID              string
		Region               string
		KeyName              string
		SecurityGroups       []string
		SubnetID             string
		DefaultAvailableZone string
	}
)

func main() {
	flag.Parse()

	clusterFile := flag.Arg(0)
	if clusterFile == "" {
		logger.Fatal("You need to pass the cluster file, type: %s <cluster-file.yml>\n", os.Args[0])
	}

	clusterContent, err := ioutil.ReadFile(clusterFile)
	if err != nil {
		logger.Fatal("Error open cluster file: %s", err.Error())
	}

	var clusters Clusters
	err = yaml.Unmarshal(clusterContent, &clusters)
	if err != nil {
		logger.Fatal("Error reading cluster file: %s", err.Error())
	}

	// First verify if I can open all machine files
	machines := make([]Cluster, len(clusters.Clusters))
	for key := range clusters.Clusters {
		myCluster := &clusters.Clusters[key]

		machineContent, err := ioutil.ReadFile(myCluster.Machine)
		if err != nil {
			logger.Fatal("Error open machine file: %s", err.Error())
		}

		var myMachine machine.Machine
		err = yaml.Unmarshal(machineContent, &myMachine)
		if err != nil {
			logger.Fatal("Error reading machine file: %s", err.Error())
		}

		// Verify if cloud-config file exists
		if myMachine.Instance.CloudConfig != "" {
			_, err := os.Stat(myMachine.Instance.CloudConfig)
			if err != nil {
				logger.Fatal("Error reading cloud-config: %s", err.Error())
			}
		}

		// Set default values of cluster to machine
		if myMachine.Instance.ImageID == "" {
			myMachine.Instance.ImageID = clusters.Default.ImageID
		}
		if myMachine.Instance.Region == "" {
			myMachine.Instance.Region = clusters.Default.Region
		}
		if myMachine.Instance.KeyName == "" {
			myMachine.Instance.KeyName = clusters.Default.KeyName
		}
		if len(myMachine.Instance.SecurityGroups) == 0 {
			myMachine.Instance.SecurityGroups = clusters.Default.SecurityGroups
		}
		if myMachine.Instance.SubnetID == "" {
			myMachine.Instance.SubnetID = clusters.Default.SubnetID
		}
		if myMachine.Instance.DefaultAvailableZone == "" {
			myMachine.Instance.DefaultAvailableZone = clusters.Default.DefaultAvailableZone
		}

		machines[key] = Cluster{Machine: myMachine, Nodes: myCluster.Nodes}
	}

	auth, err := AwsAuth()
	if err != nil {
		logger.Fatal("Error reading aws credentials: %s", err.Error())
	}

	machine.SetLogger(ioutil.Discard, "", 0)

	for key, myCluster := range machines {
		fmt.Printf("================ Running machines of %d. cluster ================\n", key+1)

		for i := 1; i <= myCluster.Nodes; i++ {
			myMachine := myCluster.Machine
			myMachine.Volumes = make([]volume.Volume, len(myMachine.Volumes))

			// append machine number to name of instance
			myMachine.Instance.Name += fmt.Sprintf("-%d", i)

			// append machine number to name of volume
			for key := range myCluster.Machine.Volumes {
				referenceVolume := &myCluster.Machine.Volumes[key]

				myVolume := *referenceVolume
				myVolume.Name += fmt.Sprintf("-%d", i)
				myMachine.Volumes[key] = myVolume
			}

			fmt.Printf("Running machine: %s\n", myMachine.Instance.Name)
			err = machine.Get(&myMachine, auth)
			if err != nil {
				logger.Fatal("Error getting machine: %s", err.Error())
			}
			fmt.Printf("Machine Id <%s>, IP Address <%s>\n", myMachine.Instance.ID, myMachine.Instance.PrivateIPAddress)
			if i < myCluster.Nodes {
				fmt.Println("----------------------------------")
			}
		}
	}
	fmt.Println("================================================================")
}
