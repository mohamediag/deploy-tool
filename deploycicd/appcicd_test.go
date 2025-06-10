package deployappcicd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	
	"gopkg.in/yaml.v3"
)

// Test helper function to create temporary config files
func createTempConfigFile(t *testing.T, content string) string {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test-config.yaml")
	err := os.WriteFile(configFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp config file: %v", err)
	}
	return configFile
}

// Test data
var testDeployments = []Deployment{
	{
		ValueFile:     "value-dev.yaml",
		TargetCluster: "dev-01",
		InstanceName:  "my-app",
		Env:           "dev",
		PathToProd:    true,
	},
	{
		ValueFile:     "value-preprod.yaml",
		TargetCluster: "prod-01",
		InstanceName:  "my-app-preprod",
		Env:           "preprod",
		PathToProd:    true,
	},
	{
		ValueFile:     "value-prod.yaml",
		TargetCluster: "prod-01",
		InstanceName:  "my-app-prod",
		Env:           "prod",
		PathToProd:    false,
	},
}

func TestGetConfigConfigImport(t *testing.T) {
	result := getConfigConfigImport()
	
	expectedContent := []string{
		"include:",
		"- project: 'digital-factory/devops/continuous-integration-delivery'",
		"ref: 'master'",
		"file:",
		"- 'gitlab-ci/templates/cno-apps-multistage-pipeline.gitlab-ci.yaml'",
	}
	
	for _, expected := range expectedContent {
		if !strings.Contains(result, expected) {
			t.Errorf("Expected config import to contain '%s', but it was missing", expected)
		}
	}
}

func TestBuildJobNameByDeployment(t *testing.T) {
	t.Run("ValidDeployments", func(t *testing.T) {
		result := buildJobNameByDeployment(testDeployments)
		
		expectedJobs := []string{
			"push-my-app-to-dev-01",
			"push-my-app-preprod-to-prod-01",
			"push-my-app-prod-to-prod-01",
		}
		
		if len(result) != len(expectedJobs) {
			t.Errorf("Expected %d jobs, got %d", len(expectedJobs), len(result))
		}
		
		for _, expectedJob := range expectedJobs {
			if _, exists := result[expectedJob]; !exists {
				t.Errorf("Expected job '%s' not found in result", expectedJob)
			}
		}
		
		// Verify the deployment data is correctly mapped
		job := result["push-my-app-to-dev-01"]
		if job.InstanceName != "my-app" || job.TargetCluster != "dev-01" {
			t.Errorf("Job data not correctly mapped: %+v", job)
		}
	})
}

func TestGetDeployJob(t *testing.T) {
	t.Run("JobWithoutNeeds", func(t *testing.T) {
		result := getDeployJob("test-job", "test-value.yaml", "test-cluster", "test-instance", "dev", "")
		
		expectedContent := []string{
			"test-job:",
			"VALUE_FILE: test-value.yaml",
			"TARGET_CLUSTER: test-cluster",
			"INSTANCENAME: test-instance",
			"ENV: dev",
			"stage: Push Manifests Dev",
			"extends: .push-to-target-cluster-repo",
			"when: manual",
		}
		
		for _, expected := range expectedContent {
			if !strings.Contains(result, expected) {
				t.Errorf("Expected job to contain '%s', but it was missing", expected)
			}
		}
		
		// Should not contain needs section
		if strings.Contains(result, "needs:") {
			t.Error("Job should not contain needs section when needs is empty")
		}
	})
	
	t.Run("JobWithNeeds", func(t *testing.T) {
		result := getDeployJob("test-job", "test-value.yaml", "test-cluster", "test-instance", "preprod", "dependency-job")
		
		expectedContent := []string{
			"stage: Push Manifests Preprod",
			"needs:",
			"- dependency-job",
		}
		
		for _, expected := range expectedContent {
			if !strings.Contains(result, expected) {
				t.Errorf("Expected job to contain '%s', but it was missing", expected)
			}
		}
	})
	
	t.Run("DifferentEnvironments", func(t *testing.T) {
		testCases := []struct {
			env      string
			expected string
		}{
			{"dev", "Push Manifests Dev"},
			{"preprod", "Push Manifests Preprod"},
			{"prod", "Push Manifests Prod"},
		}
		
		for _, tc := range testCases {
			result := getDeployJob("test-job", "test-value.yaml", "test-cluster", "test-instance", tc.env, "")
			if !strings.Contains(result, tc.expected) {
				t.Errorf("For env '%s', expected stage '%s' not found", tc.env, tc.expected)
			}
		}
	})
}

