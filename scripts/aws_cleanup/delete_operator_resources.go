package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elasticsearchservice"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/rds"
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
	namePrefix := os.Getenv("RAND_OWNER_PREFIX")
	if namePrefix == "" {
		fmt.Printf("$RAND_OWNER_PREFIX not set, aborting")
		os.Exit(1)
	}

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-1"),
	})
	if err != nil {
		panic(err)
	}

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
	s3BucketsToDeleteOutput, err := getS3BucketsToDelete(s3svc, namePrefix)
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
	iamUsersToDeleteOutput, err := getIAMUsersToDelete(iamsvc, namePrefix)
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

	rdssvc := rds.New(sess)
	rdsInstancesToDeleteOutput, err := getRDSInstancesToDelete(rdssvc, namePrefix)
	if err != nil {
		panic(err)
	}

	if len(rdsInstancesToDeleteOutput) > 2 {
		fmt.Printf("more than one rds instance to delete, aborting\n")
		os.Exit(1)
	}

	for _, rdsInstanceToDelete := range rdsInstancesToDeleteOutput {
		err := deleteRDSInstance(rdssvc, rdsInstanceToDelete)
		if err != nil {
			panic(err)
		}
	}

	parameterGroupsToDeleteOutput, err := getParameterGroupsToDelete(rdssvc, namePrefix)
	if err != nil {
		panic(err)
	}

	// This may need to change later but for now we only make one test database
	if len(parameterGroupsToDeleteOutput) > 1 {
		fmt.Printf("more than one parameter group to delete, aborting\n")
		os.Exit(1)
	}

	for _, parameterGroupToDelete := range parameterGroupsToDeleteOutput {
		err := deleteParameterGroup(rdssvc, parameterGroupToDelete)
		if err != nil {
			panic(err)
		}
	}

	ec2svc := ec2.New(sess)

	securityGroupsToDeleteOutput, err := getSecurityGroupsToDelete(ec2svc, namePrefix)
	if err != nil {
		panic(err)
	}

	// This may need to change later but for now we only make one test database
	if len(securityGroupsToDeleteOutput) > 1 {
		fmt.Printf("more than one security group to delete, aborting\n")
		os.Exit(1)
	}

	for _, securityGroupToDelete := range securityGroupsToDeleteOutput {
		err := deleteSecurityGroup(ec2svc, securityGroupToDelete)
		if err != nil {
			panic(err)
		}
	}

	snapshotsToDeleteOutput, err := getSnapshotsToDelete(rdssvc, namePrefix)
	if err != nil {
		panic(err)
	}

	// This may need to change later but for now we only make one test database
	if len(snapshotsToDeleteOutput) > 3 {
		fmt.Printf("more than one db snapshot to delete, aborting\n")
		os.Exit(1)
	}

	for _, snapshotToDelete := range snapshotsToDeleteOutput {
		err := deleteSnapshot(rdssvc, snapshotToDelete)
		if err != nil {
			panic(err)
		}
	}

	essvc := elasticsearchservice.New(sess)
	err = deleteElasticsearchIfExists(essvc, namePrefix)
	if err != nil {
		panic(err)
	}
}

