resource "aws_iam_role" "publish_autoscaling_events_role" {
  name = "publish_autoscaling_events_role"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": "autoscaling.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "lifecycle_hook_autoscaling_policy" {
  name = "lifecycle_hook_autoscaling_policy"
  role = "${aws_iam_role.publish_autoscaling_events_role.id}"

  policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "",
            "Effect": "Allow",
            "Action": [
                "sqs:GetQueueUrl",
                "sqs:SendMessage"
            ],
            "Resource": [
                "*"
            ]
        }
    ]
}
EOF
}

resource "aws_iam_role_policy" "consume_lifecycle_events" {
  name = "consume_lifecycle_events"
  role = "${module.my_iam.role}"

  policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "",
            "Effect": "Allow",
            "Action": [
                "sqs:GetQueueUrl",
                "sqs:ReceiveMessage",
                "sqs:DeleteMessage"
            ],
            "Resource": [
                "${aws_sqs_queue.graceful_termination_queue.arn}"
            ]
        }
    ]
}
EOF
}

resource "aws_sqs_queue" "graceful_termination_queue" {
  name = "queue_asg_lifecycle"

  message_retention_seconds  = 900
  receive_wait_time_seconds  = 20
  visibility_timeout_seconds = 0
}

resource "aws_autoscaling_lifecycle_hook" "graceful_shutdown_asg_hook" {
  name                   = "graceful_shutdown_asg_hook"
  autoscaling_group_name = "${module.my_asg.autoscaling_group}"
  default_result         = "CONTINUE"

  # Give 15 minutes to the instance shutdown hooks
  heartbeat_timeout       = "900"
  lifecycle_transition    = "autoscaling:EC2_INSTANCE_TERMINATING"
  notification_target_arn = "${aws_sqs_queue.graceful_termination_queue.arn}"
  role_arn                = "${aws_iam_role.publish_autoscaling_events_role.arn}"
}