func TestExtractNeededPreviousJobNameForEnv(t *testing.T) {
	jobNameByDeployment := map[string]Deployment{
		"push-my-app-to-dev-01": {
			Env:        "dev",
			PathToProd: true,
		},
		"push-my-app-feature-to-dev-01": {
			Env:        "dev",
			PathToProd: false,
		},
		"push-my-app-preprod-to-prod-01": {
			Env:        "preprod",
			PathToProd: true,
		},
		"push-my-app-demo-to-prod-01": {
			Env:        "preprod",
			PathToProd: false,
		},
	}
	
	t.Run("PreprodNeedsDev", func(t *testing.T) {
		result := extractNeededPreviousJobNameForEnv(jobNameByDeployment, "preprod")
		expected := "push-my-app-to-dev-01"
		if result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})
	
	t.Run("ProdNeedsPreprod", func(t *testing.T) {
		result := extractNeededPreviousJobNameForEnv(jobNameByDeployment, "prod")
		expected := "push-my-app-preprod-to-prod-01"
		if result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})
	
	t.Run("DevNeedsNothing", func(t *testing.T) {
		result := extractNeededPreviousJobNameForEnv(jobNameByDeployment, "dev")
		if result != "" {
			t.Errorf("Expected empty string for dev environment, got '%s'", result)
		}
	})
	
	t.Run("NoPathToProdDependencies", func(t *testing.T) {
		jobMap := map[string]Deployment{
			"push-my-app-feature-to-dev-01": {
				Env:        "dev",
				PathToProd: false,
			},
		}
		
		result := extractNeededPreviousJobNameForEnv(jobMap, "preprod")
		if result != "" {
			t.Errorf("Expected empty string when no pathToProd dev job exists, got '%s'", result)
		}
	})
}

func TestGeneratePipeline(t *testing.T) {
	t.Run("SimpleDeployments", func(t *testing.T) {
		deployments := []Deployment{
			{
				ValueFile:     "value-dev.yaml",
				TargetCluster: "dev-01",
				InstanceName:  "test-app",
				Env:           "dev",
				PathToProd:    false,
			},
			{
				ValueFile:     "value-prod.yaml",
				TargetCluster: "prod-01",
				InstanceName:  "test-app",
				Env:           "prod",
				PathToProd:    false,
			},
		}
		
		result := generatePipeline(deployments)
		
		// Should contain the config import
		if !strings.Contains(result, "include:") {
			t.Error("Pipeline should contain include section")
		}
		
		// Should contain both jobs
		if !strings.Contains(result, "push-test-app-to-dev-01:") {
			t.Error("Pipeline should contain dev job")
		}
		if !strings.Contains(result, "push-test-app-to-prod-01:") {
			t.Error("Pipeline should contain prod job")
		}
		
		// Dev job should not have needs (no PathToProd)
		devJobStart := strings.Index(result, "push-test-app-to-dev-01:")
		nextJobStart := strings.Index(result[devJobStart+1:], "push-")
		if nextJobStart == -1 {
			nextJobStart = len(result)
		} else {
			nextJobStart += devJobStart + 1
		}
		devJobSection := result[devJobStart:nextJobStart]
		if strings.Contains(devJobSection, "needs:") {
			t.Error("Dev job should not have needs section when PathToProd is false")
		}
	})
	
	t.Run("DeploymentsWithDependencies", func(t *testing.T) {
		deployments := []Deployment{
			{
				ValueFile:     "value-dev.yaml",
				TargetCluster: "dev-01",
				InstanceName:  "test-app",
				Env:           "dev",
				PathToProd:    true,
			},
			{
				ValueFile:     "value-preprod.yaml",
				TargetCluster: "preprod-01",
				InstanceName:  "test-app",
				Env:           "preprod",
				PathToProd:    true,
			},
			{
				ValueFile:     "value-prod.yaml",
				TargetCluster: "prod-01",
				InstanceName:  "test-app",
				Env:           "prod",
				PathToProd:    false,
			},
		}
		
		result := generatePipeline(deployments)
		
		// Preprod job should depend on dev job
		if !strings.Contains(result, "needs:\n    - push-test-app-to-dev-01") {
			t.Error("Preprod job should depend on dev job")
		}
		
		// Prod job should depend on preprod job
		if !strings.Contains(result, "needs:\n    - push-test-app-to-preprod-01") {
			t.Error("Prod job should depend on preprod job")
		}
	})
}

