package jivavolume

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/go-logr/logr"
	jv "github.com/utkarshmani1997/jiva-operator/pkg/apis/openebs/v1alpha1"
	container "github.com/utkarshmani1997/jiva-operator/pkg/kubernetes/container/v1alpha1"
	pts "github.com/utkarshmani1997/jiva-operator/pkg/kubernetes/podtemplatespec/v1alpha1"

	operr "github.com/utkarshmani1997/jiva-operator/pkg/errors/v1alpha1"
	deploy "github.com/utkarshmani1997/jiva-operator/pkg/kubernetes/deployment/appsv1/v1alpha1"
	pvc "github.com/utkarshmani1997/jiva-operator/pkg/kubernetes/pvc/v1alpha1"
	svc "github.com/utkarshmani1997/jiva-operator/pkg/kubernetes/service/v1alpha1"
	sts "github.com/utkarshmani1997/jiva-operator/pkg/kubernetes/statefulset/appsv1/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_jivavolume")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new JivaVolume Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileJivaVolume{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("jivavolume-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource JivaVolume
	err = c.Watch(&source.Kind{Type: &jv.JivaVolume{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner JivaVolume
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &jv.JivaVolume{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileJivaVolume implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileJivaVolume{}

// ReconcileJivaVolume reconciles a JivaVolume object
type ReconcileJivaVolume struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a JivaVolume object and makes changes based on the state read
// and what is in the JivaVolume.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileJivaVolume) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling JivaVolume")

	// Fetch the JivaVolume instance
	instance := &jv.JivaVolume{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	switch instance.Status.Phase {
	case jv.JivaVolumePhaseCreated, jv.JivaVolumePhaseSyncing:
		return reconcile.Result{}, getVolumeStatus(instance.Spec.TargetIP + fmt.Sprint(instance.Spec.TargetPort))
	case jv.JivaVolumePhaseFailed:
		return reconcile.Result{}, teardownJivaComponents(instance)
	case jv.JivaVolumePhasePending:
		reqLogger.Info("JivaVolume is in pending state, start bootstraping jiva components")
		return reconcile.Result{}, r.bootstrapJiva(instance, reqLogger)
	default:
		reqLogger.Info("JivaVolume is in pending state, start bootstraping jiva components")
		return reconcile.Result{}, r.bootstrapJiva(instance, reqLogger)
	}

	return reconcile.Result{}, nil
}

// 1. Create controller svc
// 2. Create controller deploy
// 3. Create replica statefulset
func (r *ReconcileJivaVolume) bootstrapJiva(cr *jv.JivaVolume, reqLog logr.Logger) error {
	if err := r.createControllerService(cr, reqLog); err != nil {
		return err
	}

	if err := r.updateJivaVolume(cr); err != nil {
		return err
	}

	if err := r.createControllerDeployment(cr, reqLog); err != nil {
		return err
	}

	if err := r.createReplicaStatefulSet(cr, reqLog); err != nil {
		return err
	}
	return nil
}

// newPodForCR returns a busybox pod with the same name/namespace as the cr
func newPodForCR(cr *jv.JivaVolume) *corev1.Pod {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-pod",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "busybox",
					Image:   "busybox",
					Command: []string{"sleep", "3600"},
				},
			},
		},
	}
}

// TODO: Add code to configure resource limits, nodeAffinity etc.
func (r *ReconcileJivaVolume) createControllerDeployment(cr *jv.JivaVolume,
	reqLog logr.Logger) error {
	labels := map[string]string{
		"openebs.io/cas-type":                "jiva",
		"openebs.io/controller":              "jiva-controller",
		"openebs.io/persistent-volume":       cr.Name,
		"openebs.io/persistent-volume-claim": cr.Spec.PVC,
	}

	reps := int32(1)
	dep, err := deploy.NewBuilder().WithName(cr.Name + "-jiva-ctrl").
		WithNamespace(cr.Namespace).
		WithLabels(labels).
		WithReplicas(&reps).
		WithStrategyType(appsv1.RecreateDeploymentStrategyType).
		WithSelectorMatchLabelsNew(map[string]string{
			"openebs.io/persistent-volume":       cr.Name,
			"openebs.io/persistent-volume-claim": cr.Spec.PVC,
		}).
		WithPodTemplateSpecBuilder(
			pts.NewBuilder().
				WithLabels(labels).
				WithAnnotations(map[string]string{
					"prometheus.io/path":  "/metrics",
					"prometheus.io/port":  "9500",
					"prometheus.io/scrap": "true",
				}).
				WithContainerBuilders(
					container.NewBuilder().
						WithName("jiva-controller").
						WithImage(getImage("OPENEBS_IO_JIVA_CONTROLLER_IMAGE",
							"jiva-controller")).
						WithPortsNew([]corev1.ContainerPort{
							{
								ContainerPort: 3260,
								Protocol:      "TCP",
							},
							{
								ContainerPort: 9501,
								Protocol:      "TCP",
							},
						}).
						WithCommandNew([]string{
							"launch",
						}).
						WithArgumentsNew([]string{
							"controller",
							"--frontend",
							"gotgt",
							"--clusterIP",
							cr.Name + "-ctrl-svc." + cr.Namespace + ".svc.cluster.local",
							cr.Name,
						}).
						WithEnvsNew([]corev1.EnvVar{
							{
								Name:  "REPLICATION_FACTOR",
								Value: cr.Spec.ReplicationFactor,
							},
						}).
						WithImagePullPolicy(corev1.PullIfNotPresent),
					container.NewBuilder().
						WithImage(getImage("OPENEBS_IO_MAYA_EXPORTER_IMAGE",
							"exporter")).
						WithName("maya-volume-exporter").
						WithCommandNew([]string{"maya-exporter"}).
						WithPortsNew([]corev1.ContainerPort{
							{
								ContainerPort: 9500,
								Protocol:      "TCP",
							},
						},
						),
				),
		).Build()

	if err != nil {
		return operr.Wrapf(err, "failed to build deployment object")
	}

	found := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: dep.Name, Namespace: dep.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLog.Info("Creating a new deployment", "Deploy.Namespace", dep.Namespace, "Deploy.Name", dep.Name)
		err = r.client.Create(context.TODO(), dep)
		if err != nil {
			return operr.Wrapf(err, "failed to create deployment: %v", dep.Name)
		}

		// Statefulset created successfully - don't requeue
		return nil
	} else if err != nil {
		return operr.Wrapf(err, "failed to get the deployment details: %v", dep.Name)
	}

	return nil
}

func getImage(key, component string) string {
	image, present := os.LookupEnv(key)
	if !present {
		switch component {
		case "jiva-controller", "jiva-replica":
			image = "openebs/jiva:ci"
		case "exporter":
			image = "openebs/m-exporter:ci"
		}
	}
	return image
}

func (r *ReconcileJivaVolume) createReplicaStatefulSet(cr *jv.JivaVolume,
	reqLog logr.Logger) error {

	var (
		err          error
		replicaCount int32
		stsObj       *appsv1.StatefulSet
	)
	labels := map[string]string{
		"openebs.io/cas-type":                "jiva",
		"openebs.io/replica":                 "jiva-replica",
		"openebs.io/persistent-volume":       cr.Name,
		"openebs.io/persistent-volume-claim": cr.Spec.PVC,
	}
	rc, err := strconv.ParseInt(cr.Spec.ReplicationFactor, 10, 32)
	if err != nil {
		return operr.Wrapf(err,
			"failed to create replica statefulsets: error while parsing RF: %v",
			cr.Spec.ReplicationFactor)
	}

	replicaCount = int32(rc)
	prev := true

	stsObj, err = sts.NewBuilder().
		WithName(cr.Name + "rep").
		WithLabelsNew(labels).
		WithNamespace(cr.Namespace).
		WithServiceName("jiva-replica-svc").
		WithAnnotationsNew(map[string]string{
			"openebs.io/capacity": strconv.FormatInt(cr.Spec.Capacity, 10),
		}).
		WithStrategyType(appsv1.RollingUpdateStatefulSetStrategyType).
		WithReplicas(&replicaCount).
		WithSelectorMatchLabels(map[string]string{
			"openebs.io/persistent-volume":       cr.Name,
			"openebs.io/persistent-volume-claim": cr.Spec.PVC,
			"openebs.io/replica":                 "jiva-replica",
		}).
		WithPodTemplateSpecBuilder(
			pts.NewBuilder().
				WithLabels(labels).
				WithAnnotations(map[string]string{
					"openebs.io/capacity": strconv.FormatInt(cr.Spec.Capacity, 10),
				}).
				WithAffinity(&corev1.Affinity{
					PodAntiAffinity: &corev1.PodAntiAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
							{
								LabelSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										"openebs.io/replica":           "jiva-replica",
										"openebs.io/persistent-volume": cr.Name,
									},
								},
								TopologyKey: "kubernetes.io/hostname",
							},
						},
					},
				}).
				WithContainerBuilders(
					container.NewBuilder().
						WithName("jiva-replica").
						WithImage(getImage("OPENEBS_IO_JIVA_REPLICA_IMAGE",
							"jiva-replica")).
						WithPortsNew([]corev1.ContainerPort{
							{
								ContainerPort: 9502,
								Protocol:      "TCP",
							},
							{
								ContainerPort: 9503,
								Protocol:      "TCP",
							},
							{
								ContainerPort: 9504,
								Protocol:      "TCP",
							},
						}).
						WithCommandNew([]string{
							"launch",
						}).
						WithArgumentsNew([]string{
							"replica",
							"--frontendIP",
							cr.Name + "-ctrl-svc." + cr.Namespace + ".svc.cluster.local",
							"--size",
							strconv.FormatInt(cr.Spec.Capacity, 10),
							"openebs",
						}).
						WithImagePullPolicy(corev1.PullIfNotPresent).
						WithPrivilegedSecurityContext(&prev).
						WithVolumeMountsNew([]corev1.VolumeMount{
							{
								Name:      "openebs",
								MountPath: "/openebs",
							},
						}),
				),
		).
		WithPVC(
			pvc.NewBuilder().
				WithName("openebs").
				WithNamespace(cr.Namespace).
				WithStorageClass("openebs-hostpath").
				WithAccessModes([]corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}).
				WithCapacity(strconv.FormatInt(cr.Spec.Capacity, 10)),
		).Build()

	if err != nil {
		return operr.Wrapf(err, "failed to build statefulset object")
	}

	found := &appsv1.StatefulSet{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: stsObj.Name, Namespace: stsObj.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLog.Info("Creating a new Statefulset", "Statefulset.Namespace", stsObj.Namespace, "Sts.Name", stsObj.Name)
		err = r.client.Create(context.TODO(), stsObj)
		if err != nil {
			return operr.Wrapf(err, "failed to create statefulset: %v", stsObj.Name)
		}

		// Statefulset created successfully - don't requeue
		return nil
	} else if err != nil {
		return operr.Wrapf(err, "failed to get the statefulset details: %v", stsObj.Name)
	}

	return nil
}

