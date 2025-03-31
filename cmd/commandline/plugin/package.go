package plugin

import (
	"os"

	"github.com/langgenius/dify-plugin-daemon/internal/utils/log"
	"github.com/langgenius/dify-plugin-daemon/pkg/plugin_packager/decoder"
	"github.com/langgenius/dify-plugin-daemon/pkg/plugin_packager/packager"
	"github.com/spf13/viper"
)

var (
	MaxPluginPackageSize = viper.GetInt64("plugin.max_package_size") // 从配置文件中读取，默认值为 50MB
)

func init() {
	viper.SetDefault("plugin.max_package_size", int64(52428800)) // 设置默认值
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
