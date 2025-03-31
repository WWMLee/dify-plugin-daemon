package plugin

import (
	"os"

	"github.com/langgenius/dify-plugin-daemon/internal/utils/log"
	"github.com/langgenius/dify-plugin-daemon/pkg/plugin_packager/decoder"
	"github.com/langgenius/dify-plugin-daemon/pkg/plugin_packager/packager"
	"github.com/spf13/viper"
)

var (
	MaxPluginPackageSize int64
)

func init() {
	viper.SetDefault("plugin.max_package_size", int64(52428800)) // 先设置默认值
	MaxPluginPackageSize = viper.GetInt64("plugin.max_package_size") // 然后获取值
	log.Info("plugin package max size: %s", MaxPluginPackageSize)
}

func PackagePlugin(inputPath string, outputPath string) {
	decoder, err := decoder.NewFSPluginDecoder(inputPath)
	if err != nil {
		log.Error("failed to create plugin decoder , plugin path: %s, error: %v", inputPath, err)
		os.Exit(1)
		return
	}

	packager := packager.NewPackager(decoder)
	zipFile, err := packager.Pack(MaxPluginPackageSize)

	if err != nil {
		log.Error("failed to package plugin %v", err)
		os.Exit(1)
		return
	}

	err = os.WriteFile(outputPath, zipFile, 0644)
	if err != nil {
		log.Error("failed to write package file %v", err)
		os.Exit(1)
		return
	}

	log.Info("plugin packaged successfully, output path: %s", outputPath)
}