func getS3BucketsToDelete(s3svc *s3.S3, prefix string) ([]*string, error) {
	// List all the buckets
	listBucketsOutput, err := s3svc.ListBuckets(&s3.ListBucketsInput{})
	if err != nil {
		fmt.Printf("Error was here\n")
		return nil, err
	}

	// Iterate through buckets, find our targets via a combo of naming and tags
	var bucketsToDelete []*string
	for _, bucket := range listBucketsOutput.Buckets {
		regexString := fmt.Sprintf(`^ridecell-%s-.*-static$`, prefix)
		match := regexp.MustCompile(regexString).Match([]byte(aws.StringValue(bucket.Name)))
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

func getIAMUsersToDelete(iamsvc *iam.IAM, prefix string) ([]*string, error) {
	// List all the users
	listUsersOutput, err := iamsvc.ListUsers(&iam.ListUsersInput{})
	if err != nil {
		return nil, err
	}

	var iamUsersToDelete []*string
	for _, user := range listUsersOutput.Users {
		regexString := fmt.Sprintf(`^%s-.*-summon-platform$`, prefix)
		match := regexp.MustCompile(regexString).Match([]byte(aws.StringValue(user.UserName)))
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

func getIAMRolesToDelete(iamsvc *iam.IAM, prefix string) ([]*string, error) {
	// List all the users
	listRolesOutput, err := iamsvc.ListRoles(&iam.ListRolesInput{})
	if err != nil {
		return nil, err
	}

	var iamRolesToDelete []*string
	for _, role := range listRolesOutput.Roles {
		regexString := fmt.Sprintf(`^%s-.*-summon-platform$`, prefix)
		match := regexp.MustCompile(regexString).Match([]byte(aws.StringValue(role.RoleName)))
		if match {
			getRoleOutput, err := iamsvc.GetRole(&iam.GetRoleInput{RoleName: role.RoleName})
			if err != nil {
				return nil, err
			}
			permissionsBoundaryArn := getRoleOutput.Role.PermissionsBoundary.PermissionsBoundaryArn
			if aws.StringValue(permissionsBoundaryArn) != os.Getenv("AWS_TEST_ACCOUNT_PERMISSIONS_BOUNDARY_ARN") {
				fmt.Printf("PermissionsBoundaryArn did not match for role %s, skipping\n", aws.StringValue(role.RoleName))
			}
			for _, tagSet := range getRoleOutput.Role.Tags {
				if aws.StringValue(tagSet.Key) == "ridecell-operator" {
					iamRolesToDelete = append(iamRolesToDelete, role.RoleName)
				}
			}
		}
	}
	return iamRolesToDelete, nil
}

func deleteIamUser(iamsvc *iam.IAM, username *string) error {
	fmt.Printf("Starting user deletion for %s:\n", aws.StringValue(username))

	// Users cannot be deleted if they have access keys, we have to clean them up first.
	listAccessKeysOutput, err := iamsvc.ListAccessKeys(&iam.ListAccessKeysInput{UserName: username})
	if err != nil {
		return err
	}
	for _, accessKeysMeta := range listAccessKeysOutput.AccessKeyMetadata {
		fmt.Printf("- Deleting Access Key %s\n", aws.StringValue(accessKeysMeta.AccessKeyId))
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
		fmt.Printf("- Deleting Policy %s\n", aws.StringValue(policyName))
		_, err = iamsvc.DeleteUserPolicy(&iam.DeleteUserPolicyInput{
			UserName:   username,
			PolicyName: policyName,
		})
		if err != nil {
			return err
		}
	}
	fmt.Printf("- Deleting User\n")
	//Now that other resources tied to user are deleted we can delete the user itself
	_, err = iamsvc.DeleteUser(&iam.DeleteUserInput{UserName: username})
	if err != nil {
		return err
	}
	return nil
}

func deleteIamRole(iamsvc *iam.IAM, roleName *string) error {
	fmt.Printf("Starting role deletion for %s:\n", aws.StringValue(roleName))

	// We have to delete all role policies cause AWS
	listRolePoliciesOutput, err := iamsvc.ListRolePolicies(&iam.ListRolePoliciesInput{RoleName: roleName})
	if err != nil {
		return err
	}
	for _, policyName := range listRolePoliciesOutput.PolicyNames {
		fmt.Printf("- Deleting Policy %s\n", aws.StringValue(policyName))
		_, err = iamsvc.DeleteRolePolicy(&iam.DeleteRolePolicyInput{
			RoleName:   roleName,
			PolicyName: policyName,
		})
		if err != nil {
			return err
		}
	}
	fmt.Printf("- Deleting Role\n")
	//Now that other resources tied to user are deleted we can delete the role itself
	_, err = iamsvc.DeleteRole(&iam.DeleteRoleInput{RoleName: roleName})
	if err != nil {
		return err
	}
	return nil
}

func getRDSInstancesToDelete(rdssvc *rds.RDS, prefix string) ([]*string, error) {
	regexString := fmt.Sprintf(`^%s-test-rds|%s-snapshot-controller$`, prefix, prefix)
	var dbInstancesToDelete []*string
	describeDBInstancesOutput, err := rdssvc.DescribeDBInstances(&rds.DescribeDBInstancesInput{})
	if err != nil {
		return nil, err
	}
	for _, instance := range describeDBInstancesOutput.DBInstances {
		match := regexp.MustCompile(regexString).Match([]byte(aws.StringValue(instance.DBInstanceIdentifier)))
		if match {
			dbInstancesToDelete = append(dbInstancesToDelete, instance.DBInstanceIdentifier)
		}
	}
	return dbInstancesToDelete, nil
}

func deleteRDSInstance(rdssvc *rds.RDS, instanceIdentifier *string) error {
	fmt.Printf("Starting RDS Instance deletion for %s:\n", aws.StringValue(instanceIdentifier))
	//TODO:	ew
	for {
		describeDBInstancesOutput, err := rdssvc.DescribeDBInstances(&rds.DescribeDBInstancesInput{
			DBInstanceIdentifier: instanceIdentifier,
		})
		if err != nil {
			aerr, ok := err.(awserr.Error)
			if ok && aerr.Code() == rds.ErrCodeDBInstanceNotFoundFault {
				fmt.Printf("- RDS Instance deleted\n")
				return nil
			}
			return err
		}

		if aws.StringValue(describeDBInstancesOutput.DBInstances[0].DBInstanceStatus) == "deleting" {
			match := regexp.MustCompile(`^.*-snapshot-controller.*$`).MatchString(*instanceIdentifier)
			if match {
				fmt.Printf("- RDS Instance deleted\n")
				return nil
			}
			time.Sleep(time.Second * 30)
			continue
		}

		_, err = rdssvc.DeleteDBInstance(&rds.DeleteDBInstanceInput{
			DBInstanceIdentifier:   instanceIdentifier,
			DeleteAutomatedBackups: aws.Bool(true),
			SkipFinalSnapshot:      aws.Bool(true),
		})
		if err != nil {
			aerr, ok := err.(awserr.Error)
			if ok && aerr.Code() == rds.ErrCodeInvalidDBInstanceStateFault {
				time.Sleep(time.Second * 30)
				continue
			}
			return err
		}
	}
}

func getSecurityGroupsToDelete(ec2svc *ec2.EC2, prefix string) ([]*string, error) {
	regexString := fmt.Sprintf(`^%s-test-rds$`, prefix)
	var securityGroupsToDelete []*string
	describeSecurityGroupsOutput, err := ec2svc.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("vpc-id"),
				Values: []*string{aws.String(os.Getenv("VPC_ID"))},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	for _, securityGroup := range describeSecurityGroupsOutput.SecurityGroups {
		match := regexp.MustCompile(regexString).Match([]byte(aws.StringValue(securityGroup.GroupName)))
		if match {
			securityGroupsToDelete = append(securityGroupsToDelete, securityGroup.GroupId)
		}
	}
	return securityGroupsToDelete, nil
}

func deleteSecurityGroup(ec2svc *ec2.EC2, securityGroupID *string) error {
	fmt.Printf("- Deleting Security Group: %s\n", aws.StringValue(securityGroupID))
	_, err := ec2svc.DeleteSecurityGroup(&ec2.DeleteSecurityGroupInput{
		GroupId: securityGroupID,
	})
	if err != nil {
		return err
	}
	return nil
}

func getParameterGroupsToDelete(rdssvc *rds.RDS, prefix string) ([]*string, error) {
	regexString := fmt.Sprintf(`^%s-test-rds$`, prefix)
	var parameterGroupsToDelete []*string
	describeParameterGroupsOutput, err := rdssvc.DescribeDBParameterGroups(&rds.DescribeDBParameterGroupsInput{})
	if err != nil {
		return nil, err
	}

	for _, parameterGroup := range describeParameterGroupsOutput.DBParameterGroups {
		match := regexp.MustCompile(regexString).Match([]byte(aws.StringValue(parameterGroup.DBParameterGroupName)))
		if match {
			parameterGroupsToDelete = append(parameterGroupsToDelete, parameterGroup.DBParameterGroupName)
		}
	}
	return parameterGroupsToDelete, nil
}

func deleteParameterGroup(rdssvc *rds.RDS, parameterGroupName *string) error {
	fmt.Printf("- Deleting Parameter Group: %s\n", aws.StringValue(parameterGroupName))
	_, err := rdssvc.DeleteDBParameterGroup(&rds.DeleteDBParameterGroupInput{
		DBParameterGroupName: parameterGroupName,
	})
	if err != nil {
		return err
	}
	return nil
}

func getSnapshotsToDelete(rdssvc *rds.RDS, prefix string) ([]*string, error) {
	regexString := fmt.Sprintf(`^(%s|final-%s)-.*`, prefix, prefix)
	var snapshotsToDelete []*string
	err := rdssvc.DescribeDBSnapshotsPages(&rds.DescribeDBSnapshotsInput{}, func(page *rds.DescribeDBSnapshotsOutput, lastPage bool) bool {
		for _, snapshot := range page.DBSnapshots {
			match := regexp.MustCompile(regexString).Match([]byte(aws.StringValue(snapshot.DBSnapshotIdentifier)))
			if match {
				snapshotsToDelete = append(snapshotsToDelete, snapshot.DBSnapshotIdentifier)
			}
		}
		// if we get less than 100 results we hit the last page
		return !(len(page.DBSnapshots) < 100)
	})
	if err != nil {
		return nil, err
	}
	return snapshotsToDelete, nil
}

func deleteSnapshot(rdssvc *rds.RDS, snapshotIdentifier *string) error {
	fmt.Printf("- Deleting Snapshot: %s\n", aws.StringValue(snapshotIdentifier))
	_, err := rdssvc.DeleteDBSnapshot(&rds.DeleteDBSnapshotInput{
		DBSnapshotIdentifier: snapshotIdentifier,
	})
	if err != nil {
		return err
	}
	return nil
}

func deleteElasticsearchIfExists(essvc *elasticsearchservice.ElasticsearchService, prefix string) error {
	esDomainName := fmt.Sprintf("test-%s", prefix)
	_, err := essvc.DeleteElasticsearchDomain(&elasticsearchservice.DeleteElasticsearchDomainInput{
		DomainName: aws.String(strings.ToLower(esDomainName)),
	})
	if err != nil {
		aerr, ok := err.(awserr.Error)
		if ok {
			if aerr.Code() != elasticsearchservice.ErrCodeResourceNotFoundException {
				return err
			}
		} else {
			return err
		}
	}
	return nil
}
