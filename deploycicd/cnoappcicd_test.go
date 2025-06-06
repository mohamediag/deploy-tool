package deployappcicd

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"testing"
)

func TestGenerateDeployPipeline_FromYaml(t *testing.T) {
	// Charger le fichier YAML
	data, err := ioutil.ReadFile("deployAppDeployconf.yaml")
	if err != nil {
		t.Fatalf("Erreur lors de la lecture du fichier YAML: %v", err)
	}

	// Désérialiser le YAML en structure Go
	var conf Config
	if err := yaml.Unmarshal(data, &conf); err != nil {
		t.Fatalf("Erreur lors du parsing YAML: %v", err)
	}

	// Appeler la fonction à tester
	GenerateDeployPipeline(conf.Deployments)

	// Ici, vous pouvez ajouter des assertions sur les effets attendus (logs, etc.)
}
