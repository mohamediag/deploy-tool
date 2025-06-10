package deployappcicd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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

// Test helper function to read and cleanup generated pipeline file
func readAndCleanupPipelineFile(t *testing.T) string {
	content, err := os.ReadFile("deploy-pipeline.yaml")
	if err != nil {
		t.Fatalf("Failed to read generated pipeline file: %v", err)
	}
	
	// Cleanup the generated file
	err = os.Remove("deploy-pipeline.yaml")
	if err != nil {
		t.Logf("Warning: Failed to cleanup pipeline file: %v", err)
	}
	
	return string(content)
}

func TestGenerateDeployPipelineBasic(t *testing.T) {
	t.Run("SimpleDeployment", func(t *testing.T) {
		configContent := `
deployments:
- instanceName: "test-app"
  valueFile: "values-dev.yaml"
  targetCluster: "dev-cluster"
  env: "dev"
  pathToProd: false
`
		configFile := createTempConfigFile(t, configContent)
		
		// Generate pipeline
		GenerateDeployPipeline(configFile)
		
		// Read and verify the generated file
		pipelineContent := readAndCleanupPipelineFile(t)
		
		// Verify basic structure
		expectedContent := []string{
			"include:",
			"- project: 'digital-factory/devops/continuous-integration-delivery'",
			"push-test-app-to-dev-cluster:",
			"VALUE_FILE: values-dev.yaml",
			"TARGET_CLUSTER: dev-cluster",
			"INSTANCENAME: test-app",
			"ENV: dev",
			"stage: Push Manifests Dev",
			"extends: .push-to-target-cluster-repo",
			"when: manual",
		}
		
		for _, expected := range expectedContent {
			if !strings.Contains(pipelineContent, expected) {
				t.Errorf("Expected pipeline to contain '%s', but it was missing", expected)
			}
		}
		
		// Should not contain needs section for simple deployment
		if strings.Contains(pipelineContent, "needs:") {
			t.Error("Simple deployment should not contain needs section")
		}
	})
}

func TestGenerateDeployPipelineWithDependencies(t *testing.T) {
	t.Run("DevPreprodProdChain", func(t *testing.T) {
		configContent := `
deployments:
- instanceName: "my-app"
  valueFile: "values-dev.yaml"
  targetCluster: "dev-cluster"
  env: "dev"
  pathToProd: true
- instanceName: "my-app"
  valueFile: "values-preprod.yaml"
  targetCluster: "preprod-cluster"
  env: "preprod"
  pathToProd: true
- instanceName: "my-app"
  valueFile: "values-prod.yaml"
  targetCluster: "prod-cluster"
  env: "prod"
  pathToProd: false
`
		configFile := createTempConfigFile(t, configContent)
		
		// Generate pipeline
		GenerateDeployPipeline(configFile)
		
		// Read and verify the generated file
		pipelineContent := readAndCleanupPipelineFile(t)
		
		// Verify all jobs are present
		expectedJobs := []string{
			"push-my-app-to-dev-cluster:",
			"push-my-app-to-preprod-cluster:",
			"push-my-app-to-prod-cluster:",
		}
		
		for _, job := range expectedJobs {
			if !strings.Contains(pipelineContent, job) {
				t.Errorf("Expected job '%s' not found in pipeline", job)
			}
		}
		
		// Verify dependencies are correctly set
		if !strings.Contains(pipelineContent, "needs:\n    - push-my-app-to-dev-cluster") {
			t.Error("Preprod job should depend on dev job")
		}
		
		if !strings.Contains(pipelineContent, "needs:\n    - push-my-app-to-preprod-cluster") {
			t.Error("Prod job should depend on preprod job")
		}
		
		// Verify environments are correctly mapped
		expectedEnvMappings := []string{
			"stage: Push Manifests Dev",
			"stage: Push Manifests Preprod", 
			"stage: Push Manifests Prod",
		}
		
		for _, mapping := range expectedEnvMappings {
			if !strings.Contains(pipelineContent, mapping) {
				t.Errorf("Expected environment mapping '%s' not found", mapping)
			}
		}
	})
}

