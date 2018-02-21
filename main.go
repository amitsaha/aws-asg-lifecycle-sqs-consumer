package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/sirupsen/logrus"
)

var (
	// Version populated during build
	Version string
	log     *logrus.Entry
)

type instanceTerminationMessageBody struct {
	LifecycleTransition string `json:"LifecycleTransition"`
	Ec2InstanceID       string `json:"EC2InstanceId"`
}

func main() {

	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)
	log = logrus.WithFields(logrus.Fields{"version": Version})

	log.Printf("Starting aws lifecycle consumer")

	// Look for the script to execute
	scriptPath := os.Getenv("SHUTDOWN_SCRIPT")
	if scriptPath == "" {
		log.Fatalf("SHUTDOWN_SCRIPT not specified")
	}

	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		log.Fatalf("%s does not exist\n", scriptPath)
	}

	awsRegion := os.Getenv("AWS_REGION")
	if awsRegion == "" {
		awsRegion = "us-east-1"
	}
	sess, err := session.NewSession(&aws.Config{Region: aws.String(awsRegion)})
	svc := sqs.New(sess)

	queueName := os.Getenv("SQS_QUEUE_NAME")
	if queueName == "" {
		log.Fatalf("SQS_QUEUE_NAME not specified")
	}
	resultURL, err := svc.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == sqs.ErrCodeQueueDoesNotExist {
			log.Fatalf("Unable to find queue %q.", queueName)
		}
		log.Fatalf("Unable to queue %q, %v.", queueName, err)
	}

	// Get our own instance ID
	// Create a EC2Metadata client from just a session.
	metadataSvc := ec2metadata.New(sess)
	if !metadataSvc.Available() {
		log.Fatal("Cannot access instance metadata or not running in AWS\n")
	}

	instanceID, err := metadataSvc.GetMetadata("instance-id/")
	if err != nil {
		log.Fatalf("Error retrieving the instance ID: %s", err)
	}

	for {
		result, err := svc.ReceiveMessage(&sqs.ReceiveMessageInput{
			AttributeNames: []*string{
				aws.String(sqs.MessageSystemAttributeNameSentTimestamp),
			},
			MessageAttributeNames: []*string{
				aws.String(sqs.QueueAttributeNameAll),
			},
			QueueUrl:            resultURL.QueueUrl,
			MaxNumberOfMessages: aws.Int64(1),
			VisibilityTimeout:   aws.Int64(0),
			WaitTimeSeconds:     aws.Int64(20),
		})

		if err != nil {
			log.Printf("Error receiving: %s\n", err)
		}

		if len(result.Messages) != 0 {
			messageBody := instanceTerminationMessageBody{}
			json.Unmarshal([]byte(*result.Messages[0].Body), &messageBody)
			sentTimestamp, err := strconv.ParseInt(*result.Messages[0].Attributes["SentTimestamp"], 10, 64)
			if err != nil {
				log.Printf("Error parsing sentTimestamp: %s\n", err)
			}

			now := time.Now()
			nowSecs := now.Unix()

			// We only process this message if it is meant for this instance ID
			// as we run miltiple consumers with a visibility timeout of 0 seconds

			if messageBody.LifecycleTransition == "autoscaling:EC2_INSTANCE_TERMINATING" && messageBody.Ec2InstanceID == instanceID {
				if (nowSecs - sentTimestamp) > 300 {
					log.Printf("Stale message recieved. But since it's for me - actioning")
				}
				cmd := exec.Command("/bin/bash", "-c", scriptPath)
				cmdOutput, err := cmd.Output()
				if err != nil {
					errorMsg := ""
					ee, ok := err.(*exec.ExitError)
					if ok {
						errorMsg = string(ee.Stderr)
					}
					log.Printf("Error executing shutdown script: %s %s", errorMsg, err.Error())
				} else {
					log.Printf("Shutdown script executed successfully: %s", string(cmdOutput))
				}

				// Delete the message
				_, err = svc.DeleteMessage(&sqs.DeleteMessageInput{
					QueueUrl:      resultURL.QueueUrl,
					ReceiptHandle: result.Messages[0].ReceiptHandle,
				})
				if err != nil {
					log.Printf("Error deleting: %s\n", err)
				}
			} else {
				log.Printf("Ignoring message:%v since it was not meant for me\n", result.Messages[0])
			}
		}
		// Sleep for 60 seconds before polling
		time.Sleep(60 * time.Second)
		log.Printf("Sleeping now\n")
	}
}
