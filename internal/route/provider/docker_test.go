package provider

import (
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/yusing/go-proxy/internal/common"
	D "github.com/yusing/go-proxy/internal/docker"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/proxy/entry"
	T "github.com/yusing/go-proxy/internal/proxy/fields"
	. "github.com/yusing/go-proxy/internal/utils/testing"
)

var (
	dummyNames = []string{"/a"}
	p          DockerProvider
)

func TestApplyLabel(t *testing.T) {
	pathPatterns := `
- /
- POST /upload/{$}
- GET /static
`[1:]
	pathPatternsExpect := []string{
		"/",
		"POST /upload/{$}",
		"GET /static",
	}
	middlewaresExpect := D.NestedLabelMap{
		"middleware1": {
			"prop1": "value1",
			"prop2": "value2",
		},
		"middleware2": {
			"prop3": "value3",
			"prop4": "value4",
		},
	}
	var p DockerProvider
	entries, err := p.entriesFromContainerLabels(D.FromDocker(&types.Container{
		Names: dummyNames,
		Labels: map[string]string{
			D.LabelAliases:                          "a,b",
			D.LabelIdleTimeout:                      "",
			D.LabelStopMethod:                       common.StopMethodDefault,
			D.LabelStopSignal:                       "SIGTERM",
			D.LabelStopTimeout:                      common.StopTimeoutDefault,
			D.LabelWakeTimeout:                      common.WakeTimeoutDefault,
			"proxy.*.no_tls_verify":                 "true",
			"proxy.*.scheme":                        "https",
			"proxy.*.host":                          "app",
			"proxy.*.port":                          "4567",
			"proxy.a.no_tls_verify":                 "true",
			"proxy.a.path_patterns":                 pathPatterns,
			"proxy.a.middlewares.middleware1.prop1": "value1",
			"proxy.a.middlewares.middleware1.prop2": "value2",
			"proxy.a.middlewares.middleware2.prop3": "value3",
			"proxy.a.middlewares.middleware2.prop4": "value4",
			"proxy.a.homepage.show":                 "true",
			"proxy.a.homepage.icon":                 "png/example.png",
			"proxy.a.healthcheck.path":              "/ping",
			"proxy.a.healthcheck.interval":          "10s",
		},
	}, client.DefaultDockerHost))
	ExpectNoError(t, err)

	a, ok := entries.Load("a")
	ExpectTrue(t, ok)
	b, ok := entries.Load("b")
	ExpectTrue(t, ok)

	ExpectEqual(t, a.Scheme, "https")
	ExpectEqual(t, b.Scheme, "https")

	ExpectEqual(t, a.Host, "app")
	ExpectEqual(t, b.Host, "app")

	ExpectEqual(t, a.Port, "4567")
	ExpectEqual(t, b.Port, "4567")

	ExpectTrue(t, a.NoTLSVerify)
	ExpectTrue(t, b.NoTLSVerify)

	ExpectDeepEqual(t, a.PathPatterns, pathPatternsExpect)
	ExpectEqual(t, len(b.PathPatterns), 0)

	ExpectDeepEqual(t, a.Middlewares, middlewaresExpect)
	ExpectEqual(t, len(b.Middlewares), 0)

	ExpectEqual(t, a.Container.IdleTimeout, "")
	ExpectEqual(t, b.Container.IdleTimeout, "")

	ExpectEqual(t, a.Container.StopTimeout, common.StopTimeoutDefault)
	ExpectEqual(t, b.Container.StopTimeout, common.StopTimeoutDefault)

	ExpectEqual(t, a.Container.StopMethod, common.StopMethodDefault)
	ExpectEqual(t, b.Container.StopMethod, common.StopMethodDefault)

	ExpectEqual(t, a.Container.WakeTimeout, common.WakeTimeoutDefault)
	ExpectEqual(t, b.Container.WakeTimeout, common.WakeTimeoutDefault)

	ExpectEqual(t, a.Container.StopSignal, "SIGTERM")
	ExpectEqual(t, b.Container.StopSignal, "SIGTERM")

	ExpectEqual(t, a.Homepage.Show, true)
	ExpectEqual(t, a.Homepage.Icon, "png/example.png")

	ExpectEqual(t, a.HealthCheck.Path, "/ping")
	ExpectEqual(t, a.HealthCheck.Interval, 10*time.Second)
}

