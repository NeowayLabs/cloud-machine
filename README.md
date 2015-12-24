[![Build Status](https://travis-ci.org/NeowayLabs/cloud-machine.svg?branch=master)](https://travis-ci.org/NeowayLabs/cloud-machine)

# Cloud Machine

This is a Go Project that should be used to create a cloud environment. The app
will create volumes and instance through AWS, although in the next future it'll
be possible use other backends like Microsoft Azure, Google Cloud Platform, etc.

## How compile?

The easier way to compile this project is getting
[Docker](http://docs.docker.com/linux/step_one/) in your **Linux distribuition**.

#### Docker at Linux

If you already have Docker installed in your Linux, you can simply type:
```make```. Two files will be created in root directory of project:
```machine-up``` and ```cluster-up```

#### Other way

If you have a Docker in other platforms (like MacOS X) you only can use it if
you execute all commands inside of docker container, because the executables
was compiled using Ubuntu. If execute all commands inside a Docker container is
not a problem to you go ahead, otherwise you should install Go to compile them.

To compile the project in your own platfomr, first you need
[Go](http://golang.org), to install [click here](https://golang.org/doc/install).
After that, you need to execute following commands, pay attention that we have
two entry points (```machine-up.go``` and ```cluster-up.go```), because that
we need to pass ```-d``` to ```go get```

```
export GOPATH=~/go # change to your go workspace
go get -d github.com/NeowayLabs/cloud-machine
cd $GOPATH/src/github.com/NeowayLabs/cloud-machine
go build machine-up.go auth.go
go build cluster-up.go auth.go
```

## How use?

We have two different executables:

* ```machine-up```: it's to create only one machine that have **ONLY one**
instance and how many volumes you need.

* ```cluster-up```: it's to create a cluster of machines, you need to tell it
which machine-config use and how many of this machine should run.

**IMPORTANT:** Each machine will verify if you are creating new volumes, if yes
a new provisory machine will be create only to format theses volumes, after
format the machine will be automatically destroyed. Cost will be applied.

#### Machine UP

This app will create a machine, it have a instance with one or more volumes, you
need pass to it a machine-config file. You can have two section, one to
describe information about your instance and another one with information about
your volumes.

To instance you can use follow properties:

* **id:** The id to load a already created instance. *[Optional]*. If you pass
this property all other properties will be ignored.
* **name:** It will be create a tag with Name key. *[Required]*
* **type:** The type of instance to create. *[Required]*
* **imageid:** The Image Id used to create the instance. *[Required]*
* **region:** Region used to create the instance. *[Required]*
* **defaultavailablezone:** The default available zone to create volumes. *[Optional]*
* **keyname:** Keyname used to permit access to instance. *[Required]*
* **securitygroups:** Array of security group Id. *[Required]*
* **subnetid:** Subnet Id of instance. *[Required]*
* **cloudconfig:** File that will be used to pass as userdata to instance. *[Optional]*
* **ebsoptimized:** If instance should be EBS Optimized, default is false. *[Optional]*
* **shutdownbehavior:** When you shutdown the machine will terminate or stop, default is stop. *[Optional]*
* **enableapitermination:** If you authorize terminate this instance by aws console, cli, etc, default is false. *[Optional]*
* **tags:** You can pass a list of key=values to add as tags to your instance. *[Optional]*

Volume obligatory parameters:

* **name:** It will be create a tag with Name key.
* **type:** The type of volume to create, can be standard, io1 and gp2. 
* **size:** The size of volume to created. 
* **availablezone:** Where to create the volume
* **device:** The device used of this volume.
* **mount:** Where should mount the volume.
* **filesystem:** File system used to mount the device.

Volume optional parameters:

* **iops:** The IOPS used to create volume, *only to io1 type*
* **snapshotid:** When informed, the volume is created from an existing snapshot. In this case the volume is not formatted, obviously.
* **tags:** You can pass a list of key=values to add as tags to your volume.
* **id:** The id to load a already created instance. If you pass this property all other properties will be ignored.


**IMPORTANT:** If you have new volumes (without ID property or snapshotId) a new machine will
be created only to format this volume, after format the machine will be
automatically destroyed. Cost will be applied.

**IMPORTANT:** If you want to create a volume from a snapshot and increase the size of the new volume, you need to run a resize2fs as [Aws Increase Volumes](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ebs-expand-volume.html)

Here we have an example of machine-config

```
# cloud-machine/mongo-node.yml
instance:
  name: mongo-node
  type: r3.xlarge
  imageid: ami-5d4d486d
  region: us-west-2
  defaultavailablezone: us-west-2a
  keyname: aws-keyname
  securitygroups: [sg-00000000,sg-00000001]
  subnetid: subnet-abcd0000
  cloudconfig: cloud-config/mongo-node.yaml
  ebsoptimized: true
  shutdownbehavior: stop
  enableapitermination: false
  tags:
    - key1: value1
    - key2: value2
    - key3: value3

volumes:
  - name: mongo-data
    type: io1
    size: 200
    iops: 1000
    availablezone: us-west-2a
    device: /dev/xvdk
    mount: /data
    filesystem: ext4
    tags:
    - key1: value1
    - key2: value2
    - key3: value3

  - name: mongo-journal
    #id: vol-123456
    type: io1
    availablezone: us-west-2a
    size: 25
    iops: 250
    device: /dev/xvdl
    mount: /journal
    filesystem: ext4
```

To run you need export AWS_ACCESS_KEY and AWS_SECRET_KEY and pass machine-config
file

```
export AWS_ACCESS_KEY=<your access key>
export AWS_SECRET_KEY=<your secret key>
./machine-up ./cloud-machine/mongo-node.yml
```

#### Cluster UP

This app will create a cluster of machine, each machine is defined
[above](#machine-up), you need pass a cloud-config, that it's a file that you
need to describe which machine-config.yml use and how nodes of this machine
should run.

In cluster-config you can pass follow properties:

* **machine:** The file used to describe machine ([see above](#machine-up)). *[Required]*
* **nodes:** How many machines should be create. *[Required]*

Sometimes you need use some default value to all your instances, for that leave theses fields empty inside in your
*machine spec*, and fill inside of default in your *cloud spec*. This is very helpful when you need update you image id
or if you want create your infra in another region for example, in this case you only need update in one file.

Here we have an example of cluster-config

```
# cloud-machine/app-cluster.yml
default:
  imageid: ami-5d4d486d
  region: us-west-2
  keyname: aws-keyname
  securitygroups: [sg-00000000,sg-00000001]
  subnetid: subnet-abcd0000
  defaultavailablezone: us-west-2a
  tags:
    - key1: value1
    - key2: value2
    - key3: value3

clusters:
  - machine: cloud-machine/mongo-node.yml
    nodes: 3

  - machine: cloud-machine/elasticsearch-node.yml
    nodes: 2
```

To run you need export AWS_ACCESS_KEY and AWS_SECRET_KEY and pass cluster-config
file

```
export AWS_ACCESS_KEY=<your access key>
export AWS_SECRET_KEY=<your secret key>
./cluster-up ./cloud-machine/app-cluster.yml
```

## Publishing the image

If you have the permissions and are logged (using docker login) just run:

    make publish

And a latest image will be generated. If you want to generate a specific version just run:

    make publish version=<version>

Where **version** is the version that you are generating (should match with the tag, you created a tag... right ?)
