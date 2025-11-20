package test

import (
	"strings"
	"testing"

	"github.com/teamcurri/puff/test/helpers"
)

// TestPrecedence_SixLevelHierarchy tests the complete 6-level precedence system
func TestPrecedence_SixLevelHierarchy(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Level 1: base/shared (lowest precedence)
	env.Set("VAR", "level1-base-shared", "-a", "shared", "-e", "base").AssertSuccess()

	// Verify level 1 is used
	result := env.Generate("myapp", "dev", "json")
	result.AssertSuccess()
	if !strings.Contains(result.GetStdout(), "level1-base-shared") {
		t.Error("Level 1 (base/shared) not applied")
	}

	// Level 2: base/app
	env.Set("VAR", "level2-base-app", "-a", "myapp", "-e", "base").AssertSuccess()

	// Verify level 2 overrides level 1
	result = env.Generate("myapp", "dev", "json")
	result.AssertSuccess()
	output := result.GetStdout()
	if !strings.Contains(output, "level2-base-app") {
		t.Error("Level 2 (base/app) not applied")
	}
	if strings.Contains(output, "level1-base-shared") {
		t.Error("Level 1 should be overridden by level 2")
	}

	// Level 3: env/shared
	env.Set("VAR", "level3-env-shared", "-a", "shared", "-e", "dev").AssertSuccess()

	// Verify level 3 overrides level 2
	result = env.Generate("myapp", "dev", "json")
	result.AssertSuccess()
	output = result.GetStdout()
	if !strings.Contains(output, "level3-env-shared") {
		t.Error("Level 3 (env/shared) not applied")
	}
	if strings.Contains(output, "level2-base-app") {
		t.Error("Level 2 should be overridden by level 3")
	}

	// Level 4: env/app
	env.Set("VAR", "level4-env-app", "-a", "myapp", "-e", "dev").AssertSuccess()

	// Verify level 4 overrides level 3
	result = env.Generate("myapp", "dev", "json")
	result.AssertSuccess()
	output = result.GetStdout()
	if !strings.Contains(output, "level4-env-app") {
		t.Error("Level 4 (env/app) not applied")
	}
	if strings.Contains(output, "level3-env-shared") {
		t.Error("Level 3 should be overridden by level 4")
	}

	// Level 5: target/shared
	env.Set("VAR", "level5-target-shared", "-a", "shared", "-e", "dev", "-t", "docker").AssertSuccess()

	// Verify level 5 overrides level 4 when target is specified
	result = env.Generate("myapp", "dev", "json", "-t", "docker")
	result.AssertSuccess()
	output = result.GetStdout()
	if !strings.Contains(output, "level5-target-shared") {
		t.Error("Level 5 (target/shared) not applied")
	}
	if strings.Contains(output, "level4-env-app") {
		t.Error("Level 4 should be overridden by level 5")
	}

	// Level 6: target/app (highest precedence)
	env.Set("VAR", "level6-target-app", "-a", "myapp", "-e", "dev", "-t", "docker").AssertSuccess()

	// Verify level 6 overrides all others
	result = env.Generate("myapp", "dev", "json", "-t", "docker")
	result.AssertSuccess()
	output = result.GetStdout()
	if !strings.Contains(output, "level6-target-app") {
		t.Error("Level 6 (target/app) not applied")
	}
	if strings.Contains(output, "level5-target-shared") {
		t.Error("Level 5 should be overridden by level 6")
	}

	// Verify without target, only levels 1-4 apply
	result = env.Generate("myapp", "dev", "json")
	result.AssertSuccess()
	output = result.GetStdout()
	if !strings.Contains(output, "level4-env-app") {
		t.Error("Without target, should use level 4")
	}
	if strings.Contains(output, "level5-target-shared") || strings.Contains(output, "level6-target-app") {
		t.Error("Target levels should not apply without target specified")
	}
}