func TestApplyLabelWithAlias(t *testing.T) {
	entries, err := p.entriesFromContainerLabels(D.FromDocker(&types.Container{
		Names: dummyNames,
		State: "running",
		Labels: map[string]string{
			D.LabelAliases:          "a,b,c",
			"proxy.a.no_tls_verify": "true",
			"proxy.a.port":          "3333",
			"proxy.b.port":          "1234",
			"proxy.c.scheme":        "https",
		},
	}, client.DefaultDockerHost))
	a, ok := entries.Load("a")
	ExpectTrue(t, ok)
	b, ok := entries.Load("b")
	ExpectTrue(t, ok)
	c, ok := entries.Load("c")
	ExpectTrue(t, ok)

	ExpectNoError(t, err)
	ExpectEqual(t, a.Scheme, "http")
	ExpectEqual(t, a.Port, "3333")
	ExpectEqual(t, a.NoTLSVerify, true)
	ExpectEqual(t, b.Scheme, "http")
	ExpectEqual(t, b.Port, "1234")
	ExpectEqual(t, c.Scheme, "https")
}

func TestApplyLabelWithRef(t *testing.T) {
	entries := E.Must(p.entriesFromContainerLabels(D.FromDocker(&types.Container{
		Names: dummyNames,
		State: "running",
		Labels: map[string]string{
			D.LabelAliases:    "a,b,c",
			"proxy.#1.host":   "localhost",
			"proxy.#1.port":   "4444",
			"proxy.#2.port":   "9999",
			"proxy.#3.port":   "1111",
			"proxy.#3.scheme": "https",
		},
	}, client.DefaultDockerHost)))
	a, ok := entries.Load("a")
	ExpectTrue(t, ok)
	b, ok := entries.Load("b")
	ExpectTrue(t, ok)
	c, ok := entries.Load("c")
	ExpectTrue(t, ok)

	ExpectEqual(t, a.Scheme, "http")
	ExpectEqual(t, a.Host, "localhost")
	ExpectEqual(t, a.Port, "4444")
	ExpectEqual(t, b.Port, "9999")
	ExpectEqual(t, c.Scheme, "https")
	ExpectEqual(t, c.Port, "1111")
}

func TestApplyLabelWithRefIndexError(t *testing.T) {
	c := D.FromDocker(&types.Container{
		Names: dummyNames,
		State: "running",
		Labels: map[string]string{
			D.LabelAliases:    "a,b",
			"proxy.#1.host":   "localhost",
			"proxy.#4.scheme": "https",
		},
	}, "")
	_, err := p.entriesFromContainerLabels(c)
	ExpectError(t, ErrAliasRefIndexOutOfRange, err)

	_, err = p.entriesFromContainerLabels(D.FromDocker(&types.Container{
		Names: dummyNames,
		State: "running",
		Labels: map[string]string{
			D.LabelAliases:  "a,b",
			"proxy.#0.host": "localhost",
		},
	}, ""))
	ExpectError(t, ErrAliasRefIndexOutOfRange, err)
}

func TestDynamicAliases(t *testing.T) {
	c := D.FromDocker(&types.Container{
		Names: []string{"app1"},
		State: "running",
		Labels: map[string]string{
			"proxy.app1.port":         "1234",
			"proxy.app1_backend.port": "5678",
		},
	}, client.DefaultDockerHost)

	raw, ok := E.Must(p.entriesFromContainerLabels(c)).Load("app1")
	ExpectTrue(t, ok)
	ExpectEqual(t, raw.Scheme, "http")
	ExpectEqual(t, raw.Port, "1234")

	raw, ok = E.Must(p.entriesFromContainerLabels(c)).Load("app1_backend")
	ExpectTrue(t, ok)
	ExpectEqual(t, raw.Scheme, "http")
	ExpectEqual(t, raw.Port, "5678")
}

func TestPublicIPLocalhost(t *testing.T) {
	c := D.FromDocker(&types.Container{Names: dummyNames, State: "running"}, client.DefaultDockerHost)
	raw, ok := E.Must(p.entriesFromContainerLabels(c)).Load("a")
	ExpectTrue(t, ok)
	ExpectEqual(t, raw.Container.PublicIP, "127.0.0.1")
	ExpectEqual(t, raw.Host, raw.Container.PublicIP)
}

func TestPublicIPRemote(t *testing.T) {
	c := D.FromDocker(&types.Container{Names: dummyNames, State: "running"}, "tcp://1.2.3.4:2375")
	raw, ok := E.Must(p.entriesFromContainerLabels(c)).Load("a")
	ExpectTrue(t, ok)
	ExpectEqual(t, raw.Container.PublicIP, "1.2.3.4")
	ExpectEqual(t, raw.Host, raw.Container.PublicIP)
}

