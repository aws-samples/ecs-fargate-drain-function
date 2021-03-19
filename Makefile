build:
	docker build -t aws-samples/aws-lambda-deregister-targets-fargate-spot .

compile:
	docker run -e GOOS=linux -e GOARCH=amd64 -v $$(pwd):/app -w /app golang:1.15.8 go build  -ldflags="-s -w" -o build/aws-lambda-deregister-target-go

plan: compile
	docker run -v $$(pwd)/main.tf:/srv/main.tf -v $$(pwd)/terraform.tfstate:/srv/terraform.tfstate -v $$(pwd)/build:/srv/build -e AWS_ACCESS_KEY_ID=$(awsAccessKey) -e AWS_SECRET_ACCESS_KEY=$(awsSecretKey) -e AWS_DEFAULT_REGION=$(awsDefaultRegion) aws-samples/aws-lambda-deregister-targets-fargate-spot plan

apply: compile
	docker run -v $$(pwd)/main.tf:/srv/main.tf -v $$(pwd)/terraform.tfstate:/srv/terraform.tfstate -v $$(pwd)/build:/srv/build -e AWS_ACCESS_KEY_ID=$(awsAccessKey) -e AWS_SECRET_ACCESS_KEY=$(awsSecretKey) -e AWS_DEFAULT_REGION=$(awsDefaultRegion) aws-samples/aws-lambda-deregister-targets-fargate-spot apply -auto-approve

destroy:
	docker run -v $$(pwd)/main.tf:/srv/main.tf -v $$(pwd)/terraform.tfstate:/srv/terraform.tfstate -v $$(pwd)/build:/srv/build -e AWS_ACCESS_KEY_ID=$(awsAccessKey) -e AWS_SECRET_ACCESS_KEY=$(awsSecretKey) -e AWS_DEFAULT_REGION=$(awsDefaultRegion) aws-samples/aws-lambda-deregister-targets-fargate-spot destroy -auto-approve
