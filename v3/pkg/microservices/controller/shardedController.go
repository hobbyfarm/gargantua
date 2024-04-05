package microservices

import (
	"context"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type ShardedController struct {
	BaseController
	LoadScheduler
	kubeClient       *kubernetes.Clientset
	statefulset_name string
	replica_count    int
	replica_identity int
}

func NewShardedController(ctx context.Context, informer cache.SharedIndexInformer, kubeclient *kubernetes.Clientset, name string, resyncPeriod time.Duration) *ShardedController {
	dc := &ShardedController{
		BaseController: *newBaseController(name, ctx, informer, resyncPeriod),
		kubeClient:     kubeclient,
	}
	return dc
}

// This method calculates if the object will be reconciled on this shard and adds it to the local queue
// otherwise it will be ignored
func (c *ShardedController) enqueue(obj interface{}) {
	if c.replica_identity > c.replica_count || c.replica_count == 0 {
		// we have likely scaled down. No longer enqueue in this replica.
		return
	}

	// calculate the placement of the object
	shardPlacement, err := c.getShardPlacement(obj)

	if err != nil {
		glog.Errorf("Could not enqueue object due to error in placement calculation: %v", err)
		return
	}

	// if this object is placed on this replica then enqueue it
	if shardPlacement == c.replica_identity {
		c.BaseController.enqueue(obj)
	}
}

// This method calculates on which replica an object needs to be reoconciled.
// It uses a hash of the objects name to guarantee an almost equally distribution between replicas.
func (c *ShardedController) getShardPlacement(obj interface{}) (int, error) {
	hasher := fnv.New32a()
	var key string
	var err error
	// Get the objects cache name
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		glog.V(4).Infof("Error enquing %s: %v", key, err)
		return -1, err
	}

	// calc md5 hash of the key
	_, err = io.WriteString(hasher, key)
	if err != nil {
		panic(err)
	}

	//store the has as bytearray
	hash := hasher.Sum(nil)

	// convert the hash into an integer by truncating it
	truncatedHash := int(binary.BigEndian.Uint32(hash[:4]))

	if truncatedHash < 0 {
		//Ensure only positive values are taken
		truncatedHash = -truncatedHash
	}

	// return the hash modulo the total replica count, this creates an almost equally distributed placement
	return truncatedHash % c.replica_count, nil
}

// RunSharded will start a sharded controller that watches the parent StatefulSet and applies sharding based on the total replica count
func (c *ShardedController) RunSharded(stopCh <-chan struct{}, statefulSetName microservices.MicroService) error {
	c.statefulset_name = string(statefulSetName)
	podIdentityName, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("Error in getting Hostname")
	}

	parts := strings.Split(podIdentityName, "-")
	ordinalIndex, err := strconv.Atoi(parts[len(parts)-1])

	if err != nil {
		return fmt.Errorf("Error in getting a ordinal pod identity from string: %s", podIdentityName)
	}
	c.replica_identity = ordinalIndex

	err = c.watchStatefulSetUpdates(stopCh)
	if err != nil {
		return err
	}

	return c.run(stopCh)
}

// handle updates to the statefulset. set the replica count to the current total replica count
func (c *ShardedController) handleStatefulsetUpdate(obj interface{}) {
	statefulset, ok := obj.(*v1.StatefulSet)
	if !ok {
		glog.V(4).Infof("Not a StatefulSet: %v", obj)
		return
	}

	replicaCount := int(*statefulset.Spec.Replicas)
	if replicaCount != c.replica_count {
		glog.V(8).Infof("Statefulset %s updated replica count from %d to %d replicas", statefulset.Name, c.replica_count, replicaCount)
		c.replica_count = replicaCount
	}
}

// handle updates to the parent StatefulSet of this replica. Set the replica count to the current total replica count
func (c *ShardedController) watchStatefulSetUpdates(stopCh <-chan struct{}) error {
	// client to watch for updates of the parent statefulset object
	watchlist := cache.NewListWatchFromClient(
		c.kubeClient.AppsV1().RESTClient(),
		"statefulsets",
		util.GetReleaseNamespace(),
		fields.OneTermEqualSelector("metadata.name", c.statefulset_name),
	)

	// build an informer to watch updates on the parent statefulset and update the total number of replicas accordingly
	_, controller := cache.NewInformer(
		watchlist,
		&v1.StatefulSet{},
		c.ResyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				c.handleStatefulsetUpdate(obj)
			},
			DeleteFunc: func(obj interface{}) {
				c.handleStatefulsetUpdate(obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				c.handleStatefulsetUpdate(newObj)
			},
		},
	)

	go controller.Run(stopCh)

	glog.V(4).Info("Waiting for StatefulSet Informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, controller.HasSynced); !ok {
		return fmt.Errorf("failed to wait for StatefulSet Informer caches to sync")
	}

	return nil
}
