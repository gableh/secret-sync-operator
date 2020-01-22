package secret

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strconv"
	"time"
)

var log = logf.Log.WithName("controller_secret")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Secret Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileSecret{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("secret-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Secret
	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileSecret implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileSecret{}

// ReconcileSecret reconciles a Secret object
type ReconcileSecret struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Secret object and makes changes based on the state read
// and what is in the Secret.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileSecret) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Secret")

	// Fetch the Secret instance
	instance := &corev1.Secret{}
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

	// Check labels of secret
	if rep, ok := instance.Labels["sso.gable.dev/secret"]; ok {
		reqLogger.Info(fmt.Sprintf("Found secret with label [%s][%s]...[%s]", instance.Name, instance.Namespace, rep))
		result, err := json.Marshal(instance.Data)
		if err != nil {
			reqLogger.Error(err, "Failed to hash secret data")
		}

		hashedData := hash(string(result))
		if currentHash, ok := instance.Annotations["hash"]; ok {
			if hashedData != currentHash {
				_, createErr := controllerutil.CreateOrUpdate(context.TODO(), r.client, instance, func() error {
					instance.Annotations["hash"] = hashedData
					return nil
				})

				if createErr != nil {
					reqLogger.Error(err, "Failed to update secret")
				}

				deploymentList := &appsv1.DeploymentList{}
				listOpts := []client.ListOption{
					client.InNamespace(instance.Namespace),
					client.MatchingLabels{
						"sso.gable.dev/secret": instance.Name,
					},
				}
				if err = r.client.List(context.TODO(), deploymentList, listOpts...); err != nil {
					reqLogger.Error(err, "Failed to list deployments")
					return reconcile.Result{}, err
				}
				deploymentNames := getDeploymentNames(deploymentList.Items)
				log.Info("Printing Found", "Deployments", deploymentNames)
				log.Info("Redeploying")

				for _, deploymentName := range deploymentNames {
					deployment := &appsv1.Deployment{}
					err := r.client.Get(context.TODO(), client.ObjectKey{Namespace: request.Namespace, Name: deploymentName}, deployment)
					if err != nil {
						log.Error(err, "Failed to redeploy")
					} else {
						op, err := controllerutil.CreateOrUpdate(context.TODO(), r.client, deployment, func() error {
							// update the Secret
							deployment.Spec.Template.Labels["updatedSecretAt"] = strconv.FormatInt((time.Now().UnixNano()), 10)
							deployment.Labels["updatedSecretAt"] = strconv.FormatInt((time.Now().UnixNano()), 10)
							return nil
						})
						if err != nil {
							log.Error(err, "Deployment reconcile failed")
						} else {
							log.Info("Deployment successfully reconciled", "operation", op)
						}
					}
				}
			}
		}

	}
	return reconcile.Result{}, nil
}

func updateDeployments() {

}

func hash(s string) string {
	h := fnv.New32a()
	h.Write([]byte(s))
	return strconv.FormatUint(uint64(h.Sum32()), 10)
}

// getDeploymentNames returns the names of the array of deployments passed in
func getDeploymentNames(deployments []appsv1.Deployment) []string {
	var deploymentNames []string
	for _, deployment := range deployments {
		deploymentNames = append(deploymentNames, deployment.Name)
	}
	return deploymentNames
}
