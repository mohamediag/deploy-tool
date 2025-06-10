package deployappcicd

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"os"
)

type Config struct {
	Deployments []Deployment `json:"Deployments"`
}
type Deployment struct {
	ValueFile     string `yaml:"valueFile"`
	TargetCluster string `yaml:"targetCluster"`
	InstanceName  string `yaml:"instanceName"`
	Env           string `yaml:"env"`
	PathToProd    bool   `yaml:"pathToProd"`
}

func GenerateDeployPipeline(configFile string) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatalf("Error when reading config file %s: %v", configFile, err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("Error when unmarshalling config file %s: %v", configFile, err)
	}

	log.Infof("Loaded configuration with %d deployments", len(config.Deployments))

	deployPipeline := generatePipeline(config.Deployments)

	//Write the generated pipeline to a file named "deploy-pipeline.yaml"
	err = os.WriteFile("deploy-pipeline.yaml", []byte(deployPipeline), 0644)
	if err != nil {
		log.Fatalf("Error when writing deploy-pipeline.yaml: %v", err)
	}

}

var envByStage = map[string]string{
	"dev":     "Push Manifests Dev",
	"preprod": "Push Manifests Preprod",
	"prod":    "Push Manifests Prod",
}

func buildJobNameByDeployment(Deployments []Deployment) map[string]Deployment {
	jobNameByDeployment := make(map[string]Deployment)
	for _, deployment := range Deployments {
		jobName := fmt.Sprintf("push-%s-to-%s",
			deployment.InstanceName,
			deployment.TargetCluster)

		if _, exists := jobNameByDeployment[jobName]; exists {
			log.Fatalf("Job name %s already exists. "+
				"Please ensure unique InstanceName/TargetCluster (%s/%s) .",
				jobName, deployment.InstanceName, deployment.TargetCluster)
		}
		jobNameByDeployment[jobName] = deployment
	}
	log.Infof("Generated job names by deployment: %v", jobNameByDeployment)
	return jobNameByDeployment
}

func generatePipeline(Deployments []Deployment) string {
	deployPipeline := getConfigConfigImport()
	jobNameByDeployment := buildJobNameByDeployment(Deployments)

	for jobName, deployment := range jobNameByDeployment {
		needs := ""
		if deployment.Env == "preprod" || deployment.Env == "prod" {
			needs = extractNeededPreviousJobNameForEnv(jobNameByDeployment, deployment.Env)
		}
		deployPipeline += getDeployJob(
			jobName,
			deployment.ValueFile,
			deployment.TargetCluster,
			deployment.InstanceName,
			deployment.Env,
			needs)
	}
	log.Infof("Generated pipeline %s", deployPipeline)
	return deployPipeline
}

func getConfigConfigImport() string {
	conf := `
include:
  - project: 'digital-factory/devops/continuous-integration-delivery'
    ref: 'master'
    file:
      - 'gitlab-ci/templates/cno-apps-multistage-pipeline.gitlab-ci.yaml'

`
	return conf
}

func getDeployJob(jobName, valueFile, targetCluster, instanceName, env, needs string) string {
	baseTemplate := `
%s:
  variables:
    VALUE_FILE: %s
    TARGET_CLUSTER: %s
    INSTANCENAME: %s
    ENV: %s
  stage: %s
  extends: .push-to-target-cluster-repo
  when: manual
`
	needsSection := ""
	if needs != "" {
		needsSection = fmt.Sprintf("  needs:\n    - %s\n", needs)
	}
	jobTemplate := baseTemplate + needsSection

	return fmt.Sprintf(jobTemplate, jobName, valueFile, targetCluster, instanceName, env, envByStage[env])
}

func extractNeededPreviousJobNameForEnv(jobNameByDeployment map[string]Deployment, env string) string {
	if env == "preprod" {
		for jobName, deployment := range jobNameByDeployment {
			if deployment.Env == "dev" && deployment.PathToProd {
				return jobName
			}
		}
	}
	if env == "prod" {
		for jobName, deployment := range jobNameByDeployment {
			if deployment.Env == "preprod" && deployment.PathToProd {
				return jobName
			}
		}
	}
	return ""
}
