terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 3.28.0"
    }
  }
}

data "archive_file" "zip" {
  type        = "zip"
  source_file = "build/aws-lambda-deregister-target-go"
  output_path = "build/aws-lambda-deregister-target-go.zip"
}

resource "aws_iam_role" "iam_for_lambda" {
  name = "tf-role-deregister-target-fargate-spot"

  assume_role_policy = jsonencode({
    Version: "2012-10-17",
    Statement: [
      {
        Effect: "Allow",
        Action: [ "sts:AssumeRole"],
        Principal: { "Service": "lambda.amazonaws.com"},
      }
    ]
  })
}

resource "aws_sqs_queue" "deadletter_queue_for_deregister_lambda" {
  name                      = "tf-deadelteer-queue-failed-deregister"
  message_retention_seconds = 1209600
  receive_wait_time_seconds = 10
}

resource "aws_iam_role_policy" "deregister_policy" {
  depends_on = [aws_iam_role.iam_for_lambda, aws_sqs_queue.deadletter_queue_for_deregister_lambda]

  name = "tf-policy-deregister-target-fargate-spot"
  role = aws_iam_role.iam_for_lambda.id

  policy = jsonencode({
    Version: "2012-10-17",
    Statement: [
      {
        Effect: "Allow",
        Action: [
          "ecs:DescribeServices",
          "elasticloadbalancing:DeregisterTargets",
          "ec2:DescribeSubnets"
        ],
        Resource: "*"
      },
      {
        Effect: "Allow",
        Action: [
          "ec2:DescribeNetworkInterfaces",
          "ec2:CreateNetworkInterface",
          "ec2:DeleteNetworkInterface",
          "ec2:DescribeInstances",
          "ec2:AttachNetworkInterface"
        ],
        Resource: "*"
      },
      {
        Effect: "Allow",
        Action: [ "sqs:SendMessage" ],
        Resource: aws_sqs_queue.deadletter_queue_for_deregister_lambda.arn
      }
    ]
  })
}

resource "aws_cloudwatch_event_rule" "fargate-spot-rule" {
  name        = "tf-deregister-targets-fargate-spot-rule"
  description = "Capture Fargate Spot tasks that are going to be shutdown."

  event_pattern = <<EOF
{
  "source": ["aws.ecs"],
  "detail-type": ["ECS Task State Change"],
  "detail": {
    "clusterArn": ["*"]
  }
}
EOF
}

resource "aws_lambda_function" "lambda_deregister_targets_fargate_spot" {
  depends_on = [data.archive_file.zip, aws_iam_role.iam_for_lambda, aws_iam_role_policy.deregister_policy, aws_cloudwatch_event_rule.fargate-spot-rule, aws_sqs_queue.deadletter_queue_for_deregister_lambda]

  function_name    = "tf_deregister_targets_fargate_spot"
  filename         = "build/aws-lambda-deregister-target-go.zip"
  handler          = "aws-lambda-deregister-target-go"
  source_code_hash = data.archive_file.zip.output_base64sha256
  role             = aws_iam_role.iam_for_lambda.arn
  runtime          = "go1.x"
  memory_size      = 128
  timeout          = 10

  dead_letter_config {
    target_arn = aws_sqs_queue.deadletter_queue_for_deregister_lambda.arn
  }
}

resource "aws_lambda_permission" "allow_cloudwatch_to_call_deregister_lambda" {
  depends_on = [aws_lambda_function.lambda_deregister_targets_fargate_spot]

  statement_id = "AllowExecutionFromCloudWatch"
  action = "lambda:InvokeFunction"
  function_name = aws_lambda_function.lambda_deregister_targets_fargate_spot.function_name
  principal = "events.amazonaws.com"
  source_arn = aws_cloudwatch_event_rule.fargate-spot-rule.arn
}

resource "aws_cloudwatch_event_target" "rule_target_lambda_deregister" {
  depends_on = [aws_lambda_function.lambda_deregister_targets_fargate_spot, aws_lambda_permission.allow_cloudwatch_to_call_deregister_lambda]

  rule = aws_cloudwatch_event_rule.fargate-spot-rule.name
  target_id = "aws-lambda-deregister-target-go"
  arn = aws_lambda_function.lambda_deregister_targets_fargate_spot.arn
}
