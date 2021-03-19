# ecs-fargate-drain-function

Terraform scripts for deploying lambda for de-registering load balancer targets on `FARGATE_SPOT` interruption.

- Lambda function code that is used here is written by @jicowan on issue: https://github.com/aws/containers-roadmap/issues/797
  Lambda for de-registering tasks: https://gist.github.com/jicowan/ad5e13d12577b41a22f83ed91a3e61bf
- EventBridge rule is created based on: https://docs.aws.amazon.com/AmazonECS/latest/developerguide/fargate-capacity-providers.html#fargate-capacity-providers-termination

A deadletter SQS queue is created for failed Lambda executions.

### Overview

All commands are defined in `MakeFile`.
You do not need to have installed `go` or `terraform` environments on your computer.
All commands are executed in inside docker image which needs to be build at the beginning.  

### Init - Build Docker image

At the very beginning we need to build image which will be used laetr for other commands.

Command: `make build` will build local `aws-samples/aws-lambda-deregister-targets-fargate-spot` image. In that image we initialize terraform with our terraform configuration. 
Check `DockerFile` for details.

### Source code compile

To compile your source code, run `make compile`. 
With this command executes `go build` command in docker Go env.
After this command we have new `build/` folder created containing compiled program.
All terrafrom tasks are depending on this one so we do not execute it when runnign `plan` or `apply`

### Terraform commands

`plan`, `apply` and `destroy` are supported terraform commands.
WIth `plan` we will just get output what resources will be created.

In order to run these commands we need to pass AWS acces,secret and region params to make. 

- `make awsDefaultRegion=eu-central-1  awsAccessKey=accessxxx   awsSecretKey=secretxxx plan`
- `make awsDefaultRegion=eu-central-1  awsAccessKey=accessxxx   awsSecretKey=secretxxx apply`
- `make awsDefaultRegion=eu-central-1  awsAccessKey=accessxxx   awsSecretKey=secretxxx destroy`

## Security

See [CONTRIBUTING](CONTRIBUTING.md#security-issue-notifications) for more information.

## License

This library is licensed under the MIT-0 License. See the LICENSE file.

