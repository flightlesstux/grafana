package finder

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/grafana/grafana/pkg/infra/fs"
	"github.com/grafana/grafana/pkg/plugins"
	"github.com/grafana/grafana/pkg/plugins/log"
	"github.com/grafana/grafana/pkg/util"
)

var walk = util.Walk

var (
	ErrInvalidPluginJSON         = errors.New("did not find valid type or id properties in plugin.json")
	ErrInvalidPluginJSONFilePath = errors.New("invalid plugin.json filepath was provided")
)

type Local struct {
	paths []string
	log   log.Logger
}

func NewLocalFinder(paths []string) *Local {
	return &Local{
		paths: paths,
		log:   log.New("finder.local"),
	}
}

func (l *Local) Find(_ context.Context) ([]*plugins.FoundBundle, error) {
	var pluginJSONPaths []string
	for _, path := range l.paths {
		exists, err := fs.Exists(path)
		if err != nil {
			l.log.Warn("Skipping finding plugins as an error occurred", "path", path, "err", err)
			continue
		}
		if !exists {
			l.log.Warn("Skipping finding plugins as directory does not exist", "path", path)
			continue
		}

		paths, err := l.getAbsPluginJSONPaths(path)
		if err != nil {
			return nil, err
		}
		pluginJSONPaths = append(pluginJSONPaths, paths...)
	}

	// load plugin.json files and map directory to JSON data
	foundPlugins := make(map[string]plugins.JSONData)
	for _, pluginJSONPath := range pluginJSONPaths {
		reader, err := l.readFile(pluginJSONPath)
		if err != nil {
			l.log.Warn("Skipping plugin loading as its plugin.json could not be read", "path", pluginJSONPath, "err", err)
			continue
		}
		plugin, err := ReadPluginJSON(reader)
		if err != nil {
			l.log.Warn("Skipping plugin loading as its plugin.json could not be read", "path", pluginJSONPath, "err", err)
			continue
		}
		if err = reader.Close(); err != nil {
			l.log.Warn("Failed to close plugin JSON file", "path", pluginJSONPath, "err", err)
		}

		pluginJSONAbsPath, err := filepath.Abs(pluginJSONPath)
		if err != nil {
			l.log.Warn("Skipping plugin loading as absolute plugin.json path could not be calculated", "pluginID", plugin.ID, "err", err)
			continue
		}

		if _, dupe := foundPlugins[filepath.Dir(pluginJSONAbsPath)]; dupe {
			l.log.Warn("Skipping plugin loading as it's a duplicate", "pluginID", plugin.ID)
			continue
		}
		foundPlugins[filepath.Dir(pluginJSONAbsPath)] = plugin
	}

	var res = make(map[string]*plugins.FoundBundle)
	for pluginDir, data := range foundPlugins {
		files, err := collectFilesWithin(pluginDir)
		if err != nil {
			return nil, err
		}

		res[pluginDir] = &plugins.FoundBundle{
			Primary: plugins.FoundPlugin{
				JSONData: data,
				FS:       plugins.NewLocalFS(files, pluginDir),
			},
		}
	}

	var result []*plugins.FoundBundle
	for dir := range foundPlugins {
		ancestors := strings.Split(dir, string(filepath.Separator))
		ancestors = ancestors[0 : len(ancestors)-1]

		pluginPath := ""
		if runtime.GOOS != "windows" && filepath.IsAbs(dir) {
			pluginPath = "/"
		}
		add := true
		for _, ancestor := range ancestors {
			pluginPath = filepath.Join(pluginPath, ancestor)
			if _, ok := foundPlugins[pluginPath]; ok {
				if fp, exists := res[pluginPath]; exists {
					fp.Children = append(fp.Children, &res[dir].Primary)
					add = false
					break
				}
			}
		}
		if add {
			result = append(result, res[dir])
		}
	}

	return result, nil
}

func (l *Local) getAbsPluginJSONPaths(path string) ([]string, error) {
	var pluginJSONPaths []string

	var err error
	path, err = filepath.Abs(path)
	if err != nil {
		return []string{}, err
	}

	if err = walk(path, true, true,
		func(currentPath string, fi os.FileInfo, err error) error {
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					l.log.Error("Couldn't scan directory since it doesn't exist", "pluginDir", path, "err", err)
					return nil
				}
				if errors.Is(err, os.ErrPermission) {
					l.log.Error("Couldn't scan directory due to lack of permissions", "pluginDir", path, "err", err)
					return nil
				}

				return fmt.Errorf("filepath.Walk reported an error for %q: %w", currentPath, err)
			}

			if fi.Name() == "node_modules" {
				return util.ErrWalkSkipDir
			}

			if fi.IsDir() {
				return nil
			}

			if fi.Name() != "plugin.json" {
				return nil
			}

			pluginJSONPaths = append(pluginJSONPaths, currentPath)
			return nil
		}); err != nil {
		return []string{}, err
	}

	return pluginJSONPaths, nil
}

func collectFilesWithin(dir string) (map[string]struct{}, error) {
	files := map[string]struct{}{}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.Mode()&os.ModeSymlink == os.ModeSymlink {
			symlinkPath, err := filepath.EvalSymlinks(path)
			if err != nil {
				return err
			}

			symlink, err := os.Stat(symlinkPath)
			if err != nil {
				return err
			}

			// verify that symlinked file is within plugin directory
			p, err := filepath.Rel(dir, symlinkPath)
			if err != nil {
				return err
			}
			if p == ".." || strings.HasPrefix(p, ".."+string(filepath.Separator)) {
				return fmt.Errorf("file '%s' not inside of plugin directory", p)
			}

			// skip adding symlinked directories
			if symlink.IsDir() {
				return nil
			}
		}

		// skip directories
		if info.IsDir() {
			return nil
		}

		// verify that file is within plugin directory
		file, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if strings.HasPrefix(file, ".."+string(filepath.Separator)) {
			return fmt.Errorf("file '%s' not inside of plugin directory", file)
		}

		files[path] = struct{}{}

		return nil
	})

	return files, err
}

func (l *Local) readFile(pluginJSONPath string) (io.ReadCloser, error) {
	l.log.Debug("Loading plugin", "path", pluginJSONPath)

	if !strings.EqualFold(filepath.Ext(pluginJSONPath), ".json") {
		return nil, ErrInvalidPluginJSONFilePath
	}

	absPluginJSONPath, err := filepath.Abs(pluginJSONPath)
	if err != nil {
		return nil, err
	}

	// Wrapping in filepath.Clean to properly handle
	// gosec G304 Potential file inclusion via variable rule.
	return os.Open(filepath.Clean(absPluginJSONPath))
}