// TestPrecedence_BaseSharedIsLowest tests that base/shared is overridden by everything
func TestPrecedence_BaseSharedIsLowest(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set base/shared values
	env.Set("LOG_LEVEL", "error", "-a", "shared", "-e", "base").AssertSuccess()
	env.Set("TIMEOUT", "30", "-a", "shared", "-e", "base").AssertSuccess()

	// Override LOG_LEVEL at different levels
	env.Set("LOG_LEVEL", "warn", "-a", "api", "-e", "base").AssertSuccess()           // Level 2
	env.Set("LOG_LEVEL", "info", "-a", "shared", "-e", "dev").AssertSuccess()         // Level 3
	env.Set("LOG_LEVEL", "debug", "-a", "api", "-e", "dev").AssertSuccess()           // Level 4

	// Generate for api/dev
	result := env.Generate("api", "dev", "json")
	result.AssertSuccess()
	output := result.GetStdout()

	// Verify LOG_LEVEL uses level 4
	if !strings.Contains(output, "\"LOG_LEVEL\":\"debug\"") && !strings.Contains(output, "\"LOG_LEVEL\": \"debug\"") {
		t.Error("LOG_LEVEL should be 'debug' from level 4")
	}

	// Verify TIMEOUT uses base/shared (no overrides)
	if !strings.Contains(output, "\"TIMEOUT\":\"30\"") && !strings.Contains(output, "\"TIMEOUT\": \"30\"") {
		t.Error("TIMEOUT should be '30' from base/shared")
	}
}

// TestPrecedence_TargetAppIsHighest tests that target/app overrides everything
func TestPrecedence_TargetAppIsHighest(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set values at all levels
	env.Set("PORT", "3000", "-a", "shared", "-e", "base").AssertSuccess()         // Level 1
	env.Set("PORT", "3001", "-a", "api", "-e", "base").AssertSuccess()            // Level 2
	env.Set("PORT", "3002", "-a", "shared", "-e", "dev").AssertSuccess()          // Level 3
	env.Set("PORT", "3003", "-a", "api", "-e", "dev").AssertSuccess()             // Level 4
	env.Set("PORT", "3004", "-a", "shared", "-e", "dev", "-t", "local").AssertSuccess() // Level 5
	env.Set("PORT", "3005", "-a", "api", "-e", "dev", "-t", "local").AssertSuccess()    // Level 6

	// Generate with target
	result := env.Generate("api", "dev", "json", "-t", "local")
	result.AssertSuccess()
	output := result.GetStdout()

	// Verify PORT uses level 6
	if !strings.Contains(output, "\"PORT\":\"3005\"") && !strings.Contains(output, "\"PORT\": \"3005\"") {
		t.Errorf("PORT should be '3005' from level 6 (target/app). Output: %s", output)
	}
}

// TestPrecedence_DifferentKeysAtDifferentLevels tests multiple keys with different precedence
func TestPrecedence_DifferentKeysAtDifferentLevels(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set different keys at different levels
	env.Set("KEY_LEVEL1", "value1", "-a", "shared", "-e", "base").AssertSuccess()
	env.Set("KEY_LEVEL2", "value2", "-a", "api", "-e", "base").AssertSuccess()
	env.Set("KEY_LEVEL3", "value3", "-a", "shared", "-e", "dev").AssertSuccess()
	env.Set("KEY_LEVEL4", "value4", "-a", "api", "-e", "dev").AssertSuccess()
	env.Set("KEY_LEVEL5", "value5", "-a", "shared", "-e", "dev", "-t", "local").AssertSuccess()
	env.Set("KEY_LEVEL6", "value6", "-a", "api", "-e", "dev", "-t", "local").AssertSuccess()

	// Generate with target
	result := env.Generate("api", "dev", "json", "-t", "local")
	result.AssertSuccess()
	output := result.GetStdout()

	// Verify all keys are present
	if !strings.Contains(output, "KEY_LEVEL1") || !strings.Contains(output, "value1") {
		t.Error("KEY_LEVEL1 missing or incorrect")
	}
	if !strings.Contains(output, "KEY_LEVEL2") || !strings.Contains(output, "value2") {
		t.Error("KEY_LEVEL2 missing or incorrect")
	}
	if !strings.Contains(output, "KEY_LEVEL3") || !strings.Contains(output, "value3") {
		t.Error("KEY_LEVEL3 missing or incorrect")
	}
	if !strings.Contains(output, "KEY_LEVEL4") || !strings.Contains(output, "value4") {
		t.Error("KEY_LEVEL4 missing or incorrect")
	}
	if !strings.Contains(output, "KEY_LEVEL5") || !strings.Contains(output, "value5") {
		t.Error("KEY_LEVEL5 missing or incorrect")
	}
	if !strings.Contains(output, "KEY_LEVEL6") || !strings.Contains(output, "value6") {
		t.Error("KEY_LEVEL6 missing or incorrect")
	}
}