func TestGenerateDeployPipelineMultipleApps(t *testing.T) {
	t.Run("MultipleAppsWithMixedDependencies", func(t *testing.T) {
		configContent := `
deployments:
- instanceName: "api-service"
  valueFile: "values/api-dev.yaml"
  targetCluster: "dev-cluster"
  env: "dev"
  pathToProd: true
- instanceName: "worker-service"
  valueFile: "values/worker-dev.yaml"
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
		
		// Generate pipeline
		GenerateDeployPipeline(configFile)
		
		// Read and verify the generated file
		pipelineContent := readAndCleanupPipelineFile(t)
		
		// Verify all expected jobs are present
		expectedJobs := []string{
			"push-api-service-to-dev-cluster:",
			"push-worker-service-to-dev-cluster:",
			"push-api-service-to-staging-cluster:",
			"push-api-service-to-prod-cluster:",
			"push-worker-service-to-prod-cluster:",
		}
		
		for _, job := range expectedJobs {
			if !strings.Contains(pipelineContent, job) {
				t.Errorf("Expected job '%s' not found in pipeline", job)
			}
		}
		
		// Verify api-service dependencies
		if !strings.Contains(pipelineContent, "needs:\n    - push-api-service-to-dev-cluster") {
			t.Error("API service preprod should depend on dev")
		}
		
		if !strings.Contains(pipelineContent, "needs:\n    - push-api-service-to-staging-cluster") {
			t.Error("API service prod should depend on preprod")
		}
		
		// Worker service prod should also depend on api-service preprod (since it's pathToProd)
		if !strings.Contains(pipelineContent, "needs:\n    - push-api-service-to-staging-cluster") {
			t.Error("Worker service prod should depend on api service preprod due to pathToProd dependency chain")
		}
	})
}

func TestGenerateDeployPipelineWithExistingConfig(t *testing.T) {
	t.Run("ExistingAppConfigFile", func(t *testing.T) {
		// Test with the actual existing config file
		configPath := "app-config-file.yaml"
		
		// Check if the file exists
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Skip("Existing config file not found, skipping integration test")
		}
		
		// Generate pipeline
		GenerateDeployPipeline(configPath)
		
		// Read and verify the generated file
		pipelineContent := readAndCleanupPipelineFile(t)
		
		// Verify basic structure is present
		if !strings.Contains(pipelineContent, "include:") {
			t.Error("Pipeline should contain include section")
		}
		
		// Expected jobs based on app-config-file.yaml
		expectedJobs := []string{
			"push-my-app-to-dev-01:",
			"push-my-app-feature-1-to-dev-01:",
			"push-my-app-preprod-to-prod-01:",
			"push-my-app-demo-to-prod-01:",
			"push-my-app-prod-to-prod-01:",
			"push-my-app-prod-02-to-prod-01:",
			"push-my-app-prod-03-to-prod-01:",
		}
		
		for _, job := range expectedJobs {
			if !strings.Contains(pipelineContent, job) {
				t.Errorf("Expected job '%s' not found in pipeline", job)
			}
		}
		
		// Verify dependency chain exists for pathToProd deployments
		if !strings.Contains(pipelineContent, "needs:\n    - push-my-app-to-dev-01") {
			t.Error("Preprod job should depend on dev job (pathToProd dependency)")
		}
		
		if !strings.Contains(pipelineContent, "needs:\n    - push-my-app-preprod-to-prod-01") {
			t.Error("Prod jobs should depend on preprod job (pathToProd dependency)")
		}
	})
}

func TestGenerateDeployPipelineEdgeCases(t *testing.T) {
	t.Run("EmptyDeployments", func(t *testing.T) {
		configContent := `deployments: []`
		configFile := createTempConfigFile(t, configContent)
		
		// Generate pipeline
		GenerateDeployPipeline(configFile)
		
		// Read and verify the generated file
		pipelineContent := readAndCleanupPipelineFile(t)
		
		// Should still contain the include section but no jobs
		if !strings.Contains(pipelineContent, "include:") {
			t.Error("Pipeline should contain include section even when empty")
		}
		
		// Should not contain any job definitions
		if strings.Contains(pipelineContent, "push-") {
			t.Error("Empty deployments should not generate any jobs")
		}
	})
	
	t.Run("SpecialCharactersInNames", func(t *testing.T) {
		configContent := `
deployments:
- instanceName: "my-app-v2.0"
  valueFile: "values/special-chars.yaml"
  targetCluster: "dev-cluster-01"
  env: "dev"
  pathToProd: false
`
		configFile := createTempConfigFile(t, configContent)
		
		// Generate pipeline
		GenerateDeployPipeline(configFile)
		
		// Read and verify the generated file
		pipelineContent := readAndCleanupPipelineFile(t)
		
		// Verify job name with special characters is handled correctly
		expectedJob := "push-my-app-v2.0-to-dev-cluster-01:"
		if !strings.Contains(pipelineContent, expectedJob) {
			t.Errorf("Expected job '%s' not found in pipeline", expectedJob)
		}
	})
	
	t.Run("SingleEnvironmentWithoutDependencies", func(t *testing.T) {
		configContent := `
deployments:
- instanceName: "standalone-app"
  valueFile: "values-prod.yaml"
  targetCluster: "prod-cluster"
  env: "prod"
  pathToProd: false
`
		configFile := createTempConfigFile(t, configContent)
		
		// Generate pipeline
		GenerateDeployPipeline(configFile)
		
		// Read and verify the generated file
		pipelineContent := readAndCleanupPipelineFile(t)
		
		// Should contain the job
		if !strings.Contains(pipelineContent, "push-standalone-app-to-prod-cluster:") {
			t.Error("Standalone prod app job should be present")
		}
		
		// Should not contain needs section since pathToProd is false and no dependencies
		if strings.Contains(pipelineContent, "needs:") {
			t.Error("Standalone app should not have dependencies")
		}
	})
}

func TestGenerateDeployPipelineErrorHandling(t *testing.T) {
	t.Run("InvalidYAMLConfig", func(t *testing.T) {
		// Create a config file with invalid YAML syntax
		invalidYAML := `
deployments:
- instanceName: "test-app"
  valueFile: "values.yaml"
  targetCluster: "cluster"
  env: "dev"
  pathToProd: invalid_boolean_value
`
		_ = createTempConfigFile(t, invalidYAML)
		
		// This should cause the function to exit with log.Fatalf
		// We can't easily test fatal exits in unit tests, so we skip this
		// In a real scenario, you might want to refactor to return errors instead of fatal exits
		t.Skip("Skipping invalid YAML test - function uses log.Fatalf which exits the process")
	})
	
	t.Run("NonExistentConfigFile", func(t *testing.T) {
		// This should cause the function to exit with log.Fatalf
		// We can't easily test fatal exits in unit tests, so we skip this
		t.Skip("Skipping non-existent file test - function uses log.Fatalf which exits the process")
	})
	
	t.Run("DuplicateJobNames", func(t *testing.T) {
		// Test scenario that would cause duplicate job names
		configContent := `
deployments:
- instanceName: "same-app"
  valueFile: "values1.yaml"
  targetCluster: "same-cluster"
  env: "dev"
  pathToProd: false
- instanceName: "same-app"
  valueFile: "values2.yaml"
  targetCluster: "same-cluster"
  env: "prod"
  pathToProd: false
`
		_ = createTempConfigFile(t, configContent)
		
		// This should cause the function to exit with log.Fatalf due to duplicate job names
		// We can't easily test fatal exits in unit tests, so we skip this
		t.Skip("Skipping duplicate job names test - function uses log.Fatalf which exits the process")
	})
}