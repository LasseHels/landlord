package landlord

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2024-07-01/virtualmachinescalesetvms"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/LasseHels/landlord/pkg/errors"
	"github.com/LasseHels/landlord/pkg/log"
)

type lister interface {
	List(ctx context.Context, opts metav1.ListOptions) (*v1.NodeList, error)
}

type evicter interface {
	SimulateEviction(ctx context.Context, id virtualmachinescalesetvms.VirtualMachineScaleSetVirtualMachineId) (result virtualmachinescalesetvms.SimulateEvictionOperationResponse, err error)
}

type rand interface {
	Intn(n int) int
	Shuffle(n int, swap func(i, j int))
}

type Landlord struct {
	logger       log.Logger
	lister       lister
	evicter      evicter
	rand         rand
	minEvictions int
	maxEvictions int
	interval     time.Duration
}

func New(logger log.Logger, lister lister, evicter evicter, rand rand) *Landlord {
	return &Landlord{
		logger:       logger,
		lister:       lister,
		evicter:      evicter,
		rand:         rand,
		minEvictions: 5,
		maxEvictions: 20,
		interval:     time.Second * 10,
	}
}

func (l *Landlord) Start(ctx context.Context) {
	l.logger.Info("Starting landlord with interval %s", l.interval.String())
	tick(ctx, l.tick, l.interval)
}

func (l *Landlord) tick(ctx context.Context) {
	if err := l.sweep(ctx); err != nil {
		l.logger.Error(err.Error())
	}
}

// sweep nodes in the cluster for eviction targets.
func (l *Landlord) sweep(ctx context.Context) error {
	l.logger.Info("Sweeping nodes with min evictions %d and max evictions %d", l.minEvictions, l.maxEvictions)
	evictionCount := l.count()
	l.logger.Info("Generated an eviction count of %d", evictionCount)

	result, err := l.lister.List(ctx, metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "error listing nodes")
	}
	nodes := result.Items
	nodeCount := len(nodes)
	l.logger.Info("Found %d nodes", nodeCount)
	filtered := l.filterNodes(nodes)
	filteredCount := len(filtered)
	l.logger.Info("Filtered %d nodes to %d", nodeCount, filteredCount)

	if evictionCount > filteredCount {
		l.logger.Info(
			"Eviction count (%d) is greater than the amount of filtered nodes (%d), setting eviction count to %d",
			evictionCount,
			filteredCount,
			filteredCount,
		)
		evictionCount = filteredCount
	}

	// Shuffle slice of nodes so the same nodes are not always first.
	l.rand.Shuffle(filteredCount, func(i, j int) {
		filtered[i], filtered[j] = filtered[j], filtered[i]
	})

	for i := 0; i < evictionCount; i++ {
		index := i

		go func() {
			node := filtered[index]

			if err := l.evict(ctx, node); err != nil {
				err = errors.Wrap(err, "error evicting %s", node.Name)
				l.logger.Error(err.Error())
			}
		}()
	}

	return nil
}

// evict a node.
func (l *Landlord) evict(ctx context.Context, node v1.Node) error {
	seconds := l.rand.Intn(int(l.interval.Seconds()))
	delay := time.Duration(seconds) * time.Second
	l.logger.Info("Sleeping %s before evicting %s", delay.String(), node.Name)
	// Sleep a bit before starting the eviction so that not all evictions of the sweep have the same timestamp.
	time.Sleep(delay)

	ctx, cancel := context.WithTimeout(ctx, l.interval)
	defer cancel()

	providerID := node.Spec.ProviderID

	if providerID == "" {
		l.logger.Info("Node %s has no provider ID (perhaps it was already evicted?), returning", node.Name)
		return nil
	}

	// Remove azure:// prefix as the SDK expects a clean resource ID.
	//
	// Before: azure:///subscriptions/3baee020-e0a1-4297-964d-f901c9f12c87/...
	// After: /subscriptions/3baee020-e0a1-4297-964d-f901c9f12c87/...
	resourceID := strings.ReplaceAll(providerID, "azure://", "")
	l.logger.Debug("Parsing resource ID %s", resourceID)
	id, err := virtualmachinescalesetvms.ParseVirtualMachineScaleSetVirtualMachineID(resourceID)
	if err != nil {
		return errors.Wrap(err, "error parsing VM ID %s", resourceID)
	}

	l.logger.Info("Evicting %s", node.Name)
	result, err := l.evicter.SimulateEviction(ctx, *id)
	if err != nil {
		return errors.Wrap(err, "error evicting %s", node.Name)
	}

	body, err := l.readBody(result.HttpResponse.Body)
	if err != nil {
		return err
	}

	l.logger.Info(
		"Eviction of %s got response code %d with body %s",
		node.Name,
		result.HttpResponse.StatusCode,
		body,
	)

	return nil
}

func (l *Landlord) readBody(body io.ReadCloser) (string, error) {
	bytes, err := io.ReadAll(body)
	if err != nil {
		return "", errors.Wrap(err, "could not read body")
	}

	return string(bytes), nil
}

// count of nodes to evict.
func (l *Landlord) count() int {
	return l.rand.Intn(l.maxEvictions-l.minEvictions) + l.minEvictions
}

// filterNodes based on whether they already have been scheduled for eviction.
func (l *Landlord) filterNodes(nodes []v1.Node) []v1.Node {
	var filtered []v1.Node

	for _, node := range nodes {
		// syspool nodes cannot be evicted.
		if strings.Contains(node.Name, "syspool") {
			continue
		}

		if nodeHasScheduledEvent(node) {
			continue
		}

		filtered = append(filtered, node)
	}

	return filtered
}

func nodeHasScheduledEvent(node v1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == "VMEventScheduled" && condition.Status == v1.ConditionTrue {
			return true
		}
	}

	return false
}