// TestPrecedence_EnvironmentIsolation tests that environments don't interfere
func TestPrecedence_EnvironmentIsolation(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set values in dev environment
	env.Set("ENV_VAR", "dev-value", "-a", "api", "-e", "dev").AssertSuccess()

	// Set values in prod environment
	env.Set("ENV_VAR", "prod-value", "-a", "api", "-e", "prod").AssertSuccess()

	// Generate for dev
	result := env.Generate("api", "dev", "json")
	result.AssertSuccess()
	devOutput := result.GetStdout()

	// Generate for prod
	result = env.Generate("api", "prod", "json")
	result.AssertSuccess()
	prodOutput := result.GetStdout()

	// Verify isolation
	if !strings.Contains(devOutput, "dev-value") {
		t.Error("Dev environment should have dev-value")
	}
	if strings.Contains(devOutput, "prod-value") {
		t.Error("Dev environment should not have prod-value")
	}

	if !strings.Contains(prodOutput, "prod-value") {
		t.Error("Prod environment should have prod-value")
	}
	if strings.Contains(prodOutput, "dev-value") {
		t.Error("Prod environment should not have dev-value")
	}
}

// TestPrecedence_AppIsolation tests that apps don't interfere
func TestPrecedence_AppIsolation(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set values for api app
	env.Set("APP_VAR", "api-value", "-a", "api", "-e", "dev").AssertSuccess()

	// Set values for worker app
	env.Set("APP_VAR", "worker-value", "-a", "worker", "-e", "dev").AssertSuccess()

	// Generate for api
	result := env.Generate("api", "dev", "json")
	result.AssertSuccess()
	apiOutput := result.GetStdout()

	// Generate for worker
	result = env.Generate("worker", "dev", "json")
	result.AssertSuccess()
	workerOutput := result.GetStdout()

	// Verify isolation
	if !strings.Contains(apiOutput, "api-value") {
		t.Error("API app should have api-value")
	}
	if strings.Contains(apiOutput, "worker-value") {
		t.Error("API app should not have worker-value")
	}

	if !strings.Contains(workerOutput, "worker-value") {
		t.Error("Worker app should have worker-value")
	}
	if strings.Contains(workerOutput, "api-value") {
		t.Error("Worker app should not have api-value")
	}
}

// TestPrecedence_TargetIsolation tests that targets don't interfere
func TestPrecedence_TargetIsolation(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set base value
	env.Set("TARGET_VAR", "base-value", "-a", "api", "-e", "dev").AssertSuccess()

	// Set target overrides
	env.Set("TARGET_VAR", "local-value", "-a", "api", "-e", "dev", "-t", "local").AssertSuccess()
	env.Set("TARGET_VAR", "docker-value", "-a", "api", "-e", "dev", "-t", "docker").AssertSuccess()

	// Generate without target
	result := env.Generate("api", "dev", "json")
	result.AssertSuccess()
	baseOutput := result.GetStdout()

	// Generate with local target
	result = env.Generate("api", "dev", "json", "-t", "local")
	result.AssertSuccess()
	localOutput := result.GetStdout()

	// Generate with docker target
	result = env.Generate("api", "dev", "json", "-t", "docker")
	result.AssertSuccess()
	dockerOutput := result.GetStdout()

	// Verify isolation
	if !strings.Contains(baseOutput, "base-value") {
		t.Error("Base config should have base-value")
	}
	if strings.Contains(baseOutput, "local-value") || strings.Contains(baseOutput, "docker-value") {
		t.Error("Base config should not have target values")
	}

	if !strings.Contains(localOutput, "local-value") {
		t.Error("Local target should have local-value")
	}
	if strings.Contains(localOutput, "docker-value") {
		t.Error("Local target should not have docker-value")
	}

	if !strings.Contains(dockerOutput, "docker-value") {
		t.Error("Docker target should have docker-value")
	}
	if strings.Contains(dockerOutput, "local-value") {
		t.Error("Docker target should not have local-value")
	}
}

