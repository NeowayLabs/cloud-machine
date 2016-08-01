package auth

import (
	"errors"
	"fmt"
	"os"
	"os/user"

	"github.com/vaughan0/go-ini"

	"gopkg.in/amz.v3/aws"
)

// Aws get the AWS credentials from environment variables or
// from the file ~/.aws/credentials.
func Aws() (aws.Auth, error) {
	auth, err := aws.EnvAuth()

	if err == nil {
		return auth, nil
	}

	userInfo, err := user.Current()

	if err != nil || userInfo.HomeDir == "" {
		return aws.Auth{}, fmt.Errorf("Home directory not found: %s", err.Error())
	}

	file, err := ini.LoadFile(userInfo.HomeDir + "/.aws/credentials")

	if err != nil {
		if err == os.ErrNotExist {
			err = errors.New("You need inform your AWS credentials using a) AWS_ACCESS_KEY and AWS_SECRET_KEY; b) -access-key and -secret-key; c) aws configure.")
		}

		return aws.Auth{}, err
	}

	auth.AccessKey, _ = file.Get("default", "aws_access_key_id")
	auth.SecretKey, _ = file.Get("default", "aws_secret_access_key")

	if auth.AccessKey == "" || auth.SecretKey == "" {
		err = errors.New("Your credentials ~/.aws/credentials is not valid")
		return aws.Auth{}, err
	}

	return auth, nil
}
