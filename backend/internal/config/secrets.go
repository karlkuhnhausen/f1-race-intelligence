// Package config provides configuration loading including Key Vault secrets.
package config

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
)

// Config holds application configuration resolved from env + Key Vault.
type Config struct {
	Port                  string
	CosmosAccountEndpoint string
	KeyVaultURI           string
	Season                int
}

// Load reads configuration from environment variables.
func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	season := 2026

	return Config{
		Port:                  port,
		CosmosAccountEndpoint: os.Getenv("COSMOS_ACCOUNT_ENDPOINT"),
		KeyVaultURI:           os.Getenv("KEYVAULT_URI"),
		Season:                season,
	}
}

// SecretClient creates an Azure Key Vault secrets client using DefaultAzureCredential.
func SecretClient(vaultURI string) (*azsecrets.Client, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("config: credential: %w", err)
	}

	client, err := azsecrets.NewClient(vaultURI, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("config: keyvault client: %w", err)
	}

	return client, nil
}

// GetSecret retrieves a single secret value from Key Vault.
func GetSecret(ctx context.Context, client *azsecrets.Client, name string, logger *slog.Logger) (string, error) {
	resp, err := client.GetSecret(ctx, name, "", nil)
	if err != nil {
		return "", fmt.Errorf("config: get secret %q: %w", name, err)
	}

	if resp.Value == nil {
		return "", fmt.Errorf("config: secret %q has nil value", name)
	}

	logger.Info("secret loaded from key vault", "name", name)
	return *resp.Value, nil
}