// TestPrecedence_SharedAcrossApps tests that shared config is inherited
func TestPrecedence_SharedAcrossApps(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set shared values
	env.Set("COMPANY", "Acme Corp", "-a", "shared", "-e", "base").AssertSuccess()
	env.Set("REGION", "us-east-1", "-a", "shared", "-e", "dev").AssertSuccess()

	// Set app-specific values
	env.Set("APP_NAME", "api-service", "-a", "api", "-e", "dev").AssertSuccess()
	env.Set("APP_NAME", "worker-service", "-a", "worker", "-e", "dev").AssertSuccess()

	// Generate for api
	result := env.Generate("api", "dev", "json")
	result.AssertSuccess()
	apiOutput := result.GetStdout()

	// Generate for worker
	result = env.Generate("worker", "dev", "json")
	result.AssertSuccess()
	workerOutput := result.GetStdout()

	// Verify both apps get shared values
	if !strings.Contains(apiOutput, "Acme Corp") {
		t.Error("API app should inherit COMPANY from shared")
	}
	if !strings.Contains(apiOutput, "us-east-1") {
		t.Error("API app should inherit REGION from shared")
	}
	if !strings.Contains(workerOutput, "Acme Corp") {
		t.Error("Worker app should inherit COMPANY from shared")
	}
	if !strings.Contains(workerOutput, "us-east-1") {
		t.Error("Worker app should inherit REGION from shared")
	}

	// Verify app-specific values are isolated
	if !strings.Contains(apiOutput, "api-service") {
		t.Error("API app should have its APP_NAME")
	}
	if strings.Contains(apiOutput, "worker-service") {
		t.Error("API app should not have worker's APP_NAME")
	}
}

// TestPrecedence_AppOverridesShared tests that app config overrides shared
func TestPrecedence_AppOverridesShared(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set shared default
	env.Set("TIMEOUT", "30", "-a", "shared", "-e", "dev").AssertSuccess()

	// Override for specific app
	env.Set("TIMEOUT", "60", "-a", "api", "-e", "dev").AssertSuccess()

	// Generate for api (should use override)
	result := env.Generate("api", "dev", "json")
	result.AssertSuccess()
	apiOutput := result.GetStdout()

	if !strings.Contains(apiOutput, "\"TIMEOUT\":\"60\"") && !strings.Contains(apiOutput, "\"TIMEOUT\": \"60\"") {
		t.Error("API app should override shared TIMEOUT")
	}

	// Generate for another app (should use shared default)
	result = env.Generate("worker", "dev", "json")
	result.AssertSuccess()
	workerOutput := result.GetStdout()

	if !strings.Contains(workerOutput, "\"TIMEOUT\":\"30\"") && !strings.Contains(workerOutput, "\"TIMEOUT\": \"30\"") {
		t.Error("Worker app should use shared TIMEOUT")
	}
}

