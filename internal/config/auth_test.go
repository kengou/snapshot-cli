package config

import (
	"strings"
	"testing"
)

// helpers

func allEnvVars() map[string]string {
	return map[string]string{
		"OS_AUTH_URL":            "https://keystone.example.com/v3",
		"OS_USERNAME":            "testuser",
		"OS_PASSWORD":            "testpass",
		"OS_USER_DOMAIN_NAME":    "Default",
		"OS_PROJECT_NAME":        "myproject",
		"OS_PROJECT_DOMAIN_NAME": "Default",
		"OS_REGION_NAME":         "RegionOne",
	}
}

func setEnv(t *testing.T, vars map[string]string) {
	t.Helper()
	for k, v := range vars {
		t.Setenv(k, v)
	}
}

// Auth.verify

func TestVerify_AllFieldsSet_ReturnsNil(t *testing.T) {
	a := &Auth{
		AuthURL:           "https://keystone.example.com/v3",
		Username:          "user",
		Password:          "pass",
		UserDomainName:    "Default",
		ProjectName:       "proj",
		ProjectDomainName: "Default",
	}
	if err := a.verify(); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestVerify_MissingAuthURL(t *testing.T) {
	a := &Auth{
		Username:          "user",
		Password:          "pass",
		UserDomainName:    "Default",
		ProjectName:       "proj",
		ProjectDomainName: "Default",
	}
	err := a.verify()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "OS_AUTH_URL") {
		t.Errorf("expected OS_AUTH_URL in error, got: %v", err)
	}
}

func TestVerify_MissingPassword(t *testing.T) {
	a := &Auth{
		AuthURL:           "https://keystone.example.com/v3",
		Username:          "user",
		UserDomainName:    "Default",
		ProjectName:       "proj",
		ProjectDomainName: "Default",
	}
	err := a.verify()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "OS_PASSWORD") {
		t.Errorf("expected OS_PASSWORD in error, got: %v", err)
	}
}

func TestVerify_MultipleFieldsMissing_ListsAll(t *testing.T) {
	a := &Auth{}
	err := a.verify()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	for _, field := range []string{"OS_AUTH_URL", "OS_USERNAME", "OS_PASSWORD", "OS_USER_DOMAIN_NAME", "OS_PROJECT_NAME", "OS_PROJECT_DOMAIN_NAME"} {
		if !strings.Contains(err.Error(), field) {
			t.Errorf("expected %s in error message, got: %v", field, err)
		}
	}
}

func TestVerify_RegionNotRequired(t *testing.T) {
	// RegionName is optional — verify should pass without it
	a := &Auth{
		AuthURL:           "https://keystone.example.com/v3",
		Username:          "user",
		Password:          "pass",
		UserDomainName:    "Default",
		ProjectName:       "proj",
		ProjectDomainName: "Default",
		// RegionName intentionally empty
	}
	if err := a.verify(); err != nil {
		t.Errorf("expected nil (region optional), got %v", err)
	}
}

// ReadAuthConfig

func TestReadAuthConfig_AllEnvVarsSet(t *testing.T) {
	setEnv(t, allEnvVars())

	cfg, err := ReadAuthConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AuthURL != "https://keystone.example.com/v3" {
		t.Errorf("AuthURL = %q, want %q", cfg.AuthURL, "https://keystone.example.com/v3")
	}
	if cfg.Username != "testuser" {
		t.Errorf("Username = %q, want %q", cfg.Username, "testuser")
	}
	if cfg.Password != "testpass" {
		t.Errorf("Password = %q, want %q", cfg.Password, "testpass")
	}
	if cfg.UserDomainName != "Default" {
		t.Errorf("UserDomainName = %q, want %q", cfg.UserDomainName, "Default")
	}
	if cfg.ProjectName != "myproject" {
		t.Errorf("ProjectName = %q, want %q", cfg.ProjectName, "myproject")
	}
	if cfg.ProjectDomainName != "Default" {
		t.Errorf("ProjectDomainName = %q, want %q", cfg.ProjectDomainName, "Default")
	}
	if cfg.RegionName != "RegionOne" {
		t.Errorf("RegionName = %q, want %q", cfg.RegionName, "RegionOne")
	}
}

func TestReadAuthConfig_MissingEnvVars_ReturnsError(t *testing.T) {
	// Unset all relevant vars by not setting them (t.Setenv clears on cleanup)
	for k := range allEnvVars() {
		t.Setenv(k, "")
	}

	_, err := ReadAuthConfig()
	if err == nil {
		t.Fatal("expected error when env vars missing, got nil")
	}
}

func TestReadAuthConfig_PartialEnvVars_ErrorListsMissing(t *testing.T) {
	t.Setenv("OS_AUTH_URL", "https://keystone.example.com/v3")
	t.Setenv("OS_USERNAME", "user")
	t.Setenv("OS_PASSWORD", "")
	t.Setenv("OS_USER_DOMAIN_NAME", "")
	t.Setenv("OS_PROJECT_NAME", "")
	t.Setenv("OS_PROJECT_DOMAIN_NAME", "")

	_, err := ReadAuthConfig()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "OS_PASSWORD") {
		t.Errorf("expected OS_PASSWORD in error, got: %v", err)
	}
}

func TestReadAuthConfig_ReturnsNonNilConfigEvenOnError(t *testing.T) {
	for k := range allEnvVars() {
		t.Setenv(k, "")
	}

	cfg, err := ReadAuthConfig()
	if err == nil {
		t.Error("expected error when all env vars are empty")
	}
	if cfg == nil {
		t.Error("expected non-nil *Auth even when validation fails")
	}
}

// FuzzAuthVerify verifies that Auth.verify never panics for any combination of field values
// and that it only returns nil when all required fields are non-empty.
func FuzzAuthVerify(f *testing.F) {
	// Seed corpus: all-valid, all-empty, partial fills
	f.Add("https://keystone.example.com/v3", "user", "pass", "Default", "proj", "Default")
	f.Add("", "", "", "", "", "")
	f.Add("https://keystone.example.com/v3", "", "pass", "Default", "proj", "Default")
	f.Add("url", "user", "pass", "domain", "proj", "")

	f.Fuzz(func(t *testing.T, authURL, username, password, userDomain, project, projectDomain string) {
		a := &Auth{
			AuthURL:           authURL,
			Username:          username,
			Password:          password,
			UserDomainName:    userDomain,
			ProjectName:       project,
			ProjectDomainName: projectDomain,
		}
		err := a.verify()
		allSet := authURL != "" && username != "" && password != "" &&
			userDomain != "" && project != "" && projectDomain != ""
		if allSet && err != nil {
			t.Errorf("verify() returned error when all fields set: %v", err)
		}
		if !allSet && err == nil {
			t.Errorf("verify() returned nil when one or more required fields are empty")
		}
	})
}
