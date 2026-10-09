package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories wires the provider under test into the
// terraform-plugin-testing harness. Resource tests in later tasks reuse it.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"ravenna": providerserver.NewProtocol6WithError(New("test")()),
}

func TestProvider_SchemaIsValid(t *testing.T) {
	p := New("test")()

	resp := &provider.SchemaResponse{}
	p.Schema(context.Background(), provider.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	tokenAttr, ok := resp.Schema.Attributes["api_token"]
	if !ok {
		t.Fatal("provider schema has no api_token attribute")
	}
	if !tokenAttr.IsSensitive() {
		t.Error("api_token is not marked Sensitive; the token must never appear in logs or plan output")
	}
}

func TestProvider_Metadata(t *testing.T) {
	p := New("1.2.3")()

	resp := &provider.MetadataResponse{}
	p.Metadata(context.Background(), provider.MetadataRequest{}, resp)

	if resp.TypeName != "ravenna" {
		t.Errorf("TypeName = %q, want ravenna", resp.TypeName)
	}
	if resp.Version != "1.2.3" {
		t.Errorf("Version = %q, want 1.2.3", resp.Version)
	}
}
