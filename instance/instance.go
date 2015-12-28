package instance

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"text/template"
	"time"

	"gopkg.in/amz.v3/ec2"
)

var loggerOutput io.Writer = os.Stderr
var logger = log.New(loggerOutput, "", 0)

// SetLogger ...
func SetLogger(out io.Writer, prefix string, flag int) {
	loggerOutput = out
	logger = log.New(out, prefix, flag)
}

// Instance ...
type Instance struct {
	ID                   string
	Name                 string
	Type                 string
	ImageID              string
	Region               string
	KeyName              string
	SecurityGroups       []string
	SubnetID             string
	DefaultAvailableZone string // backward compatibility, use availablezone instead
	AvailableZone        string
	CloudConfig          string
	EBSOptimized         bool
	ShutdownBehavior     string
	EnableAPITermination bool
	PlacementGroupName   string
	Tags                 []ec2.Tag // ec2.Instance already have this property but yml would need new section
	ec2.Instance
}

func mergeInstances(instance *Instance, ec2Instance *ec2.Instance) {
	instance.Instance = *ec2Instance
	// Instance struct has some fields that is present in ec2.Instance
	// We should rewrite this fields
	instance.ID = ec2Instance.InstanceId
	instance.Type = ec2Instance.InstanceType
	instance.ImageID = ec2Instance.ImageId
	instance.SubnetID = ec2Instance.SubnetId
	instance.KeyName = ec2Instance.KeyName
	instance.AvailableZone = ec2Instance.AvailZone
	instance.EBSOptimized = ec2Instance.EBSOptimized
	instance.SecurityGroups = make([]string, len(ec2Instance.SecurityGroups))

	for i, securityGroup := range ec2Instance.SecurityGroups {
		instance.SecurityGroups[i] = securityGroup.Id
	}

	if len(ec2Instance.Tags) > 0 {
		instance.Tags = make([]ec2.Tag, len(ec2Instance.Tags)-1)
		for i, tag := range ec2Instance.Tags {
			if tag.Key == "Name" {
				instance.Name = tag.Value
			} else {
				instance.Tags[i] = tag
			}
		}
	}
}

// WaitUntilState valid values to state is: pending, running, shutting-down, terminated, stopping, stopped
func WaitUntilState(ec2Ref *ec2.EC2, instance *Instance, state string) error {
	fmt.Fprintf(loggerOutput, "Instance state is <%s>, waiting for <%s>", instance.State.Name, state)
	for {
		fmt.Fprint(loggerOutput, ".")
		if instance.State.Name != state {
			time.Sleep(2 * time.Second)
			_, err := Load(ec2Ref, instance)
			if err != nil {
				fmt.Fprintln(loggerOutput, " [ERROR]")
				return err
			}
		} else {
			fmt.Fprintln(loggerOutput, " [OK]")
			return nil
		}
	}
}

// Get a instance, if Id was not passed a new instance will be created
func Get(ec2Ref *ec2.EC2, instance *Instance) (ec2Instance ec2.Instance, err error) {
	if instance.ID == "" {
		logger.Printf("Creating new instance...\n")
		ec2Instance, err = Create(ec2Ref, instance)
		logger.Printf("--------- NEW INSTANCE ---------\n")
	} else {
		logger.Printf("Loading instance Id <%s>...\n", instance.ID)
		ec2Instance, err = Load(ec2Ref, instance)
		logger.Printf("--------- LOADING INSTANCE ---------\n")
	}

	if err != nil {
		return
	}

	logger.Printf("    Id: %s\n", instance.ID)
	logger.Printf("    Name: %s\n", instance.Name)
	logger.Printf("    Type: %s\n", instance.Type)
	logger.Printf("    Image Id: %s\n", instance.ImageID)
	logger.Printf("    Available Zone: %s\n", instance.AvailableZone)
	logger.Printf("    Key Name: %s\n", instance.KeyName)
	logger.Printf("    Security Groups: %+v\n", instance.SecurityGroups)
	logger.Printf("    PlacementGroupName: %+v\n", instance.PlacementGroupName)
	logger.Printf("    Subnet Id: %s\n", instance.SubnetID)
	logger.Printf("    EBS Optimized: %t\n", instance.EBSOptimized)
	if len(instance.Tags) > 0 {
		logger.Printf("    Tags:\n")
		for _, tag := range instance.Tags {
			logger.Printf("        %s: %s\n", tag.Key, tag.Value)
		}
	}
	logger.Println("----------------------------------\n")

	return
}

