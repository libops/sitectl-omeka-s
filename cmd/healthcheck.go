package cmd

import "github.com/libops/sitectl/pkg/plugin"

var omekaSHealthcheckRunner = plugin.StandardComposeWebHealthcheck(plugin.StandardComposeWebHealthcheckOptions{
	AppService:              "omeka-s",
	HTTPName:                "http:omeka-s",
	DefaultScheme:           "http",
	DefaultDomain:           "localhost",
	DatabaseService:         "mariadb",
	CheckDatabaseDependency: true,
})
