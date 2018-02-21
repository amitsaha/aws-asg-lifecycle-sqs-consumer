# AWS ASG life cycle consumer

This is essentially a AWS SQS consumer, but focusing on processing 
[AWS ASG lifecycle hook](https://docs.aws.amazon.com/autoscaling/latest/userguide/lifecycle-hooks.html) messages.

To learn more see [this excellent blog post](circleci.com/blog/graceful-shutdown-using-aws/). You can consider this
as an implementation of the consumer.

Currently only `terminating` events are supported. You can specify a shutdown script to run when an instance is
set to be terminated as part of a scaling event via the `SHUTDOWN_SCRIPT` environment variable. Couple of other
configuration that must be specified are:

- `AWS_REGION`: This defaults to `us-east-1`
- `SQS_QUEUE_NAME`: Queue to consume from

## Design

The recommended approach to running this consumer is to run it along with all other services
that are running in each instance of your auto scaling group. This alongwith setting
the visibility timeout of the qeueu to 0 seconds means that a termination message
can be picked up any of the consumers. However, the consumer has inbuilt mechanism to 
only process events that are directed towards itself
(using `instance IDs`). If it is a message which it picks up for processing, it also deletes
it from the queue.

## IAM permissions

The consumer requires the following IAM permissions:

- sqs:GetQueueUrl
- sqs:ReceiveMessage
- sqs:DeleteMessage

## Building


## Example

```
$ AWS_REGION=ap-southeast-2 SQS_QUEUE_NAME=example_asg_lifecycle_queue SHUTDOWN_SCRIPT=/ tmp/shutdown.sh ./lifecycle_consumer
```
## Development

You will need Golang 1.8+ installed for local development:

```
$ make help
build:           Build the binary
vet:             Run go vet
lint:            Run go lint
help:            Show this help
```
### Vendoring

[dep](https://github.com/golang/dep) is used for vendoring and third
party package management. If you add any new dependency, run `dep ensure -add <package url>@master` and commit `Gopkg.lock` and `Gopkg.toml` files.

