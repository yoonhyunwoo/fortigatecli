package cli

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"fortigatecli/internal/fortigate"
	"fortigatecli/internal/output"

	"github.com/spf13/cobra"
)

type cmdbAlias struct {
	use       string
	short     string
	resource  string
	mkeyLabel string
	children  []cmdbAlias
}

var cmdbReadAliases = []cmdbAlias{
	{use: "address", short: "Read firewall addresses", resource: "firewall/address", mkeyLabel: "name"},
	{use: "policy", short: "Read firewall policies", resource: "firewall/policy", mkeyLabel: "policy-id"},
	{use: "addrgrp", short: "Read firewall address groups", resource: "firewall/addrgrp", mkeyLabel: "name"},
	{
		use:   "service",
		short: "Read service resources",
		children: []cmdbAlias{
			{use: "custom", short: "Read custom services", resource: "firewall.service/custom", mkeyLabel: "name"},
		},
	},
}

func newCMDBCommand(rootOpts *rootOptions) *cobra.Command {
	cmdbCmd := &cobra.Command{
		Use:   "cmdb",
		Short: "Read CMDB resources as lists or single objects",
		Long:  "Read FortiGate CMDB resources with collection, object, alias, and paging-aware output support.",
		Example: "  fortigatecli cmdb list firewall/address --page-size 50 --page 2\n" +
			"  fortigatecli cmdb show firewall/address branch-office\n" +
			"  fortigatecli cmdb address get branch-office\n" +
			"  fortigatecli raw get /api/v2/cmdb/firewall/address/branch-office",
	}

	cmdbCmd.AddCommand(
		newCMDBGetCommand(rootOpts),
		newCMDBListCommand(rootOpts),
		newCMDBShowCommand(rootOpts),
	)
	for _, alias := range cmdbReadAliases {
		cmdbCmd.AddCommand(newCMDBAliasCommand(rootOpts, alias))
	}
	return cmdbCmd
}

func newCMDBGetCommand(rootOpts *rootOptions) *cobra.Command {
	readOpts := newReadOptions()
	shapeOpts := newShapeOptions()
	cmd := &cobra.Command{
		Use:   "get <path>",
		Short: "Get a CMDB resource by raw path",
		Args:  cobra.ExactArgs(1),
		Example: "  fortigatecli cmdb get firewall/address\n" +
			"  fortigatecli cmdb get firewall/address --count 10 --start 20",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCMDBResource(rootOpts, cmd, args[0], readOpts, shapeOpts, false)
		},
	}
	bindReadFlags(cmd, readOpts)
	bindShapeFlags(cmd, shapeOpts)
	setDefaultStreams(cmd)
	return cmd
}

func newCMDBListCommand(rootOpts *rootOptions) *cobra.Command {
	readOpts := newReadOptions()
	shapeOpts := newShapeOptions()
	cmd := &cobra.Command{
		Use:   "list <path>",
		Short: "List CMDB resources",
		Args:  cobra.ExactArgs(1),
		Example: "  fortigatecli cmdb list firewall/policy --page-size 100 --page 1\n" +
			"  fortigatecli cmdb list firewall/address --all --with-meta",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCMDBResource(rootOpts, cmd, args[0], readOpts, shapeOpts, true)
		},
	}
	bindReadFlags(cmd, readOpts)
	bindShapeFlags(cmd, shapeOpts)
	setDefaultStreams(cmd)
	return cmd
}

func newCMDBShowCommand(rootOpts *rootOptions) *cobra.Command {
	readOpts := newReadOptions()
	cmd := &cobra.Command{
		Use:   "show <resource> <mkey>",
		Short: "Read a single CMDB object by resource and mkey",
		Args:  cobra.ExactArgs(2),
		Example: "  fortigatecli cmdb show firewall/address branch-office\n" +
			"  fortigatecli cmdb show firewall.service/custom HTTPS",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCMDBObject(rootOpts, cmd, args[0], args[1], readOpts)
		},
	}
	bindReadFlags(cmd, readOpts)
	setDefaultStreams(cmd)
	return cmd
}

func newCMDBAliasCommand(rootOpts *rootOptions, alias cmdbAlias) *cobra.Command {
	cmd := &cobra.Command{
		Use:   alias.use,
		Short: alias.short,
	}
	if alias.resource != "" {
		cmd.AddCommand(newCMDBAliasListCommand(rootOpts, alias), newCMDBAliasGetCommand(rootOpts, alias))
	}
	for _, child := range alias.children {
		cmd.AddCommand(newCMDBAliasCommand(rootOpts, child))
	}
	setDefaultStreams(cmd)
	return cmd
}

