package machine

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/NeowayLabs/cloud-machine/instance"
	"github.com/NeowayLabs/cloud-machine/volume"
	"gopkg.in/amz.v3/aws"
	"gopkg.in/amz.v3/ec2"
)

const (
	DefaultFormatInstanceImageID = "ami-ed8b90dd"
	DefaultFormatInstanceType    = "t2.micro"
)

var output io.Writer = os.Stderr
var logger = log.New(output, "", 0)

// SetLogger ...
func SetLogger(out io.Writer, prefix string, flag int) {
	output = out
	logger = log.New(out, prefix, flag)
	instance.SetLogger(out, prefix, flag)
	volume.SetLogger(out, prefix, flag)
}

// Machine ...
type Machine struct {
	Instance instance.Instance
	Volumes  []volume.Volume
}

// Get ...
func Get(machine *Machine, auth aws.Auth) error {
	ec2Ref := ec2.New(auth, aws.Regions[machine.Instance.Region], aws.SignV4Factory(machine.Instance.Region, "ec2"))

	// Verify if cloud-config file exists
	if machine.Instance.CloudConfig != "" {
		_, err := os.Stat(machine.Instance.CloudConfig)
		if err != nil {
			return err
		}
	}

	// get list of volumes to format
	volumesToFormat := make([]volume.Volume, 0)
	for key := range machine.Volumes {
		myVolume := &machine.Volumes[key]

		format := false
		if myVolume.ID == "" && myVolume.SnapshotID == "" {
			format = true
		}

		_, err := volume.Get(ec2Ref, myVolume)
		if err != nil {
			return err
		}

		if format == true {
			volumesToFormat = append(volumesToFormat, *myVolume)
		}

		myVolume.AvailableZone = machine.Instance.AvailableZone
	}

	// Create a machine to format theses volumes
	if len(volumesToFormat) > 0 {
		err := FormatVolumes(ec2Ref, *machine, volumesToFormat)
		if err != nil {
			return err
		}
	}

	_, err := instance.Get(ec2Ref, &machine.Instance)
	if err != nil {
		return err
	}

	err = AttachVolumes(ec2Ref, machine.Instance.ID, machine.Volumes)
	if err != nil {
		return err
	}

	err = instance.Reboot(ec2Ref, machine.Instance)
	if err != nil {
		return err
	}

	logger.Printf("The instance Id <%s> with IP Address <%s> is running with %d volume(s)!\n", machine.Instance.ID, machine.Instance.PrivateIPAddress, len(machine.Volumes))

	return nil
}

// AttachVolumes ...
func AttachVolumes(ec2Ref *ec2.EC2, InstanceID string, volumes []volume.Volume) error {
	for _, myVolume := range volumes {
		_, err := ec2Ref.AttachVolume(myVolume.ID, InstanceID, myVolume.Device)
		if err != nil {
			reqError := err.(*ec2.Error)
			if reqError.Code != "VolumeInUse" {
				return err
			}
		}
	}

	return nil
}

// FormatVolumes ...
func FormatVolumes(ec2Ref *ec2.EC2, machine Machine, volumes []volume.Volume) error {
	err := os.Mkdir("cloud-config", 0755)
	if os.IsPermission(err) == true {
		return err
	}

	name := machine.Instance.Name + "-format-volumes"
	cloudConfigName := fmt.Sprintf("cloud-config/%s.yml", name)

	// create specific cloud config to format volumes
	var units string
	for _, myVolume := range volumes {
		units += getFormatAndMountUnit(myVolume)
	}

	err = ioutil.WriteFile(cloudConfigName, []byte(getFormatCloudConfig(units)), 0644)
	if err != nil {
		return err
	}

	formatInstance := instance.Instance{
		Name:             name,
		CloudConfig:      cloudConfigName,
		ImageID:          DefaultFormatInstanceImageID,
		Type:             DefaultFormatInstanceType,
		KeyName:          machine.Instance.KeyName,
		SecurityGroups:   machine.Instance.SecurityGroups,
		SubnetID:         machine.Instance.SubnetID,
		AvailableZone:    machine.Instance.AvailableZone,
		ShutdownBehavior: "terminate",
	}

	_, err = instance.Get(ec2Ref, &formatInstance)
	if err != nil {
		return err
	}

	err = AttachVolumes(ec2Ref, formatInstance.ID, volumes)
	if err != nil {
		return err
	}

	err = instance.Reboot(ec2Ref, formatInstance)
	if err != nil {
		return err
	}

	logger.Printf("Waiting while %d volumes was formating...\n", len(volumes))
	err = instance.WaitUntilState(ec2Ref, &formatInstance, "terminated")
	logger.Println("")
	if err != nil {
		return err
	}

	return nil
}

func getFormatAndMountUnit(myVolume volume.Volume) string {
	mountUnitName := strings.Replace(strings.Trim(myVolume.Mount, "/"), "/", "-", -1)
	return fmt.Sprintf(`
    - name: format-%[1]s.service
      command: start
      content: |
        [Unit]
        Description=Formats %[1]s drive
        [Service]
        Type=oneshot
        RemainAfterExit=yes
        ExecStart=/usr/sbin/wipefs -f %[2]s
        ExecStart=/usr/sbin/mkfs.%[3]s %[2]s
    - name: %[5]s.mount
      command: start
      content: |
        [Unit]
        Description=Mount %[1]s drive to %[4]s
        Requires=format-%[1]s.service
        Before=shutdown.service
        After=format-%[1]s.service
        [Mount]
        What=%[2]s
        Where=%[4]s
        Type=%[3]s
        Options=defaults,noatime,noexec,nobarrier`, myVolume.Name, myVolume.Device, myVolume.FileSystem, myVolume.Mount, mountUnitName)
}

func getFormatCloudConfig(units string) string {
	return fmt.Sprintf(`#cloud-config

coreos:
  units:%s
    - name: shutdown.service
      command: start
      content: |
        [Unit]
        Description=Shutdown instance after format and mount all volumes
        [Service]
        Type=oneshot
        ExecStart=/usr/sbin/shutdown -h now
    - name: etcd.service
      mask: true
    - name: fleet.service
      mask: true
    - name: docker.service
      mask: true
  update:
      group: stable
      reboot-strategy: off`, units)
}
