package main

import (
	"errors"
	"flag"
	"os"

	"github.com/vaughan0/go-ini"

	"gopkg.in/amz.v3/aws"
)

var (
	accessKey = flag.String("access-key", "", "AWS Access Key")
	secretKey = flag.String("secret-key", "", "AWS Secret Key")
)

// AwsAuth ...
func AwsAuth() (auth aws.Auth, err error) {
	auth.AccessKey = *accessKey
	auth.SecretKey = *secretKey

	if auth.AccessKey != "" && auth.SecretKey != "" {
		return
	} else if auth.AccessKey == "" && auth.SecretKey != "" {
		err = errors.New("-access-key not found in your command line")
		return
	} else if auth.AccessKey != "" && auth.SecretKey == "" {
		err = errors.New("-secret-key not found in your command line")
		return
	}

	auth, err = aws.EnvAuth()
	if err == nil {
		return
	} else if auth.AccessKey == "" && auth.SecretKey != "" {
		err = errors.New("AWS_ACCESS_KEY not found in environment")
		return
	} else if auth.AccessKey != "" && auth.SecretKey == "" {
		err = errors.New("AWS_SECRET_KEY not found in environment")
		return
	}

	file, err := ini.LoadFile("~/.aws/credentials")
	if err != nil {
		if err == os.ErrNotExist {
			err = errors.New("You need inform your AWS credentials using a) AWS_ACCESS_KEY and AWS_SECRET_KEY; b) -access-key and -secret-key; c) aws configure.")
		}
		return
	}

	auth.AccessKey, _ = file.Get("default", "aws_access_key_id")
	auth.SecretKey, _ = file.Get("default", "aws_secret_access_key")

	if auth.AccessKey == "" || auth.SecretKey == "" {
		err = errors.New("Your credentials ~/.aws/credentials is not valid")
		return
	}

	return
}