func TestGenerateDeployPipelineWithExistingConfig(t *testing.T) {
	// Test with the actual existing config file
	configPath := "app-config-file.yaml"
	
	// Check if the file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Skip("Existing config file not found, skipping integration test")
	}
	
	// This should not panic or error
	GenerateDeployPipeline(configPath)
}

func TestGenerateDeployPipelineWithTempConfig(t *testing.T) {
	t.Run("ValidConfig", func(t *testing.T) {
		configContent := `
deployments:
- instanceName: "test-app"
  valueFile: "value-dev.yaml"
  targetCluster: "dev-01"
  env: "dev"
  pathToProd: true
- instanceName: "test-app-prod"
  valueFile: "value-prod.yaml"
  targetCluster: "prod-01"
  env: "prod"
  pathToProd: false
`
		configFile := createTempConfigFile(t, configContent)
		
		// This should not panic
		GenerateDeployPipeline(configFile)
	})
	
	t.Run("EmptyConfig", func(t *testing.T) {
		configContent := `deployments: []`
		configFile := createTempConfigFile(t, configContent)
		
		// This should not panic even with empty deployments
		GenerateDeployPipeline(configFile)
	})
}

func TestBuildJobNameByDeploymentEdgeCases(t *testing.T) {
	t.Run("EmptyDeployments", func(t *testing.T) {
		result := buildJobNameByDeployment([]Deployment{})
		if len(result) != 0 {
			t.Errorf("Expected empty result for empty deployments, got %d items", len(result))
		}
	})
	
	t.Run("SpecialCharactersInNames", func(t *testing.T) {
		deployments := []Deployment{
			{
				InstanceName:  "my-app-v2.0",
				TargetCluster: "dev-cluster-01",
				Env:           "dev",
			},
		}
		
		result := buildJobNameByDeployment(deployments)
		expectedJobName := "push-my-app-v2.0-to-dev-cluster-01"
		if _, exists := result[expectedJobName]; !exists {
			t.Errorf("Expected job name '%s' not found", expectedJobName)
		}
	})
}

func TestEnvByStageMapping(t *testing.T) {
	expectedMappings := map[string]string{
		"dev":     "Push Manifests Dev",
		"preprod": "Push Manifests Preprod",
		"prod":    "Push Manifests Prod",
	}
	
	for env, expectedStage := range expectedMappings {
		if actualStage, exists := envByStage[env]; !exists {
			t.Errorf("Environment '%s' not found in envByStage", env)
		} else if actualStage != expectedStage {
			t.Errorf("For env '%s', expected stage '%s', got '%s'", env, expectedStage, actualStage)
		}
	}
}

