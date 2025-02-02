package autocert

import (
	"github.com/go-acme/lego/v4/providers/dns/clouddns"
	"github.com/go-acme/lego/v4/providers/dns/cloudflare"
	"github.com/go-acme/lego/v4/providers/dns/duckdns"
	"github.com/go-acme/lego/v4/providers/dns/ovh"
)

const (
	certBasePath     = "certs/"
	CertFileDefault  = certBasePath + "cert.crt"
	KeyFileDefault   = certBasePath + "priv.key"
	RegistrationFile = certBasePath + "registration.json"
)

const (
	ProviderLocal      = "local"
	ProviderCloudflare = "cloudflare"
	ProviderClouddns   = "clouddns"
	ProviderDuckdns    = "duckdns"
	ProviderOVH        = "ovh"
)

var providersGenMap = map[string]ProviderGenerator{
	ProviderLocal:      providerGenerator(NewDummyDefaultConfig, NewDummyDNSProviderConfig),
	ProviderCloudflare: providerGenerator(cloudflare.NewDefaultConfig, cloudflare.NewDNSProviderConfig),
	ProviderClouddns:   providerGenerator(clouddns.NewDefaultConfig, clouddns.NewDNSProviderConfig),
	ProviderDuckdns:    providerGenerator(duckdns.NewDefaultConfig, duckdns.NewDNSProviderConfig),
	ProviderOVH:        providerGenerator(ovh.NewDefaultConfig, ovh.NewDNSProviderConfig),
}
