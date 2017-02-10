package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/NeowayLabs/cloud-machine/auth"
	"github.com/NeowayLabs/cloud-machine/machine"
	"github.com/NeowayLabs/cloud-machine/volume"
	"github.com/NeowayLabs/logger"
	"gopkg.in/amz.v3/ec2"
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
		AvailableZone        string
		DefaultAvailableZone string // backward compatibility, use availablezone instead
		Tags                 []ec2.Tag
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

	if clusters.Default.AvailableZone == "" {
		if clusters.Default.DefaultAvailableZone != "" {
			clusters.Default.AvailableZone = clusters.Default.DefaultAvailableZone
		}
	}

	// First verify if I can open all machine files
	machines := make([]Cluster, len(clusters.Clusters))
	for key := range clusters.Clusters {
		clusterConfig := &clusters.Clusters[key]

		machineContent, err := ioutil.ReadFile(clusterConfig.Machine)
		if err != nil {
			logger.Fatal("Error open machine file: %s", err.Error())
		}

		var machineConfig machine.Machine
		err = yaml.Unmarshal(machineContent, &machineConfig)
		if err != nil {
			logger.Fatal("Error reading machine file: %s", err.Error())
		}

		// Verify if cloud-config file exists
		if machineConfig.Instance.CloudConfig != "" {
			_, err := os.Stat(machineConfig.Instance.CloudConfig)
			if err != nil {
				logger.Fatal("Error reading cloud-config: %s", err.Error())
			}
		}

		// Set default values of cluster to machine
		if machineConfig.Instance.ImageID == "" {
			machineConfig.Instance.ImageID = clusters.Default.ImageID
		}
		if machineConfig.Instance.Region == "" {
			machineConfig.Instance.Region = clusters.Default.Region
		}
		if machineConfig.Instance.KeyName == "" {
			machineConfig.Instance.KeyName = clusters.Default.KeyName
		}
		if len(machineConfig.Instance.SecurityGroups) == 0 {
			machineConfig.Instance.SecurityGroups = clusters.Default.SecurityGroups
		}
		if machineConfig.Instance.SubnetID == "" {
			machineConfig.Instance.SubnetID = clusters.Default.SubnetID
		}

		if machineConfig.Instance.AvailableZone == "" {
			if machineConfig.Instance.DefaultAvailableZone != "" {
				machineConfig.Instance.AvailableZone = machineConfig.Instance.DefaultAvailableZone
			} else {
				machineConfig.Instance.AvailableZone = clusters.Default.AvailableZone
			}
		}

		for _, tag := range clusters.Default.Tags {
			addTag := true
			for _, instanceTag := range machineConfig.Instance.Tags {
				if strings.EqualFold(instanceTag.Key, tag.Key) {
					addTag = false
				}
			}

			if addTag {
				machineConfig.Instance.Tags = append(machineConfig.Instance.Tags, tag)
			}

			addTag = true
			for k, volume := range machineConfig.Volumes {
				for _, volumeTag := range volume.Tags {
					if strings.EqualFold(volumeTag.Key, tag.Key) {
						addTag = false
					}

				}

				if addTag {
					machineConfig.Volumes[k].Tags = append(machineConfig.Volumes[k].Tags, tag)
				}
			}
		}

		machines[key] = Cluster{Machine: machineConfig, Nodes: clusterConfig.Nodes}
	}

	auth, err := auth.Aws()
	if err != nil {
		logger.Fatal("Error reading aws credentials: %s", err.Error())
	}

	machine.SetLogger(ioutil.Discard, "", 0)

	for key, clusterConfig := range machines {
		fmt.Printf("================ Running machines of %d. cluster ================\n", key+1)

		for i := 1; i <= clusterConfig.Nodes; i++ {
			machineConfig := clusterConfig.Machine
			machineConfig.Volumes = make([]volume.Volume, len(machineConfig.Volumes))

			// append machine number to name of instance
			machineConfig.Instance.Name += fmt.Sprintf("-%d", i)

			// append machine number to name of volume
			for key := range clusterConfig.Machine.Volumes {
				volumeRef := &clusterConfig.Machine.Volumes[key]

				volumeConfig := *volumeRef
				volumeConfig.Name += fmt.Sprintf("-%d", i)
				machineConfig.Volumes[key] = volumeConfig
			}

			fmt.Printf("Running machine: %s\n", machineConfig.Instance.Name)
			err = machine.Get(&machineConfig, auth)
			if err != nil {
				logger.Fatal("Error getting machine: %s", err.Error())
			}

			fmt.Printf("Machine Id <%s>, IP Address <%s>\n", machineConfig.Instance.ID, machineConfig.Instance.PrivateIPAddress)
			if i < clusterConfig.Nodes {
				fmt.Println("----------------------------------")
			}
		}
	}
	fmt.Println("================================================================")
}