func TestCompleteWorkflow(t *testing.T) {
	// Test the complete workflow with a comprehensive config
	configContent := `
deployments:
- instanceName: "api-service"
  valueFile: "values/api-dev.yaml"
  targetCluster: "dev-cluster"
  env: "dev"
  pathToProd: true
- instanceName: "api-service-feature"
  valueFile: "values/api-dev-feature.yaml"
  targetCluster: "dev-cluster"
  env: "dev"
  pathToProd: false
- instanceName: "api-service"
  valueFile: "values/api-preprod.yaml"
  targetCluster: "staging-cluster"
  env: "preprod"
  pathToProd: true
- instanceName: "api-service"
  valueFile: "values/api-prod.yaml"
  targetCluster: "prod-cluster"
  env: "prod"
  pathToProd: false
- instanceName: "worker-service"
  valueFile: "values/worker-prod.yaml"
  targetCluster: "prod-cluster"
  env: "prod"
  pathToProd: false
`
	
	configFile := createTempConfigFile(t, configContent)
	
	// Test that GenerateDeployPipeline completes without errors
	GenerateDeployPipeline(configFile)
	
	// Test the individual pipeline generation
	var config Config
	data, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}
	
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}
	
	pipeline := generatePipeline(config.Deployments)
	
	// Verify key components are present
	expectedJobs := []string{
		"push-api-service-to-dev-cluster:",
		"push-api-service-feature-to-dev-cluster:",
		"push-api-service-to-staging-cluster:",
		"push-api-service-to-prod-cluster:",
		"push-worker-service-to-prod-cluster:",
	}
	
	for _, expectedJob := range expectedJobs {
		if !strings.Contains(pipeline, expectedJob) {
			t.Errorf("Expected job '%s' not found in pipeline", expectedJob)
		}
	}
	
	// Verify dependencies are correct
	if !strings.Contains(pipeline, "needs:\n    - push-api-service-to-dev-cluster") {
		t.Error("Preprod job should depend on dev job")
	}
	
	if !strings.Contains(pipeline, "needs:\n    - push-api-service-to-staging-cluster") {
		t.Error("Prod job should depend on preprod job")
	}
}

func TestConfigStructUnmarshalling(t *testing.T) {
	t.Run("ValidYAML", func(t *testing.T) {
		yamlContent := `
deployments:
- instanceName: "test-app"
  valueFile: "values.yaml"
  targetCluster: "test-cluster"
  env: "dev"
  pathToProd: true
`
		var config Config
		err := yaml.Unmarshal([]byte(yamlContent), &config)
		if err != nil {
			t.Fatalf("Expected no error unmarshalling valid YAML, got: %v", err)
		}
		
		if len(config.Deployments) != 1 {
			t.Errorf("Expected 1 deployment, got %d", len(config.Deployments))
		}
		
		deployment := config.Deployments[0]
		if deployment.InstanceName != "test-app" {
			t.Errorf("Expected InstanceName 'test-app', got '%s'", deployment.InstanceName)
		}
		if deployment.PathToProd != true {
			t.Errorf("Expected PathToProd true, got %v", deployment.PathToProd)
		}
	})
	
	t.Run("InvalidYAML", func(t *testing.T) {
		invalidYAML := `
deployments:
- instanceName: "test-app"
  valueFile: "values.yaml"
  targetCluster: "test-cluster"
  env: "dev"
  pathToProd: invalid_boolean
`
		var config Config
		err := yaml.Unmarshal([]byte(invalidYAML), &config)
		if err == nil {
			t.Error("Expected error unmarshalling invalid YAML, got nil")
		}
	})
}

func TestJobNameUniqueness(t *testing.T) {
	t.Run("UniqueJobNames", func(t *testing.T) {
		// Test that unique job names work correctly
		deployments := []Deployment{
			{
				InstanceName:  "app1",
				TargetCluster: "cluster-01",
				Env:           "dev",
			},
			{
				InstanceName:  "app2",
				TargetCluster: "cluster-01",
				Env:           "prod",
			},
			{
				InstanceName:  "app1",
				TargetCluster: "cluster-02",
				Env:           "prod",
			},
		}
		
		result := buildJobNameByDeployment(deployments)
		
		expectedJobs := []string{
			"push-app1-to-cluster-01",
			"push-app2-to-cluster-01", 
			"push-app1-to-cluster-02",
		}
		
		if len(result) != len(expectedJobs) {
			t.Errorf("Expected %d unique jobs, got %d", len(expectedJobs), len(result))
		}
		
		for _, expectedJob := range expectedJobs {
			if _, exists := result[expectedJob]; !exists {
				t.Errorf("Expected job '%s' not found", expectedJob)
			}
		}
	})
}