// Load a instance passing its Id
func Load(ec2Ref *ec2.EC2, instance *Instance) (ec2.Instance, error) {
	if instance.ID == "" {
		return ec2.Instance{}, errors.New("To load a instance you need to pass its Id")
	}

	resp, err := ec2Ref.Instances([]string{instance.ID}, nil)
	if err != nil {
		return ec2.Instance{}, err
	} else if len(resp.Reservations) == 0 || len(resp.Reservations[0].Instances) == 0 {
		return ec2.Instance{}, fmt.Errorf("Any instance was found with instance Id <%s>", instance.ID)
	}

	ec2Instance := resp.Reservations[0].Instances[0]
	mergeInstances(instance, &ec2Instance)

	return ec2Instance, nil
}

// Create new instance
func Create(ec2Ref *ec2.EC2, instance *Instance) (ec2.Instance, error) {
	options := ec2.RunInstances{
		ImageId:               instance.ImageID,
		InstanceType:          instance.Type,
		KeyName:               instance.KeyName,
		SecurityGroups:        make([]ec2.SecurityGroup, len(instance.SecurityGroups)),
		SubnetId:              instance.SubnetID,
		EBSOptimized:          instance.EBSOptimized,
		DisableAPITermination: !instance.EnableAPITermination,
		AvailZone:             instance.AvailableZone,
	}

	if instance.CloudConfig != "" {
		cloudConfigTemplate, err := ioutil.ReadFile(instance.CloudConfig)
		if err != nil {
			panic(err.Error())
		}

		tpl := template.Must(template.New("cloudConfig").Parse(string(cloudConfigTemplate)))

		cloudConfig := new(bytes.Buffer)
		if err = tpl.Execute(cloudConfig, instance); err != nil {
			panic(err.Error())
		}

		options.UserData = cloudConfig.Bytes()
	}

	if instance.ShutdownBehavior != "" {
		options.ShutdownBehavior = instance.ShutdownBehavior
	}

	if instance.PlacementGroupName != "" {
		options.PlacementGroupName = instance.PlacementGroupName
	}

	for i, securityGroup := range instance.SecurityGroups {
		options.SecurityGroups[i] = ec2.SecurityGroup{Id: securityGroup}
	}

	resp, err := ec2Ref.RunInstances(&options)
	if err != nil {
		return ec2.Instance{}, err
	} else if len(resp.Instances) == 0 {
		return ec2.Instance{}, errors.New("Any instance was created!")
	}

	ec2Instance := resp.Instances[0]
	tags := append(instance.Tags, ec2.Tag{"Name", instance.Name})
	_, err = ec2Ref.CreateTags([]string{ec2Instance.InstanceId}, tags)
	if err != nil {
		return ec2.Instance{}, err
	}

	mergeInstances(instance, &ec2Instance)

	err = WaitUntilState(ec2Ref, instance, "running")
	if err != nil {
		return ec2.Instance{}, err
	}

	return ec2Instance, nil
}

// Terminate ...
func Terminate(ec2Ref *ec2.EC2, instance Instance) error {
	logger.Println("Terminating instance", instance.ID)
	_, err := ec2Ref.TerminateInstances([]string{instance.ID})
	if err == nil {
		logger.Printf("Instance <%s> was destroyed!\n", instance.ID)
	}

	return err
}

// Reboot ...
func Reboot(ec2Ref *ec2.EC2, instance Instance) error {
	logger.Println("Rebooting instance", instance.ID)
	_, err := ec2Ref.RebootInstances(instance.InstanceId)
	return err
}
