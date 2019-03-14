package main

import (
	"fmt"
	"os"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sts"
)

func main() {

	if os.Getenv("AWS_TEST_ACCOUNT_PERMISSIONS_BOUNDARY_ARN") == "" {
		fmt.Printf("$AWS_TEST_ACCOUNT_PERMISSIONS_BOUNDARY_ARN not set, aborting")
		os.Exit(1)
	}
	if os.Getenv("AWS_TESTING_ACCOUNT_ID") == "" {
		fmt.Printf("$AWS_TESTING_ACCOUNT_ID not set, aborting")
		os.Exit(1)
	}

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	})

	// Check if this being run on the sandbox account
	stssvc := sts.New(sess)
	getCallerIdentityOutput, err := stssvc.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		panic(err)
	}
	if aws.StringValue(getCallerIdentityOutput.Account) != os.Getenv("AWS_TESTING_ACCOUNT_ID") {
		fmt.Printf("this cleanup script is only permitted to run on the testing account\n")
		os.Exit(1)
	}

	s3svc := s3.New(sess)
	s3BucketsToDeleteOutput, err := getS3BucketsToDelete(s3svc)
	if err != nil {
		panic(err)
	}

	// If there are a ton of results something bad happened
	if len(s3BucketsToDeleteOutput) > 10 {
		fmt.Printf("more than ten buckets to delete, aborting\n")
		os.Exit(1)
	}

	for _, s3BucketToDelete := range s3BucketsToDeleteOutput {
		err = deleteS3Bucket(s3svc, s3BucketToDelete)
		if err != nil {
			panic(err)
		}
	}

	iamsvc := iam.New(sess)
	iamUsersToDeleteOutput, err := getIAMUsersToDelete(iamsvc)
	if err != nil {
		panic(err)
	}

	// If there are a ton of results something bad happened
	if len(iamUsersToDeleteOutput) > 10 {
		fmt.Printf("more than ten users to delete, aborting\n")
		os.Exit(1)
	}

	for _, iamUserToDelete := range iamUsersToDeleteOutput {
		err = deleteIamUser(iamsvc, iamUserToDelete)
		if err != nil {
			panic(err)
		}
	}
}

func getS3BucketsToDelete(s3svc *s3.S3) ([]*string, error) {
	// List all the buckets
	listBucketsOutput, err := s3svc.ListBuckets(&s3.ListBucketsInput{})
	if err != nil {
		fmt.Printf("Error was here\n")
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

func deleteS3Bucket(s3svc *s3.S3, bucketName *string) error {
	//There should not be any items in the bucket from testing.
	// If there are items in the bucket this will fail.
	// Buckets cannot be deleted if there are objects in them.
	fmt.Printf("Starting bucket deletion for %s:\n", aws.StringValue(bucketName))
	fmt.Printf("- Deleting bucket\n")
	_, err := s3svc.DeleteBucket(&s3.DeleteBucketInput{Bucket: bucketName})
	return err
}

func getIAMUsersToDelete(iamsvc *iam.IAM) ([]*string, error) {
	// List all the users
	listUsersOutput, err := iamsvc.ListUsers(&iam.ListUsersInput{})
	if err != nil {
		return nil, err
	}

	var iamUsersToDelete []*string
	for _, user := range listUsersOutput.Users {
		match := regexp.MustCompile(`^.*-summon-platform$`).Match([]byte(aws.StringValue(user.UserName)))
		if match {
			getUserOutput, err := iamsvc.GetUser(&iam.GetUserInput{UserName: user.UserName})
			if err != nil {
				return nil, err
			}
			permissionsBoundaryArn := getUserOutput.User.PermissionsBoundary.PermissionsBoundaryArn
			if aws.StringValue(permissionsBoundaryArn) != os.Getenv("AWS_TEST_ACCOUNT_PERMISSIONS_BOUNDARY_ARN") {
				fmt.Printf("PermissionsBoundaryArn did not match for user %s, skipping\n", aws.StringValue(user.UserName))
			}
			for _, tagSet := range getUserOutput.User.Tags {
				if aws.StringValue(tagSet.Key) == "ridecell-operator" {
					iamUsersToDelete = append(iamUsersToDelete, user.UserName)
				}
			}
		}
	}
	return iamUsersToDelete, nil
}

func deleteIamUser(iamsvc *iam.IAM, username *string) error {
	fmt.Printf("Starting user deletion for %s:\n", aws.StringValue(username))

	// Users cannot be deleted if they have access keys, we have to clean them up first.
	listAccessKeysOutput, err := iamsvc.ListAccessKeys(&iam.ListAccessKeysInput{UserName: username})
	if err != nil {
		return err
	}
	for _, accessKeysMeta := range listAccessKeysOutput.AccessKeyMetadata {
		fmt.Printf("- Deleting User Access Key %s\n", aws.StringValue(accessKeysMeta.AccessKeyId))
		_, err := iamsvc.DeleteAccessKey(&iam.DeleteAccessKeyInput{
			UserName:    username,
			AccessKeyId: accessKeysMeta.AccessKeyId,
		})
		if err != nil {
			return err
		}
	}

	// We also have to delete all user policies cause AWS
	listUserPoliciesOutput, err := iamsvc.ListUserPolicies(&iam.ListUserPoliciesInput{UserName: username})
	if err != nil {
		return err
	}
	for _, policyName := range listUserPoliciesOutput.PolicyNames {
		fmt.Printf("- Deleting User Policy %s\n", aws.StringValue(policyName))
		_, err = iamsvc.DeleteUserPolicy(&iam.DeleteUserPolicyInput{
			UserName:   username,
			PolicyName: policyName,
		})
		if err != nil {
			return err
		}
	}
	fmt.Printf("- Deleting User\n")
	// Now that other resources tied to user are deleted we can delete the user itself
	_, err = iamsvc.DeleteUser(&iam.DeleteUserInput{UserName: username})
	if err != nil {
		return err
	}
	return nil
}
