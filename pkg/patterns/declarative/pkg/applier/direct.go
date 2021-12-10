package applier

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/cmd/apply"
	cmdDelete "k8s.io/kubectl/pkg/cmd/delete"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type DirectApplier struct {
}

var _ Applier = &DirectApplier{}

func NewDirectApplier() *DirectApplier {
	return &DirectApplier{}
}

func (d *DirectApplier) ApplyOld(ctx context.Context, opt ApplierOptions) error {
	ioStreams := genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}
	ioReader := strings.NewReader(opt.Manifest)

	restClientGetter := &staticRESTClientGetter{
		RESTMapper: opt.RESTMapper,
		RESTConfig: opt.RESTConfig,
	}
	b := resource.NewBuilder(restClientGetter)

	kubeConfigFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	if opt.Namespace != "" {
		kubeConfigFlags.Namespace = &opt.Namespace
	}
	// Validation potentially causes redundant work, but validation isn't the common path
	v, err := cmdutil.NewFactory(kubeConfigFlags).Validator(opt.Validate)
	if err != nil {
		return err
	}
	b.Schema(v)

	res := b.Unstructured().Stream(ioReader, "manifestString").Do()
	infos, err := res.Infos()
	if err != nil {
		return err
	}

	// Populate the namespace on any namespace-scoped objects
	if opt.Namespace != "" {
		visitor := resource.SetNamespace(opt.Namespace)
		for _, info := range infos {
			if err := info.Visit(visitor); err != nil {
				return fmt.Errorf("error from SetNamespace: %w", err)
			}
		}
	}

	applyOpts := apply.NewApplyOptions(ioStreams)
	applyOpts.Namespace = opt.Namespace
	applyOpts.SetObjects(infos)
	applyOpts.ToPrinter = func(operation string) (printers.ResourcePrinter, error) {
		applyOpts.PrintFlags.NamePrintFlags.Operation = operation
		cmdutil.PrintFlagsWithDryRunStrategy(applyOpts.PrintFlags, applyOpts.DryRunStrategy)
		return applyOpts.PrintFlags.ToPrinter()
	}
	applyOpts.DeleteOptions = &cmdDelete.DeleteOptions{
		IOStreams: ioStreams,
	}

	return applyOpts.Run()
}

// staticRESTClientGetter returns a fixed RESTClient
type staticRESTClientGetter struct {
	RESTConfig      *rest.Config
	DiscoveryClient discovery.CachedDiscoveryInterface
	RESTMapper      meta.RESTMapper
}

var _ resource.RESTClientGetter = &staticRESTClientGetter{}

func (s *staticRESTClientGetter) ToRESTConfig() (*rest.Config, error) {
	if s.RESTConfig == nil {
		return nil, fmt.Errorf("RESTConfig not set")
	}
	return s.RESTConfig, nil
}
func (s *staticRESTClientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	if s.DiscoveryClient == nil {
		return nil, fmt.Errorf("DiscoveryClient not set")
	}
	return s.DiscoveryClient, nil
}
func (s *staticRESTClientGetter) ToRESTMapper() (meta.RESTMapper, error) {
	if s.RESTMapper == nil {
		return nil, fmt.Errorf("RESTMapper not set")
	}
	return s.RESTMapper, nil
}

func (d *DirectApplier) Apply(ctx context.Context, opt ApplierOptions) error {
	stdio := bytes.NewBuffer(nil)
	errio := bytes.NewBuffer(nil)
	streams := genericclioptions.IOStreams{In: stdio, Out: stdio, ErrOut: errio}

	kubeConfigFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	if opt.Namespace != "" {
		kubeConfigFlags.Namespace = &opt.Namespace
	}
	// Validation potentially causes redundant work, but validation isn't the common path
	matchVersionKubeConfigFlags := cmdutil.NewMatchVersionFlags(kubeConfigFlags)
	factory := cmdutil.NewFactory(matchVersionKubeConfigFlags)

	cmd := apply.NewCmdApply("kubectl", factory, streams)
	err := cmd.Flags().Set("validate", "false")
	if err != nil {
		return err
	}
	opts := apply.NewApplyOptions(streams)

	ioReader := strings.NewReader(opt.Manifest)

	if err := opts.Complete(factory, cmd); err != nil {
		return fmt.Errorf("failed to complete apply options: %+v", err)
	}

	v := factory.NewBuilder().
		Unstructured().
		Schema(opts.Validator).
		ContinueOnError().
		NamespaceParam(opts.Namespace).DefaultNamespace().
		Flatten().
		Stream(ioReader, "manifestString").
		Do()

	_ = v.Visit(func(info *resource.Info, err error) error {
		if err != nil {
			return err
		}
		return nil
	})

	return opts.Run()
}