func newCMDBAliasListCommand(rootOpts *rootOptions, alias cmdbAlias) *cobra.Command {
	readOpts := newReadOptions()
	shapeOpts := newShapeOptions()
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List " + alias.short,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCMDBResource(rootOpts, cmd, alias.resource, readOpts, shapeOpts, true)
		},
	}
	bindReadFlags(cmd, readOpts)
	bindShapeFlags(cmd, shapeOpts)
	setDefaultStreams(cmd)
	return cmd
}

func newCMDBAliasGetCommand(rootOpts *rootOptions, alias cmdbAlias) *cobra.Command {
	readOpts := newReadOptions()
	cmd := &cobra.Command{
		Use:   "get <" + alias.mkeyLabel + ">",
		Short: "Read a single object from " + alias.short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCMDBObject(rootOpts, cmd, alias.resource, args[0], readOpts)
		},
	}
	bindReadFlags(cmd, readOpts)
	setDefaultStreams(cmd)
	return cmd
}

func runCMDBResource(rootOpts *rootOptions, cmd *cobra.Command, resourcePath string, readOpts *readOptions, shapeOpts *shapeOptions, listMode bool) error {
	apiOptions := readOpts.toAPIOptions()
	cfg, err := loadRuntimeConfig(rootOpts.vdom)
	if err != nil {
		return err
	}
	client, err := newClient(cfg)
	if err != nil {
		return output.NewError("client_error", err.Error(), nil)
	}
	ctx, cancel := commandContext()
	defer cancel()

	if readOpts.allVDOMs {
		envelope, err := client.GetCMDBAcrossVDOMs(ctx, resourcePath, apiOptions)
		if err != nil {
			return err
		}
		return render(cmd, rootOpts.output, envelope)
	}

	var envelope *fortigate.Envelope
	if listMode && readOpts.all {
		envelope, err = getAllCMDBPages(ctx, client, resourcePath, apiOptions)
	} else {
		envelope, err = client.GetCMDBResource(ctx, resourcePath, apiOptions)
	}
	if err != nil {
		return err
	}
	return renderRead(cmd, rootOpts.output, envelope, shapeOpts)
}

func runCMDBObject(rootOpts *rootOptions, cmd *cobra.Command, resourcePath string, mkey string, readOpts *readOptions) error {
	apiOptions := readOpts.toAPIOptions()
	cfg, err := loadRuntimeConfig(rootOpts.vdom)
	if err != nil {
		return err
	}
	client, err := newClient(cfg)
	if err != nil {
		return output.NewError("client_error", err.Error(), nil)
	}
	ctx, cancel := commandContext()
	defer cancel()
	envelope, err := client.GetCMDBObject(ctx, resourcePath, mkey, apiOptions)
	if err != nil {
		return err
	}
	return render(cmd, rootOpts.output, envelope)
}

func getAllCMDBPages(ctx context.Context, client *fortigate.Client, resourcePath string, apiOptions fortigate.ReadOptions) (*fortigate.Envelope, error) {
	envelope, err := client.GetCMDBResource(ctx, resourcePath, apiOptions)
	if err != nil {
		return nil, err
	}
	collected, ok := envelope.Results.([]any)
	if !ok {
		return envelope, nil
	}
	firstMeta := envelope.Paging()
	for next := firstMeta.Next; next != ""; {
		nextOptions, nextPath, ok := nextCMDBPage(resourcePath, apiOptions, next)
		if !ok {
			break
		}
		nextEnvelope, err := client.GetCMDBResource(ctx, nextPath, nextOptions)
		if err != nil {
			return nil, err
		}
		nextResults, ok := nextEnvelope.Results.([]any)
		if !ok {
			break
		}
		collected = append(collected, nextResults...)
		next = nextEnvelope.Paging().Next
	}
	envelope.Results = collected
	envelope.Count = len(collected)
	envelope.Next = ""
	envelope.Meta = &fortigate.EnvelopeMeta{Count: len(collected)}
	if len(collected) > 0 {
		envelope.Meta.Range = &fortigate.PageRange{Start: 0, End: len(collected) - 1}
	}
	return envelope, nil
}

func nextCMDBPage(resourcePath string, apiOptions fortigate.ReadOptions, next string) (fortigate.ReadOptions, string, bool) {
	parsed, err := url.Parse(next)
	if err != nil {
		return fortigate.ReadOptions{}, "", false
	}
	nextOptions := apiOptions
	if start := parsed.Query().Get("start"); start != "" {
		if value, err := strconv.Atoi(start); err == nil {
			nextOptions.Start = value
			nextOptions.Page.Start = value
		}
	}
	if count := parsed.Query().Get("count"); count != "" {
		if value, err := strconv.Atoi(count); err == nil {
			nextOptions.Count = value
			nextOptions.Page.Count = value
		}
	}
	nextPath := resourcePath
	if strings.TrimPrefix(parsed.Path, "/") != "" {
		nextPath = strings.TrimPrefix(strings.TrimPrefix(parsed.Path, "/api/v2/cmdb/"), "/")
	}
	return nextOptions, nextPath, true
}
