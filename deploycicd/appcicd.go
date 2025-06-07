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
	PathToProd    string `yaml:"pathToProd"`
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

	// Utilisez config ici
	log.Infof("Loaded configuration with %d deployments", len(config.Deployments))

	generatePipeline(config.Deployments)

}
func generatePipeline(Deployments []Deployment) {
	for _, deployment := range Deployments {
		log.Infof("Processing deployment %s", deployment.InstanceName)
	}
}

func getConfigConfigImport() string {
	conf := `
include:
  - project: 'digital-factory/devops/continuous-integration-delivery'
    ref: 'master'
    file:
      - 'gitlab-ci/templates/cno-apps-multistage-pipeline.gitlab-ci.yaml'
`
	return fmt.Sprintf(conf)
}

func getDeployBranch(valueFile, targetCluster, instanceName,
	env, stage string) string {
	conf := `
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
	return fmt.Sprintf(conf, environment, entity, environment, branchName, image)
}

/*
ValueFile     string `yaml:"valueFile"`
TargetCluster string `yaml:"targetCluster"`
InstanceName  string `yaml:"instanceName"`
Env           string `yaml:"env"`
PathToProd    string `yaml:"pathToProd"`

  - Push Manifests Dev
  - Push Manifests Preprod
  - Push Manifests Prod
*/
