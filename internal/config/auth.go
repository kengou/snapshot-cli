package config

import (
	"errors"
	"os"
	"strings"
)

// Auth holds OpenStack authentication parameters sourced from OS_* environment variables.
type Auth struct {
	AuthURL           string
	RegionName        string
	Username          string
	UserDomainName    string
	Password          string
	ProjectName       string
	ProjectDomainName string
}

// ReadAuthConfig reads the OS_* environment variables and returns the Auth
// config, or an error listing any required variables that are missing.
func ReadAuthConfig() (authConfig *Auth, err error) {
	authConfig = readEnv()
	return authConfig, authConfig.verify() //nolint:gocritic
}

// readEnv reads the environment variables for OpenStack authentication.
func readEnv() *Auth {
	return &Auth{
		AuthURL:           os.Getenv("OS_AUTH_URL"),
		Username:          os.Getenv("OS_USERNAME"),
		Password:          os.Getenv("OS_PASSWORD"),
		UserDomainName:    os.Getenv("OS_USER_DOMAIN_NAME"),
		ProjectName:       os.Getenv("OS_PROJECT_NAME"),
		ProjectDomainName: os.Getenv("OS_PROJECT_DOMAIN_NAME"),
		RegionName:        os.Getenv("OS_REGION_NAME"),
	}
}

// verify checks if all required OpenStack authentication parameters are set.
func (a *Auth) verify() error {
	errs := make([]string, 0)
	if a.AuthURL == "" {
		errs = append(errs, "OS_AUTH_URL")
	}
	if a.Username == "" {
		errs = append(errs, "OS_USERNAME")
	}
	if a.Password == "" {
		errs = append(errs, "OS_PASSWORD")
	}
	if a.UserDomainName == "" {
		errs = append(errs, "OS_USER_DOMAIN_NAME")
	}
	if a.ProjectName == "" {
		errs = append(errs, "OS_PROJECT_NAME")
	}
	if a.ProjectDomainName == "" {
		errs = append(errs, "OS_PROJECT_DOMAIN_NAME")
	}

	if len(errs) > 0 {
		return errors.New("missing " + strings.Join(errs, ", "))
	}
	return nil
}
