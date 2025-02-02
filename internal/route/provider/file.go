package provider

import (
	"os"
	"path"

	"github.com/rs/zerolog"
	"github.com/yusing/go-proxy/internal/common"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/proxy/entry"
	R "github.com/yusing/go-proxy/internal/route"
	U "github.com/yusing/go-proxy/internal/utils"
	W "github.com/yusing/go-proxy/internal/watcher"
)

type FileProvider struct {
	fileName string
	path     string
	l        zerolog.Logger
}

func FileProviderImpl(filename string) (ProviderImpl, error) {
	impl := &FileProvider{
		fileName: filename,
		path:     path.Join(common.ConfigBasePath, filename),
		l:        logger.With().Str("type", "file").Str("name", filename).Logger(),
	}
	_, err := os.Stat(impl.path)
	if err != nil {
		return nil, err
	}
	return impl, nil
}

func Validate(data []byte) E.Error {
	return U.ValidateYaml(U.GetSchema(common.FileProviderSchemaPath), data)
}

func (p *FileProvider) String() string {
	return p.fileName
}

func (p *FileProvider) Logger() *zerolog.Logger {
	return &p.l
}

func (p *FileProvider) loadRoutesImpl() (R.Routes, E.Error) {
	routes := R.NewRoutes()
	entries := entry.NewProxyEntries()

	data, err := os.ReadFile(p.path)
	if err != nil {
		return routes, E.From(err)
	}

	if err := entries.UnmarshalFromYAML(data); err != nil {
		return routes, E.From(err)
	}

	if err := Validate(data); err != nil {
		E.LogWarn(p.fileName+": validation failure", err)
	}

	return R.FromEntries(entries)
}

func (p *FileProvider) NewWatcher() W.Watcher {
	return W.NewConfigFileWatcher(p.fileName)
}
