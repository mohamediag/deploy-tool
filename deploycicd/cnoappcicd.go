package deployappcicd

import log "github.com/sirupsen/logrus"

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

}
func generatePipeline(Deployments []Deployment) {
	for _, deployment := range Deployments {
		log.Infof("Processing deployment %s", deployment.InstanceName)
	}
}
