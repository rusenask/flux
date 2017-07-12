package main

import (
	"github.com/spf13/cobra"
	"github.com/weaveworks/flux"
	"github.com/weaveworks/flux/policy"
	"github.com/weaveworks/flux/update"
)

type servicePolicyOpts struct {
	*rootOpts
	outputOpts

	service    string
	containers []string
	tags       []string

	automate, deautomate bool
	lock, unlock         bool

	cause update.Cause
}

func newServicePolicy(parent *rootOpts) *servicePolicyOpts {
	return &servicePolicyOpts{rootOpts: parent}
}

func (opts *servicePolicyOpts) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policy",
		Short: "Manage policies for a service.",
		Example: makeExample(
			"fluxctl policy --service=foo --automate",
			"fluxctl policy --service=foo --lock",
			"fluxctl policy --service=foo --container=bar --tag=1.2.*",
		),
		RunE: opts.RunE,
	}

	AddOutputFlags(cmd, &opts.outputOpts)
	AddCauseFlags(cmd, &opts.cause)
	flags := cmd.Flags()
	flags.StringVarP(&opts.service, "service", "s", "", "Service to modify")
	flags.StringSliceVar(&opts.containers, "container", nil, "Containers to set tag filter")
	flags.StringSliceVar(&opts.tags, "tag", nil, "Tag filter patterns")
	flags.BoolVar(&opts.automate, "automate", false, "Automate service")
	flags.BoolVar(&opts.deautomate, "deautomate", false, "Deautomate for service")
	flags.BoolVar(&opts.lock, "lock", false, "Lock service")
	flags.BoolVar(&opts.unlock, "unlock", false, "Unlock service")

	return cmd
}

func (opts *servicePolicyOpts) RunE(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return errorWantedNoArgs
	}
	if opts.service == "" {
		return newUsageError("-s, --service is required")
	}
	if opts.automate && opts.deautomate {
		return newUsageError("automate and deautomate both specified")
	}
	if opts.lock && opts.unlock {
		return newUsageError("lock and unlock both specified")
	}
	if len(opts.containers) != len(opts.tags) {
		return newUsageError("mismatched container name and tag filter count")
	}

	serviceID, err := flux.ParseServiceID(opts.service)
	if err != nil {
		return err
	}

	update := calculatePolicyChanges(opts)
	jobID, err := opts.API.UpdatePolicies(noInstanceID, policy.Updates{
		serviceID: update,
	}, opts.cause)
	if err != nil {
		return err
	}
	return await(cmd.OutOrStdout(), cmd.OutOrStderr(), opts.API, jobID, false, opts.verbose)
}

func calculatePolicyChanges(opts *servicePolicyOpts) policy.Update {
	add := policy.Set{}
	if opts.automate {
		add = add.Add(policy.Automated)
	}
	if opts.lock {
		add = add.Add(policy.Locked)
	}

	remove := policy.Set{}
	if opts.deautomate {
		remove = remove.Add(policy.Automated)
	}
	if opts.unlock {
		remove = remove.Add(policy.Locked)
	}

	for i, container := range opts.containers {
		tag := opts.tags[i]
		if tag != "" && tag != "*" {
			add = add.Set(policy.TagPrefix(container), "glob:"+tag)
		} else if tag == "*" {
			remove = remove.Add(policy.TagPrefix(container))
		}
	}

	update := policy.Update{}
	if len(add) > 0 {
		update.Add = add
	}
	if len(remove) > 0 {
		update.Remove = remove
	}

	return update
}
