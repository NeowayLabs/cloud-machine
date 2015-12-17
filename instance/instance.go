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
	Tags                 []ec2.Tag
	ec2.Instance
}

func mergeInstances(instance *Instance, instanceRef *ec2.Instance) {
	instance.Instance = *instanceRef
	// Instance struct has some fields that is present in ec2.Instance
	// We should rewrite this fields
	instance.ID = instanceRef.InstanceId
	instance.Type = instanceRef.InstanceType
	instance.ImageID = instanceRef.ImageId
	instance.SubnetID = instanceRef.SubnetId
	instance.KeyName = instanceRef.KeyName
	instance.AvailableZone = instanceRef.AvailZone
	instance.EBSOptimized = instanceRef.EBSOptimized
	instance.SecurityGroups = make([]string, len(instanceRef.SecurityGroups))

	for i, securityGroup := range instanceRef.SecurityGroups {
		instance.SecurityGroups[i] = securityGroup.Id
	}

	instance.Tags = make([]ec2.Tag, 0)
	for _, tag := range instanceRef.Tags {
		if tag.Key == "Name" {
			instance.Name = tag.Value
		} else {
			instance.Tags = append(instance.Tags, tag)
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
func Get(ec2Ref *ec2.EC2, instance *Instance) (ec2.Instance, error) {
	var instanceRef ec2.Instance
	var err error
	if instance.ID == "" {
		logger.Printf("Creating new instance...\n")
		instanceRef, err = Create(ec2Ref, instance)
		logger.Printf("--------- NEW INSTANCE ---------\n")
	} else {
		logger.Printf("Loading instance Id <%s>...\n", instance.ID)
		instanceRef, err = Load(ec2Ref, instance)
		logger.Printf("--------- LOADING INSTANCE ---------\n")
	}

	if err != nil {
		return instanceRef, err
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
	logger.Println("----------------------------------\n")

	return instanceRef, nil
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

	instanceRef := resp.Reservations[0].Instances[0]
	mergeInstances(instance, &instanceRef)

	return instanceRef, nil
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

	instanceRef := resp.Instances[0]
	tags := append(instance.Tags, ec2.Tag{"Name", instance.Name})
	_, err = ec2Ref.CreateTags([]string{instanceRef.InstanceId}, tags)
	if err != nil {
		return ec2.Instance{}, err
	}

	mergeInstances(instance, &instanceRef)

	err = WaitUntilState(ec2Ref, instance, "running")
	if err != nil {
		return ec2.Instance{}, err
	}

	return instanceRef, nil
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