func (r *ReconcileJivaVolume) updateJivaVolume(cr *jv.JivaVolume) error {
	ctrlSVC := &v1.Service{}
	if err := r.client.Get(context.TODO(),
		types.NamespacedName{
			Name:      cr.Name + "-ctrl-svc",
			Namespace: cr.Namespace,
		}, ctrlSVC); err != nil {
		return operr.Wrapf(err, "failed to get service: {%v}", cr.Name+"-ctrl-svc")
	}
	cr.Spec.TargetIP = ctrlSVC.Spec.ClusterIP
	cr.Status.Phase = jv.JivaVolumePhaseCreated
	if err := r.client.Update(context.TODO(), cr); err != nil {
		return operr.Wrapf(err, "failed to update JivaVolume CR: {%v} with targetIP", cr)
	}
	return nil
}

func (r *ReconcileJivaVolume) createControllerService(cr *jv.JivaVolume,
	reqLog logr.Logger) error {
	labels := map[string]string{
		"openebs.io/cas-type":                "jiva",
		"openebs.io/controller":              "jiva-controller",
		"openebs.io/controller-service":      "jiva-controller-service",
		"openebs.io/persistent-volume":       cr.Name,
		"openebs.io/persistent-volume-claim": cr.Spec.PVC,
	}

	svcObj, err := svc.NewBuilder().
		WithName(cr.Name + "-ctrl-svc").
		WithLabelsNew(labels).
		WithNamespace(cr.Namespace).
		WithSelectorsNew(map[string]string{
			"openebs.io/controller":        "jiva-controller",
			"openebs.io/persistent-volume": cr.Name,
		}).
		WithPorts([]corev1.ServicePort{
			{
				Name:       "iscsi",
				Port:       3260,
				Protocol:   "TCP",
				TargetPort: intstr.IntOrString{IntVal: 3260},
			},
			{
				Name:       "api",
				Port:       9501,
				Protocol:   "TCP",
				TargetPort: intstr.IntOrString{IntVal: 9501},
			},
			{
				Name:       "m-exporter",
				Port:       9500,
				Protocol:   "TCP",
				TargetPort: intstr.IntOrString{IntVal: 9500},
			},
		}).
		Build()

	if err != nil {
		return operr.Wrapf(err, "failed to build service object")
	}

	found := &v1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: svcObj.Name, Namespace: svcObj.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLog.Info("Creating a new service", "Service.Namespace", svcObj.Namespace, "Service.Name", svcObj.Name)
		err = r.client.Create(context.TODO(), svcObj)
		if err != nil {
			return operr.Wrapf(err, "failed to create svc: %v", svcObj.Name)
		}
		return nil
	} else if err != nil {
		return operr.Wrapf(err, "failed to get the service details: %v", svcObj.Name)
	}

	return nil
}

func teardownJivaComponents(cr *jv.JivaVolume) error {
	return nil
}

func getVolumeStatus(addr string) error {
	return nil
}