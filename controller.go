package main

import (
	"context"
	"fmt"
	serverv1 "github.com/hmsayem/server-controller/pkg/apis/hmsayem.com/v1"
	clientset "github.com/hmsayem/server-controller/pkg/client/clientset/versioned"
	samplescheme "github.com/hmsayem/server-controller/pkg/client/clientset/versioned/scheme"
	informers "github.com/hmsayem/server-controller/pkg/client/informers/externalversions/hmsayem.com/v1"
	listers "github.com/hmsayem/server-controller/pkg/client/listers/hmsayem.com/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	appslisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"time"
)

const controllerAgentName = "server-controller"
const (
	SuccessSynced         = "Synced"
	ErrResourceExists     = "ErrResourceExists"
	MessageResourceExists = "Resource %q already exists and is not managed by Server"
	MessageResourceSynced = "Server synced successfully"
)

type Controller struct {
	kubeclientset      kubernetes.Interface
	serverclientset    clientset.Interface
	statefulSetsLister appslisters.StatefulSetLister
	statefulSetsSynced cache.InformerSynced
	serverLister       listers.ServerLister
	serverSynced       cache.InformerSynced
	workqueue          workqueue.RateLimitingInterface
	recorder           record.EventRecorder
}

func NewController(
	kubeclientset kubernetes.Interface,
	sampleclientset clientset.Interface,
	statefulSetInformer appsinformers.StatefulSetInformer,
	serverInformer informers.ServerInformer) *Controller {

	utilruntime.Must(samplescheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:      kubeclientset,
		serverclientset:    sampleclientset,
		statefulSetsLister: statefulSetInformer.Lister(),
		statefulSetsSynced: statefulSetInformer.Informer().HasSynced,
		serverLister:       serverInformer.Lister(),
		serverSynced:       serverInformer.Informer().HasSynced,
		workqueue:          workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Server"),
		recorder:           recorder,
	}

	klog.Info("Setting up event handlers")
	// Set up an event handler for the changes in Server resources
	serverInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueue,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueue(new)
		},
	})

	statefulSetInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newDepl := new.(*appsv1.StatefulSet)
			oldDepl := old.(*appsv1.StatefulSet)
			if newDepl.ResourceVersion == oldDepl.ResourceVersion {
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})
	return controller
}
func (c *Controller) enqueue(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}
func (c *Controller) handleObject(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		klog.V(4).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	klog.V(4).Infof("Processing object: %s", object.GetName())
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {

		if ownerRef.Kind != "Server" {
			return
		}
		server, err := c.serverLister.Servers(object.GetNamespace()).Get(ownerRef.Name)
		if err != nil {
			klog.V(4).Infof("ignoring orphaned object '%s' of Server '%s'", object.GetSelfLink(), ownerRef.Name)
			return
		}
		c.enqueue(server)
		return
	}
}
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	klog.Info("Starting Foo controller")

	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.statefulSetsSynced, c.serverSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string
		var ok bool

		if key, ok = obj.(string); !ok {
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}

		if err := c.syncHandler(key); err != nil {
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}

		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}
	return true
}
func (c *Controller) syncHandler(key string) error {

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	server, err := c.serverLister.Servers(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("Server '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	statefulSetName := server.Spec.StatefulSetName
	if statefulSetName == "" {
		utilruntime.HandleError(fmt.Errorf("%s: statefulSet name must be specified", key))
		return nil
	}

	statefulSet, err := c.statefulSetsLister.StatefulSets(server.Namespace).Get(statefulSetName)
	if errors.IsNotFound(err) {
		statefulSet, err = c.kubeclientset.AppsV1().StatefulSets(server.Namespace).Create(context.TODO(), newStatefulSet(server), metav1.CreateOptions{})
	}

	if err != nil {
		return err
	}

	if !metav1.IsControlledBy(statefulSet, server) {
		msg := fmt.Sprintf(MessageResourceExists, statefulSet.Name)
		c.recorder.Event(server, corev1.EventTypeWarning, ErrResourceExists, msg)
		return fmt.Errorf(msg)
	}

	if server.Spec.Replicas != nil && *server.Spec.Replicas != *statefulSet.Spec.Replicas {
		klog.V(4).Infof("Foo %s replicas: %d, statefulSet replicas: %d", name, *server.Spec.Replicas, *statefulSet.Spec.Replicas)
		statefulSet, err = c.kubeclientset.AppsV1().StatefulSets(server.Namespace).Update(context.TODO(), newStatefulSet(server), metav1.UpdateOptions{})
	}

	if err != nil {
		return err
	}

	err = c.updateServerStatus(server, statefulSet)
	if err != nil {
		return err
	}

	c.recorder.Event(server, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}
func (c *Controller) updateServerStatus(server *serverv1.Server, statefulSet *appsv1.StatefulSet) error {
	serverCopy := server.DeepCopy()

	serverCopy.Status.AvailableReplicas = statefulSet.Status.CurrentReplicas
	//serverCopy.Status.AvailableReplicas = statefulSet.Status.
	_, err := c.serverclientset.HmsayemV1().Servers(server.Namespace).Update(context.TODO(), serverCopy, metav1.UpdateOptions{})
	return err
}
func newStatefulSet(server *serverv1.Server) *appsv1.StatefulSet {
	labels := map[string]string{
		"app":        "nginx",
		"controller": server.Name,
	}
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      server.Spec.StatefulSetName,
			Namespace: server.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(server, serverv1.SchemeGroupVersion.WithKind("Server")),
			},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: server.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx",
						},
					},
				},
			},
		},
	}
}
