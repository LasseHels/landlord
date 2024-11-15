package main

import (
	"context"
	"fmt"
	"io"
	defaultlog "log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2024-07-01/virtualmachinescalesetvms"
	"github.com/hashicorp/go-azure-sdk/sdk/auth"
	"github.com/hashicorp/go-azure-sdk/sdk/environments"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/LasseHels/landlord/pkg/errors"
	"github.com/LasseHels/landlord/pkg/landlord"
	"github.com/LasseHels/landlord/pkg/log"
)

func main() {
	os.Exit(start())
}

func start() int {
	// Discard default logs.
	defaultlog.SetOutput(io.Discard)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	config := log.Config{
		Level: "debug",
		Name:  "landlord",
	}
	logger := log.New(config, os.Stdout)

	kubeClient, err := getKubeClient()
	if err != nil {
		fmt.Println(err.Error())
		return 1
	}

	azureClient, err := getAzureClient(ctx)
	if err != nil {
		fmt.Println(err.Error())
		return 1
	}

	l := landlord.New(logger, kubeClient.CoreV1().Nodes(), azureClient, r)

	l.Start(ctx)

	return 0
}

func getKubeClient() (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromKubeconfigGetter(
		"",
		clientcmd.NewDefaultClientConfigLoadingRules().Load,
	)
	if err != nil {
		return nil, errors.Wrap(err, "could not build Kubernetes configuration")
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "error creating Kubernetes client")
	}

	return client, nil
}

func getAzureClient(ctx context.Context) (*virtualmachinescalesetvms.VirtualMachineScaleSetVMsClient, error) {
	// The SDK requires that we set a timeout for authentication.
	ctx, stop := context.WithTimeout(ctx, time.Minute)
	defer stop()
	// https://github.com/hashicorp/go-azure-sdk/blob/main/sdk/auth/README.md#example-authenticating-using-the-azure-cli.
	credentials := auth.Credentials{
		Environment:                       *environments.AzurePublic(),
		EnableAuthenticatingUsingAzureCLI: true,
	}

	api := environments.ResourceManagerAPI("https://management.azure.com")
	authorizer, err := auth.NewAuthorizerFromCredentials(ctx, credentials, api)
	if err != nil {
		return nil, errors.Wrap(err, "building authorizer from credentials")
	}

	// https://pkg.go.dev/github.com/hashicorp/go-azure-sdk/resource-manager/compute/2024-07-01/virtualmachinescalesetvms
	client, err := virtualmachinescalesetvms.NewVirtualMachineScaleSetVMsClientWithBaseURI(api)
	if err != nil {
		return nil, errors.Wrap(err, "error building new Virtual Machine Scale Set client")
	}
	client.Client.Authorizer = authorizer

	return client, nil
}
