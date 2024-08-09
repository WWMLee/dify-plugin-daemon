package plugin_manager

import (
	"path"
	"time"

	"github.com/langgenius/dify-plugin-daemon/internal/core/plugin_manager/aws_manager"
	"github.com/langgenius/dify-plugin-daemon/internal/core/plugin_manager/local_manager"
	"github.com/langgenius/dify-plugin-daemon/internal/core/plugin_manager/positive_manager"
	"github.com/langgenius/dify-plugin-daemon/internal/core/plugin_manager/remote_manager"
	"github.com/langgenius/dify-plugin-daemon/internal/storage"
	"github.com/langgenius/dify-plugin-daemon/internal/types/app"
	"github.com/langgenius/dify-plugin-daemon/internal/types/entities"
	"github.com/langgenius/dify-plugin-daemon/internal/types/entities/plugin_entities"
	"github.com/langgenius/dify-plugin-daemon/internal/utils/log"
	"github.com/langgenius/dify-plugin-daemon/internal/utils/routine"
)

func (p *PluginManager) startLocalWatcher(config *app.Config) {
	go func() {
		log.Info("start to handle new plugins in path: %s", config.StoragePath)
		p.handleNewPlugins(config)
		for range time.NewTicker(time.Second * 30).C {
			p.handleNewPlugins(config)
		}
	}()
}

func (p *PluginManager) startRemoteWatcher(config *app.Config) {
	// launch TCP debugging server if enabled
	if config.PluginRemoteInstallingEnabled {
		server := remote_manager.NewRemotePluginServer(config)
		go func() {
			err := server.Launch()
			if err != nil {
				log.Error("start remote plugin server failed: %s", err.Error())
			}
		}()
		go func() {
			server.Wrap(func(rpr *remote_manager.RemotePluginRuntime) {
				p.lifetime(config, rpr)
			})
		}()
	}
}

func (p *PluginManager) handleNewPlugins(config *app.Config) {
	// load local plugins firstly
	for plugin := range p.loadNewPlugins(config.StoragePath) {
		var plugin_interface entities.PluginRuntimeInterface

		if config.Platform == app.PLATFORM_AWS_LAMBDA {
			plugin_interface = &aws_manager.AWSPluginRuntime{
				PluginRuntime: plugin,
				PositivePluginRuntime: positive_manager.PositivePluginRuntime{
					LocalPath: plugin.State.AbsolutePath,
				},
			}
		} else if config.Platform == app.PLATFORM_LOCAL {
			plugin_interface = &local_manager.LocalPluginRuntime{
				PluginRuntime: plugin,
				PositivePluginRuntime: positive_manager.PositivePluginRuntime{
					LocalPath: plugin.State.AbsolutePath,
				},
			}
		} else {
			log.Error("unsupported platform: %s for plugin: %s", config.Platform, plugin.Config.Name)
			continue
		}

		routine.Submit(func() {
			p.lifetime(config, plugin_interface)
		})
	}
}

// chan should be closed after using that
func (p *PluginManager) loadNewPlugins(root_path string) <-chan entities.PluginRuntime {
	ch := make(chan entities.PluginRuntime)

	plugin_paths, err := storage.List(root_path)
	if err != nil {
		log.Error("no plugin found in path: %s", root_path)
		close(ch)
		return ch
	}

	routine.Submit(func() {
		for _, plugin_path := range plugin_paths {
			if plugin_path.IsDir() {
				configuration_path := path.Join(root_path, plugin_path.Name(), "manifest.yaml")
				configuration, err := parsePluginConfig(configuration_path)
				if err != nil {
					log.Error("parse plugin config error: %v", err)
					continue
				}

				status := p.verifyPluginStatus(configuration)
				if status.exist {
					continue
				}

				// check if .verified file exists
				verified_path := path.Join(root_path, plugin_path.Name(), ".verified")
				_, err = storage.Exists(verified_path)

				ch <- entities.PluginRuntime{
					Config: *configuration,
					State: entities.PluginRuntimeState{
						Status:       entities.PLUGIN_RUNTIME_STATUS_PENDING,
						Restarts:     0,
						AbsolutePath: path.Join(root_path, plugin_path.Name()),
						ActiveAt:     nil,
						Verified:     err == nil,
					},
				}
			}
		}

		close(ch)
	})

	return ch
}

func parsePluginConfig(configuration_path string) (*plugin_entities.PluginDeclaration, error) {
	text, err := storage.Read(configuration_path)
	if err != nil {
		return nil, err
	}

	result, err := plugin_entities.UnmarshalPluginDeclarationFromYaml(text)
	if err != nil {
		return nil, err
	}

	return result, nil
}

type pluginStatusResult struct {
	exist bool
}

func (p *PluginManager) verifyPluginStatus(config *plugin_entities.PluginDeclaration) pluginStatusResult {
	_, exist := p.checkPluginExist(config.Identity())
	if exist {
		return pluginStatusResult{
			exist: true,
		}
	}

	return pluginStatusResult{
		exist: false,
	}
}
