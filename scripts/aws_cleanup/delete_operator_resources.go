package main

import (
	"fmt"
	"os"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/pkg/errors"
)

func main() {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	})

	// Check if this being run on the sandbox account
	stssvc := sts.New(sess)
	getCallerIdentityOutput, err := stssvc.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		exitWithError(err)
	}
	if aws.StringValue(getCallerIdentityOutput.Account) != os.Getenv("AWS_TESTING_ACCOUNT_ID") {
		exitWithError(errors.New("this cleanup script is only permitted to run on the testing account"))
	}

	s3svc := s3.New(sess)
	s3BucketsToDeleteOutput, err := getS3BucketsToDelete(s3svc)
	if err != nil {
		exitWithError(err)
	}

	// If there are a ton of results something bad happened
	if len(s3BucketsToDeleteOutput) > 10 {
		exitWithError(errors.New("more than ten buckets to delete, aborting"))
	}

	for _, s3BucketToDelete := range s3BucketsToDeleteOutput {
		err = deleteS3Bucket(s3svc, s3BucketToDelete)
		if err != nil {
			exitWithError(err)
		}
	}
}

func getS3BucketsToDelete(s3svc *s3.S3) ([]*string, error) {
	// List all the buckets
	listBucketsOutput, err := s3svc.ListBuckets(&s3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}

	// Iterate through buckets, find our targets via a combo of naming and tags
	var bucketsToDelete []*string
	for _, bucket := range listBucketsOutput.Buckets {
		match := regexp.MustCompile(`^ridecell-.*-static$`).Match([]byte(aws.StringValue(bucket.Name)))
		if match {
			getBucketTaggingOutput, err := s3svc.GetBucketTagging(&s3.GetBucketTaggingInput{
				Bucket: bucket.Name,
			})
			if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "NoSuchTagSet" {
				// There is no tag set associated with the bucket.
				getBucketTaggingOutput = &s3.GetBucketTaggingOutput{TagSet: []*s3.Tag{}}
			} else if err != nil {
				return nil, err
			}

			for _, tagSet := range getBucketTaggingOutput.TagSet {
				if aws.StringValue(tagSet.Key) == "ridecell-operator" {
					bucketsToDelete = append(bucketsToDelete, bucket.Name)
					break
				}
			}
		}
	}
	return bucketsToDelete, nil
}

func deleteS3Bucket(s3svc *s3.S3, bucketname *string) error {
	//There should not be any items in the bucket from testing.
	// If there are items in the bucket this will fail.
	// Buckets cannot be deleted if there are objects in them.
	_, err := s3svc.DeleteBucket(&s3.DeleteBucketInput{Bucket: bucketname})
	return err
}

func exitWithError(err error) {
	fmt.Printf("%s\n", err.Error())
	os.Exit(1)
}
