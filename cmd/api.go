package cmd

import (
	"fmt"
	"net/url"
	"strings"

	sitectlplugin "github.com/libops/sitectl/pkg/plugin"
	"github.com/spf13/cobra"
)

type omekaSAPIOptions struct {
	baseURL       string
	keyIdentity   string
	keyCredential string
	query         []string
	data          string
	file          string
}

func registerOmekaSCommands(s *sitectlplugin.SDK) {
	s.AddCommand(omekaSAPICommand(s))
	for _, spec := range []struct {
		use      string
		resource string
		short    string
	}{
		{use: "items [ID]", resource: "items", short: "List or read Omeka S items"},
		{use: "item-sets [ID]", resource: "item_sets", short: "List or read Omeka S item sets"},
		{use: "media [ID]", resource: "media", short: "List or read Omeka S media"},
		{use: "vocabularies [ID]", resource: "vocabularies", short: "List or read Omeka S vocabularies"},
		{use: "resource-classes [ID]", resource: "resource_classes", short: "List or read Omeka S resource classes"},
		{use: "properties [ID]", resource: "properties", short: "List or read Omeka S properties"},
		{use: "resource-templates [ID]", resource: "resource_templates", short: "List or read Omeka S resource templates"},
		{use: "sites [ID]", resource: "sites", short: "List or read Omeka S sites"},
		{use: "site-pages [ID]", resource: "site_pages", short: "List or read Omeka S site pages"},
		{use: "modules [ID]", resource: "modules", short: "List or read Omeka S modules"},
	} {
		s.AddCommand(omekaSResourceCommand(s, spec.use, spec.resource, spec.short))
	}
}

func omekaSAPICommand(s *sitectlplugin.SDK) *cobra.Command {
	root := &cobra.Command{
		Use:   "api",
		Short: "Call the Omeka S REST API",
	}
	root.AddCommand(omekaSAPIGetCommand(s))
	root.AddCommand(omekaSAPIRequestCommand(s))
	return root
}

func omekaSAPIGetCommand(s *sitectlplugin.SDK) *cobra.Command {
	opts := defaultOmekaSAPIOptions()
	cmd := &cobra.Command{
		Use:   "get RESOURCE [ID]",
		Short: "GET an Omeka S API resource",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			if len(args) == 2 {
				path = strings.TrimRight(path, "/") + "/" + strings.TrimLeft(args[1], "/")
			}
			return runOmekaSAPIRequest(s, cmd, "GET", path, opts)
		},
	}
	bindOmekaSAPIReadFlags(cmd, &opts)
	return cmd
}

func omekaSAPIRequestCommand(s *sitectlplugin.SDK) *cobra.Command {
	opts := defaultOmekaSAPIOptions()
	cmd := &cobra.Command{
		Use:   "request METHOD PATH",
		Short: "Call an arbitrary Omeka S API path",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOmekaSAPIRequest(s, cmd, args[0], args[1], opts)
		},
	}
	bindOmekaSAPIWriteFlags(cmd, &opts)
	return cmd
}

func omekaSResourceCommand(s *sitectlplugin.SDK, use, resource, short string) *cobra.Command {
	opts := defaultOmekaSAPIOptions()
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := resource
			if len(args) == 1 {
				path += "/" + strings.TrimLeft(args[0], "/")
			}
			return runOmekaSAPIRequest(s, cmd, "GET", path, opts)
		},
	}
	bindOmekaSAPIReadFlags(cmd, &opts)
	return cmd
}

func defaultOmekaSAPIOptions() omekaSAPIOptions {
	return omekaSAPIOptions{baseURL: "http://localhost/api"}
}

func bindOmekaSAPIReadFlags(cmd *cobra.Command, opts *omekaSAPIOptions) {
	cmd.Flags().StringVar(&opts.baseURL, "url", opts.baseURL, "Base Omeka S API URL.")
	cmd.Flags().StringVar(&opts.keyIdentity, "identity", "", "Omeka S API key identity.")
	cmd.Flags().StringVar(&opts.keyCredential, "credential", "", "Omeka S API key credential.")
	cmd.Flags().StringArrayVarP(&opts.query, "query", "q", nil, "Additional query parameter as name=value. May be repeated.")
}

func bindOmekaSAPIWriteFlags(cmd *cobra.Command, opts *omekaSAPIOptions) {
	bindOmekaSAPIReadFlags(cmd, opts)
	cmd.Flags().StringVar(&opts.data, "data", "", "JSON request body.")
	cmd.Flags().StringVar(&opts.file, "file", "", "Path to a JSON request body file.")
}

func runOmekaSAPIRequest(s *sitectlplugin.SDK, cmd *cobra.Command, method, path string, opts omekaSAPIOptions) error {
	query := append([]string{}, opts.query...)
	if strings.TrimSpace(opts.keyIdentity) != "" {
		query = append(query, "key_identity="+opts.keyIdentity)
	}
	if strings.TrimSpace(opts.keyCredential) != "" {
		query = append(query, "key_credential="+opts.keyCredential)
	}
	requestURL, err := buildAPIURL(opts.baseURL, path, query)
	if err != nil {
		return err
	}
	args := []string{"curl", "-fsS", "-X", strings.ToUpper(method), "-H", "Accept: application/json"}
	if opts.data != "" || opts.file != "" {
		args = append(args, "-H", "Content-Type: application/json")
	}
	if opts.data != "" {
		args = append(args, "--data", opts.data)
	}
	if opts.file != "" {
		args = append(args, "--data-binary", "@"+opts.file)
	}
	args = append(args, requestURL)
	return s.RunActiveComposeProjectCommand(cmd, sitectlplugin.ShellJoin(args))
}

func buildAPIURL(baseURL, path string, queryPairs []string) (string, error) {
	raw := strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(path, "/")
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse API URL: %w", err)
	}
	values := parsed.Query()
	for _, pair := range queryPairs {
		key, value, ok := strings.Cut(pair, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return "", fmt.Errorf("query parameter must be name=value: %q", pair)
		}
		values.Add(key, value)
	}
	parsed.RawQuery = values.Encode()
	return parsed.String(), nil
}
