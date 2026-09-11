package config

import (
	"strings"
	"testing"
)

func TestFileOwnershipTenantIDEnvironment(t *testing.T) {
	t.Run("unset is disabled", func(t *testing.T) {
		t.Setenv("WEKNORA_TENANT_FILE_OWNERSHIP_TENANT_ID", "")
		cfg := &Config{Tenant: &TenantConfig{}}
		applyAuthAndTenantDefaults(cfg)
		if cfg.Tenant.FileOwnershipTenantID != 0 || cfg.Tenant.IsFileOwnershipEnabled(42) {
			t.Fatalf("unexpected enabled policy: %+v", cfg.Tenant)
		}
	})

	t.Run("valid tenant is selected", func(t *testing.T) {
		t.Setenv("WEKNORA_TENANT_FILE_OWNERSHIP_TENANT_ID", "42")
		cfg := &Config{Tenant: &TenantConfig{}}
		applyAuthAndTenantDefaults(cfg)
		if !cfg.Tenant.IsFileOwnershipEnabled(42) || cfg.Tenant.IsFileOwnershipEnabled(7) {
			t.Fatalf("managed tenant selection mismatch: %+v", cfg.Tenant)
		}
		if err := ValidateConfig(cfg); err != nil {
			t.Fatalf("valid config rejected: %v", err)
		}
	})

	t.Run("invalid value disables and fails validation", func(t *testing.T) {
		t.Setenv("WEKNORA_TENANT_FILE_OWNERSHIP_TENANT_ID", "not-a-tenant")
		cfg := &Config{Tenant: &TenantConfig{FileOwnershipTenantID: 42}}
		applyAuthAndTenantDefaults(cfg)
		if cfg.Tenant.FileOwnershipTenantID != 0 || cfg.Tenant.IsFileOwnershipEnabled(42) {
			t.Fatal("invalid environment value must fail closed")
		}
		err := ValidateConfig(cfg)
		if err == nil || !strings.Contains(err.Error(), "WEKNORA_TENANT_FILE_OWNERSHIP_TENANT_ID") {
			t.Fatalf("validation error = %v", err)
		}
	})
}
