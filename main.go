package main

import (
	"fmt"

	"github.com/libops/sitectl-omeka-s/cmd"
	"github.com/libops/sitectl/pkg/plugin"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	sdk := plugin.NewSDK(plugin.Metadata{
		Name:         "omeka-s",
		Version:      fmt.Sprintf("%s (Built on %s from Git SHA %s)", version, date, commit),
		Description:  "Omeka S helpers",
		Author:       "libops",
		TemplateRepo: "https://github.com/libops/omeka-s",
		Includes:     cmd.IncludedPlugins(),
	})

	cmd.RegisterCommands(sdk)
	sdk.Execute()
}