func TestEdgeCasesInDependencyResolution(t *testing.T) {
	t.Run("MultipleProdJobsOnePreprod", func(t *testing.T) {
		// Test scenario where multiple prod jobs should depend on the same preprod job
		jobNameByDeployment := map[string]Deployment{
			"push-app-to-preprod": {
				Env:        "preprod",
				PathToProd: true,
			},
			"push-app-v1-to-prod": {
				Env:        "prod",
				PathToProd: false,
			},
			"push-app-v2-to-prod": {
				Env:        "prod",
				PathToProd: false,
			},
		}
		
		result1 := extractNeededPreviousJobNameForEnv(jobNameByDeployment, "prod")
		result2 := extractNeededPreviousJobNameForEnv(jobNameByDeployment, "prod")
		
		// Both should return the same preprod job
		if result1 != result2 {
			t.Errorf("Expected consistent results, got '%s' and '%s'", result1, result2)
		}
		
		if result1 != "push-app-to-preprod" {
			t.Errorf("Expected 'push-app-to-preprod', got '%s'", result1)
		}
	})
	
	t.Run("NoValidDependencies", func(t *testing.T) {
		// Test scenario where no valid dependencies exist
		jobNameByDeployment := map[string]Deployment{
			"push-app-to-prod": {
				Env:        "prod",
				PathToProd: false,
			},
		}
		
		result := extractNeededPreviousJobNameForEnv(jobNameByDeployment, "prod")
		if result != "" {
			t.Errorf("Expected empty string when no preprod dependencies exist, got '%s'", result)
		}
	})
}

func TestCompleteDeploymentScenarios(t *testing.T) {
	t.Run("MultipleAppsWithCrossEnvironmentDependencies", func(t *testing.T) {
		deployments := []Deployment{
			// App 1 - full dev->preprod->prod path
			{InstanceName: "app1", TargetCluster: "dev", Env: "dev", PathToProd: true, ValueFile: "app1-dev.yaml"},
			{InstanceName: "app1", TargetCluster: "preprod", Env: "preprod", PathToProd: true, ValueFile: "app1-preprod.yaml"},
			{InstanceName: "app1", TargetCluster: "prod", Env: "prod", PathToProd: false, ValueFile: "app1-prod.yaml"},
			// App 2 - dev only
			{InstanceName: "app2", TargetCluster: "dev", Env: "dev", PathToProd: false, ValueFile: "app2-dev.yaml"},
			// App 3 - prod only (no dependencies)
			{InstanceName: "app3", TargetCluster: "prod", Env: "prod", PathToProd: false, ValueFile: "app3-prod.yaml"},
		}
		
		pipeline := generatePipeline(deployments)
		
		// Verify all expected jobs are present
		expectedJobs := []string{
			"push-app1-to-dev:",
			"push-app1-to-preprod:",
			"push-app1-to-prod:",
			"push-app2-to-dev:",
			"push-app3-to-prod:",
		}
		
		for _, job := range expectedJobs {
			if !strings.Contains(pipeline, job) {
				t.Errorf("Expected job '%s' not found in pipeline", job)
			}
		}
		
		// Verify dependencies
		if !strings.Contains(pipeline, "needs:\n    - push-app1-to-dev") {
			t.Error("app1 preprod should depend on app1 dev")
		}
		if !strings.Contains(pipeline, "needs:\n    - push-app1-to-preprod") {
			t.Error("app1 prod should depend on app1 preprod")
		}
		
		// Verify that app2 and app3 have no dependencies
		app2JobStart := strings.Index(pipeline, "push-app2-to-dev:")
		if app2JobStart == -1 {
			t.Fatal("app2 job not found")
		}
		app2JobEnd := strings.Index(pipeline[app2JobStart:], "push-")
		if app2JobEnd == -1 {
			app2JobEnd = len(pipeline)
		} else {
			app2JobEnd += app2JobStart
		}
		app2Section := pipeline[app2JobStart:app2JobEnd]
		if strings.Contains(app2Section, "needs:") {
			t.Error("app2 should not have dependencies")
		}
	})
}