func TestPrivateIPLocalhost(t *testing.T) {
	c := D.FromDocker(&types.Container{
		Names: dummyNames,
		NetworkSettings: &types.SummaryNetworkSettings{
			Networks: map[string]*network.EndpointSettings{
				"network": {
					IPAddress: "172.17.0.123",
				},
			},
		},
	}, client.DefaultDockerHost)
	raw, ok := E.Must(p.entriesFromContainerLabels(c)).Load("a")
	ExpectTrue(t, ok)
	ExpectEqual(t, raw.Container.PrivateIP, "172.17.0.123")
	ExpectEqual(t, raw.Host, raw.Container.PrivateIP)
}

func TestPrivateIPRemote(t *testing.T) {
	c := D.FromDocker(&types.Container{
		Names: dummyNames,
		State: "running",
		NetworkSettings: &types.SummaryNetworkSettings{
			Networks: map[string]*network.EndpointSettings{
				"network": {
					IPAddress: "172.17.0.123",
				},
			},
		},
	}, "tcp://1.2.3.4:2375")
	raw, ok := E.Must(p.entriesFromContainerLabels(c)).Load("a")
	ExpectTrue(t, ok)
	ExpectEqual(t, raw.Container.PrivateIP, "")
	ExpectEqual(t, raw.Container.PublicIP, "1.2.3.4")
	ExpectEqual(t, raw.Host, raw.Container.PublicIP)
}

func TestStreamDefaultValues(t *testing.T) {
	privPort := uint16(1234)
	pubPort := uint16(4567)
	privIP := "172.17.0.123"
	cont := &types.Container{
		Names: []string{"a"},
		State: "running",
		NetworkSettings: &types.SummaryNetworkSettings{
			Networks: map[string]*network.EndpointSettings{
				"network": {
					IPAddress: privIP,
				},
			},
		},
		Ports: []types.Port{
			{Type: "udp", PrivatePort: privPort, PublicPort: pubPort},
		},
	}

	t.Run("local", func(t *testing.T) {
		c := D.FromDocker(cont, client.DefaultDockerHost)

		raw, ok := E.Must(p.entriesFromContainerLabels(c)).Load("a")
		ExpectTrue(t, ok)
		en := E.Must(entry.ValidateEntry(raw))
		a := ExpectType[*entry.StreamEntry](t, en)
		ExpectEqual(t, a.Scheme.ListeningScheme, T.Scheme("udp"))
		ExpectEqual(t, a.Scheme.ProxyScheme, T.Scheme("udp"))
		ExpectEqual(t, a.Host, T.Host(privIP))
		ExpectEqual(t, a.Port.ListeningPort, 0)
		ExpectEqual(t, a.Port.ProxyPort, T.Port(privPort))
	})

	t.Run("remote", func(t *testing.T) {
		c := D.FromDocker(cont, "tcp://1.2.3.4:2375")
		raw, ok := E.Must(p.entriesFromContainerLabels(c)).Load("a")
		ExpectTrue(t, ok)
		en := E.Must(entry.ValidateEntry(raw))
		a := ExpectType[*entry.StreamEntry](t, en)
		ExpectEqual(t, a.Scheme.ListeningScheme, T.Scheme("udp"))
		ExpectEqual(t, a.Scheme.ProxyScheme, T.Scheme("udp"))
		ExpectEqual(t, a.Host, "1.2.3.4")
		ExpectEqual(t, a.Port.ListeningPort, 0)
		ExpectEqual(t, a.Port.ProxyPort, T.Port(pubPort))
	})
}

func TestExplicitExclude(t *testing.T) {
	_, ok := E.Must(p.entriesFromContainerLabels(D.FromDocker(&types.Container{
		Names: dummyNames,
		Labels: map[string]string{
			D.LabelAliases:          "a",
			D.LabelExclude:          "true",
			"proxy.a.no_tls_verify": "true",
		},
	}, ""))).Load("a")
	ExpectFalse(t, ok)
}

func TestImplicitExcludeDatabase(t *testing.T) {
	t.Run("mount path detection", func(t *testing.T) {
		_, ok := E.Must(p.entriesFromContainerLabels(D.FromDocker(&types.Container{
			Names: dummyNames,
			Mounts: []types.MountPoint{
				{Source: "/data", Destination: "/var/lib/postgresql/data"},
			},
		}, ""))).Load("a")
		ExpectFalse(t, ok)
	})
	t.Run("exposed port detection", func(t *testing.T) {
		_, ok := E.Must(p.entriesFromContainerLabels(D.FromDocker(&types.Container{
			Names: dummyNames,
			Ports: []types.Port{
				{Type: "tcp", PrivatePort: 5432, PublicPort: 5432},
			},
		}, ""))).Load("a")
		ExpectFalse(t, ok)
	})
}