// TestPrecedence_ComplexScenario tests a realistic complex configuration
func TestPrecedence_ComplexScenario(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Company-wide defaults (base/shared)
	env.Set("COMPANY_NAME", "Acme Corp", "-a", "shared", "-e", "base").AssertSuccess()
	env.Set("LOG_LEVEL", "error", "-a", "shared", "-e", "base").AssertSuccess()
	env.Set("TIMEOUT", "30", "-a", "shared", "-e", "base").AssertSuccess()

	// API service defaults (base/api)
	env.Set("API_VERSION", "v1", "-a", "api", "-e", "base").AssertSuccess()
	env.Set("LOG_LEVEL", "warn", "-a", "api", "-e", "base").AssertSuccess()

	// Dev environment defaults (dev/shared)
	env.Set("DB_HOST", "dev-db.local", "-a", "shared", "-e", "dev").AssertSuccess()
	env.Set("LOG_LEVEL", "info", "-a", "shared", "-e", "dev").AssertSuccess()

	// API in dev (dev/api)
	env.Set("API_PORT", "3000", "-a", "api", "-e", "dev").AssertSuccess()
	env.Set("LOG_LEVEL", "debug", "-a", "api", "-e", "dev").AssertSuccess()

	// Local target overrides (target/shared)
	env.Set("DB_HOST", "localhost", "-a", "shared", "-e", "dev", "-t", "local").AssertSuccess()

	// API local target (target/api)
	env.Set("API_PORT", "8080", "-a", "api", "-e", "dev", "-t", "local").AssertSuccess()

	// Generate for api/dev without target
	result := env.Generate("api", "dev", "json")
	result.AssertSuccess()
	baseOutput := result.GetStdout()

	// Verify expected values
	if !strings.Contains(baseOutput, "Acme Corp") {
		t.Error("Should inherit COMPANY_NAME from base/shared")
	}
	if !strings.Contains(baseOutput, "\"LOG_LEVEL\":\"debug\"") && !strings.Contains(baseOutput, "\"LOG_LEVEL\": \"debug\"") {
		t.Error("Should use LOG_LEVEL from dev/api (level 4)")
	}
	if !strings.Contains(baseOutput, "\"TIMEOUT\":\"30\"") && !strings.Contains(baseOutput, "\"TIMEOUT\": \"30\"") {
		t.Error("Should inherit TIMEOUT from base/shared")
	}
	if !strings.Contains(baseOutput, "v1") {
		t.Error("Should inherit API_VERSION from base/api")
	}
	if !strings.Contains(baseOutput, "dev-db.local") {
		t.Error("Should use DB_HOST from dev/shared")
	}
	if !strings.Contains(baseOutput, "\"API_PORT\":\"3000\"") && !strings.Contains(baseOutput, "\"API_PORT\": \"3000\"") {
		t.Error("Should use API_PORT from dev/api")
	}

	// Generate for api/dev with local target
	result = env.Generate("api", "dev", "json", "-t", "local")
	result.AssertSuccess()
	localOutput := result.GetStdout()

	// Verify target overrides
	if !strings.Contains(localOutput, "localhost") {
		t.Error("Should use DB_HOST from target/shared override")
	}
	if !strings.Contains(localOutput, "\"API_PORT\":\"8080\"") && !strings.Contains(localOutput, "\"API_PORT\": \"8080\"") {
		t.Error("Should use API_PORT from target/api override")
	}

	// Other values should remain the same
	if !strings.Contains(localOutput, "Acme Corp") {
		t.Error("Should still inherit COMPANY_NAME")
	}
	if !strings.Contains(localOutput, "\"LOG_LEVEL\":\"debug\"") && !strings.Contains(localOutput, "\"LOG_LEVEL\": \"debug\"") {
		t.Error("Should still use LOG_LEVEL from dev/api")
	}
}

// TestPrecedence_PartialOverrides tests that only specified keys are overridden
func TestPrecedence_PartialOverrides(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set multiple keys at base level
	env.Set("KEY1", "base1", "-a", "api", "-e", "dev").AssertSuccess()
	env.Set("KEY2", "base2", "-a", "api", "-e", "dev").AssertSuccess()
	env.Set("KEY3", "base3", "-a", "api", "-e", "dev").AssertSuccess()

	// Override only one key at target level
	env.Set("KEY2", "override2", "-a", "api", "-e", "dev", "-t", "local").AssertSuccess()

	// Generate with target
	result := env.Generate("api", "dev", "json", "-t", "local")
	result.AssertSuccess()
	output := result.GetStdout()

	// Verify KEY1 and KEY3 use base values
	if !strings.Contains(output, "base1") {
		t.Error("KEY1 should use base value")
	}
	if !strings.Contains(output, "base3") {
		t.Error("KEY3 should use base value")
	}

	// Verify KEY2 uses override
	if !strings.Contains(output, "override2") {
		t.Error("KEY2 should use override value")
	}
	if strings.Contains(output, "base2") {
		t.Error("KEY2 should not use base value")
	}
}
