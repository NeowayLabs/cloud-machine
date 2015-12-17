package volume

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
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

// Volume ...
type Volume struct {
	ID            string
	Name          string
	Type          string
	Size          int
	IOPS          int64
	AvailableZone string
	Device        string
	Mount         string
	FileSystem    string
	SnapshotID    string
	Tags          []ec2.Tag
	ec2.Volume
}

func mergeVolumes(volume *Volume, volumeRef *ec2.Volume) {
	volume.Volume = *volumeRef
	// Volume struct has some fields that is present in ec2.Volume
	// We should rewrite this fields
	volume.ID = volumeRef.Id
	volume.Size = volumeRef.Size
	volume.IOPS = volumeRef.IOPS
	volume.AvailableZone = volumeRef.AvailZone
	volume.Type = volumeRef.VolumeType

	volume.Tags = make([]ec2.Tag, 0)
	for _, tag := range volumeRef.Tags {
		if tag.Key == "Name" {
			volume.Name = tag.Value
		} else {
			volume.Tags = append(volume.Tags, tag)
		}
	}
}

// WaitUntilState valid values to state is: creating, available, in-use,
// deleting, deleted, error
func WaitUntilState(ec2Ref *ec2.EC2, volume *Volume, state string) error {
	fmt.Fprintf(loggerOutput, "Volume status is <%s>, waiting for <%s>", volume.Status, state)

	for {
		fmt.Fprint(loggerOutput, ".")
		if volume.Status != state {
			time.Sleep(2 * time.Second)
			_, err := Load(ec2Ref, volume)
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

// Get a volume, if Id was not passed a new volume will be created
func Get(ec2Ref *ec2.EC2, volume *Volume) (ec2.Volume, error) {
	var volumeRef ec2.Volume
	var err error
	if volume.ID == "" {
		logger.Printf("Creating new volume...\n")
		volumeRef, err = Create(ec2Ref, volume)
		logger.Printf("--------- NEW VOLUME ---------\n")
	} else {
		logger.Printf("Loading volume Id <%s>...\n", volume.ID)
		volumeRef, err = Load(ec2Ref, volume)
		logger.Printf("--------- LOADING VOLUME ---------\n")
	}

	if err != nil {
		return volumeRef, err
	}

	logger.Printf("    Id: %s\n", volume.ID)
	logger.Printf("    Name: %s\n", volume.Name)
	logger.Printf("    Type: %s\n", volume.Type)
	logger.Printf("    Size: %d\n", volume.Size)
	if volume.IOPS > 0 {
		logger.Printf("    IOPS: %d\n", volume.IOPS)
	}
	logger.Printf("    Available Zone: %s\n", volume.AvailableZone)
	logger.Printf("    Device: %s\n", volume.Device)
	logger.Printf("    Mount: %s\n", volume.Mount)
	logger.Printf("    File System: %s\n", volume.FileSystem)
	logger.Println("----------------------------------\n")

	return volumeRef, nil
}

// Load a volume passing its Id
func Load(ec2Ref *ec2.EC2, volume *Volume) (ec2.Volume, error) {
	if volume.ID == "" {
		return ec2.Volume{}, errors.New("To load a volume you need to pass its Id")
	}

	resp, err := ec2Ref.Volumes([]string{volume.ID}, nil)
	if err != nil {
		return ec2.Volume{}, err
	} else if len(resp.Volumes) == 0 {
		return ec2.Volume{}, fmt.Errorf("Any volume was found with volume Id <%s>", volume.ID)
	}

	volumeRef := resp.Volumes[0]
	mergeVolumes(volume, &volumeRef)

	return volumeRef, nil
}

// Create new volume
func Create(ec2Ref *ec2.EC2, volume *Volume) (ec2.Volume, error) {
	options := ec2.CreateVolume{
		VolumeType: volume.Type,
		AvailZone:  volume.AvailableZone,
	}

	if volume.Size > 0 {
		options.VolumeSize = volume.Size
	}

	if volume.SnapshotID != "" {
		options.SnapshotId = volume.SnapshotID
	}

	if volume.Type == "io1" {
		options.IOPS = volume.IOPS
	}

	resp, err := ec2Ref.CreateVolume(options)
	if err != nil {
		return ec2.Volume{}, err
	}

	volumeRef := resp.Volume
	tags := append(volume.Tags, ec2.Tag{"Name", volume.Name})
	_, err = ec2Ref.CreateTags([]string{volumeRef.Id}, tags)
	if err != nil {
		return ec2.Volume{}, err
	}

	mergeVolumes(volume, &volumeRef)

	err = WaitUntilState(ec2Ref, volume, "available")
	if err != nil {
		return ec2.Volume{}, err
	}

	return volumeRef, nil
}
