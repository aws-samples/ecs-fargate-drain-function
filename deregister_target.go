package main

// provided by @jicowan => https://gist.github.com/jicowan/ad5e13d12577b41a22f83ed91a3e61bf

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"log"
	"strings"
	"time"
)

type ECSEvent struct {
	Version    string    `json:"version,omitempty"`
	ID         string    `json:"id,omitempty"`
	DetailType string    `json:"detail-type,omitempty"`
	Source     string    `json:"source,omitempty"`
	Account    string    `json:"account,omitempty"`
	Time       time.Time `json:"time,omitempty"`
	Region     string    `json:"region,omitempty"`
	Resources  []string  `json:"resources,omitempty"`
	Detail     struct {
		ClusterArn string `json:"clusterArn,omitempty"`
		Containers []struct {
			ContainerArn      string `json:"containerArn,omitempty"`
			LastStatus        string `json:"lastStatus,omitempty"`
			Name              string `json:"name,omitempty"`
			TaskArn           string `json:"taskArn,omitempty"`
			NetworkInterfaces []struct {
				AttachmentID       string `json:"attachmentId,omitempty"`
				PrivateIpv4Address string `json:"privateIpv4Address,omitempty"`
			} `json:"networkInterfaces,omitempty"`
			CPU    string `json:"cpu,omitempty"`
			Memory string `json:"memory,omitempty"`
		} `json:"containers,omitempty"`
		CreatedAt     time.Time `json:"createdAt,omitempty"`
		LaunchType    string    `json:"launchType,omitempty"`
		CPU           string    `json:"cpu,omitempty"`
		Memory        string    `json:"memory,omitempty"`
		DesiredStatus string    `json:"desiredStatus,omitempty"`
		Group         string    `json:"group,omitempty"`
		LastStatus    string    `json:"lastStatus,omitempty"`
		Overrides     struct {
			ContainerOverrides []struct {
				Name string `json:"name,omitempty"`
			} `json:"containerOverrides,omitempty"`
		} `json:"overrides,omitempty"`
		Attachments []struct {
			ID      string `json:"id,omitempty"`
			Type    string `json:"type,omitempty"`
			Status  string `json:"status,omitempty"`
			Details []struct {
				Name  string `json:"name,omitempty"`
				Value string `json:"value,omitempty"`
			} `json:"details,omitempty"`
		} `json:"attachments,omitempty"`
		Connectivity      string    `json:"connectivity,omitempty"`
		ConnectivityAt    time.Time `json:"connectivityAt,omitempty"`
		PullStartedAt     time.Time `json:"pullStartedAt,omitempty"`
		StartedAt         time.Time `json:"startedAt,omitempty"`
		StartedBy         string    `json:"startedBy,omitempty"`
		StoppingAt        time.Time `json:"stoppingAt,omitempty"`
		PullStoppedAt     time.Time `json:"pullStoppedAt,omitempty"`
		StoppedReason     string    `json:"stoppedReason,omitempty"`
		StopCode          string    `json:"stopCode,omitempty"`
		UpdatedAt         time.Time `json:"updatedAt,omitempty"`
		TaskArn           string    `json:"taskArn,omitempty"`
		TaskDefinitionArn string    `json:"taskDefinitionArn,omitempty"`
		Version           int       `json:"version,omitempty"`
		PlatformVersion   string    `json:"platformVersion,omitempty"`
	} `json:"detail,omitempty"`
}

func main() {
	log.Println("Main started.")
	lambda.Start(HandleRequest)
	log.Println("Main Finished.")
}

func HandleRequest(e ECSEvent) error {
	log.Println("HandleRequest started. Parsing event.")
	eventAsJson, eventAsJsonError := json.Marshal(e)
	if eventAsJsonError != nil {
		log.Println("Parsing event JSON occurred")
		log.Println(eventAsJsonError)
	} else {
		log.Println("Event JSON parsed")
		log.Println(string(eventAsJson))
	}

	var privateIPv4Address string
	var subnetId []string
	var service []string
	for _, attachment := range e.Detail.Attachments {
		if attachment.Type != "eni" || attachment.Details == nil || len(attachment.Details) == 0  {
			break
		}

		if e.Detail.StopCode == "TerminationNotice" && strings.Contains(e.Detail.Group, "service:") {
			log.Println("TerminationNotice Event occurred.")

			for _, detail := range attachment.Details {
				if detail.Name == "privateIPv4Address" {
					privateIPv4Address = detail.Value
				}
				if detail.Name == "subnetId" {
					subnetId = append(subnetId, detail.Value)
				}
			}
			cluster := e.Detail.ClusterArn
			service = append(service, strings.Split(e.Detail.Group, ":")[1])
			targetGroups := getTargetGroups(service, cluster)
			az := getAvailabilityZone(subnetId)

			for i, tg := range targetGroups {
				log.Printf("_____ %d deregistering task from target group: %v\n", i, aws.ToString(tg))
				deregisterTask(&privateIPv4Address, az, tg, nil)
			}
		}
	}
	return nil

}
func getAvailabilityZone(subnetId []string) *string {
	ctx := context.Background()
	config, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}
	var az *string
	client := ec2.NewFromConfig(config)
	output, err := client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{SubnetIds: subnetId})
	if err != nil {
		log.Println(err)
	}
	for _, subnet := range output.Subnets {
		az = subnet.AvailabilityZone
	}
	return az
}

func deregisterTask(ip *string, az *string, tg *string, port *int32) {
	log.Println("###### Deregister task started.")
	log.Printf("###### ******* ip '%v'", aws.ToString(ip))
	log.Printf("###### ******* az '%v'", aws.ToString(az))
	log.Printf("###### ******* tg '%v'", aws.ToString(tg))

	ctx := context.Background()
	config, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}
	client := elasticloadbalancingv2.NewFromConfig(config)
	params := &elasticloadbalancingv2.DeregisterTargetsInput{
		TargetGroupArn: tg,
		Targets:        []types.TargetDescription{
			{
				Id:               ip,
				AvailabilityZone: az,
				Port:             port,
			},
		},
	}
	_, err = client.DeregisterTargets(ctx, params)
	if err != nil {
		log.Println(err)
	} else {
		log.Printf("The target %v was deregistered\n", aws.ToString(ip))
	}

	log.Println("###### Deregister task Finished.")
}

func getTargetGroups(svc []string, cluster string) []*string {
	log.Printf("The service name is: %v", svc[0])
	log.Printf("The clusterArn is: %v\n", cluster)
	ctx := context.Background()
	config, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	targetGroups := make(map[*string]bool)
	var targetGroupsResult []*string

	client := ecs.NewFromConfig(config)
	log.Printf("Finding target group\n")
	output, err := client.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Services: svc,
		Cluster:  aws.String(cluster),
	})
	if err != nil {
		log.Println(err)
	}

	for _, service := range output.Services {
		for _, lb := range service.LoadBalancers {
			targetGroups[lb.TargetGroupArn] = true
		}
	}

	for es := range targetGroups {
		targetGroupsResult = append(targetGroupsResult, es)
	}

	log.Printf("Extracted Target Groups Are: %v\n", aws.ToStringSlice(targetGroupsResult))
	return targetGroupsResult
}