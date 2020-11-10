package publishingstrategy

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/openshift/cloud-ingress-operator/pkg/apis/cloudingress/v1alpha1"
	cloudingressv1alpha1 "github.com/openshift/cloud-ingress-operator/pkg/apis/cloudingress/v1alpha1"
	"github.com/openshift/cloud-ingress-operator/pkg/cloudclient"
	baseutils "github.com/openshift/cloud-ingress-operator/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	defaultIngressName         = "default"
	ingressControllerNamespace = "openshift-ingress-operator"
)

var log = logf.Log.WithName("controller_publishingstrategy")
var serializer = json.NewSerializerWithOptions(nil, nil, nil, json.SerializerOptions{})

type patchField string

var IngressControllerSelector patchField = "IngressControllerSelector"
var IngressControllerCertificate patchField = "IngressControllerCertificate"

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new PublishingStrategy Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcilePublishingStrategy{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("publishingstrategy-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource PublishingStrategy
	err = c.Watch(&source.Kind{Type: &cloudingressv1alpha1.PublishingStrategy{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcilePublishingStrategy implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcilePublishingStrategy{}

// ReconcilePublishingStrategy reconciles a PublishingStrategy object
type ReconcilePublishingStrategy struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a PublishingStrategy object and makes changes based on the state read
// and what is in the PublishingStrategy.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcilePublishingStrategy) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling PublishingStrategy")

	// Fetch the PublishingStrategy instance
	instance := &cloudingressv1alpha1.PublishingStrategy{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if k8serr.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Loop through ingress controllers defined in PublishingStrategy spec
	for _, ingressDefinition := range instance.Spec.ApplicationIngress {

		// Set the namespaced name
		namespacedName := types.NamespacedName{Name: getIngressName(ingressDefinition.DNSName), Namespace: ingressControllerNamespace}
		// Handle the "default" IngressController
		if ingressDefinition.Default {
			namespacedName = types.NamespacedName{Name: "default", Namespace: ingressControllerNamespace}
		}

		// Generate IngressController based on the applicationIngress definition
		desiredIngressController := generateIngressController(ingressDefinition)

		// Try to get matching IngressController
		ingressController := &operatorv1.IngressController{}
		err = r.client.Get(context.TODO(), namespacedName, ingressController)
		if err != nil {
			// Attempt to create the CR if not found
			if k8serr.IsNotFound(err) {
				err = r.client.Create(context.TODO(), desiredIngressController)
				if err != nil {
					return reconcile.Result{}, err
				}
				// If the CR was created, requeue PublishingStrategy
				return reconcile.Result{Requeue: true}, nil
			}
			return reconcile.Result{}, err
		}

		// Check Spec fields that cannot be patched vs desired IngressController
		if !validateStaticSpec(*ingressController, desiredIngressController.Spec) {
			// "default" CR also needs a status check
			if ingressDefinition.Default {
				if !validateStaticStatus(*ingressController, desiredIngressController.Spec) {
					//Delete IngressController as it doesn't match ApplicationIngress
					r.client.Delete(context.TODO(), ingressController)
					return reconcile.Result{Requeue: true}, nil
				}
			} else {
				//Delete IngressController as it doesn't match ApplicationIngress
				r.client.Delete(context.TODO(), ingressController)
				return reconcile.Result{Requeue: true}, nil
			}
		}

		// Check Spec fields that CAN be patched vs desired IngressController
		if valid, field := validatePatchableSpec(*ingressController, desiredIngressController.Spec); !valid {
			baseToPatch := client.MergeFrom(ingressController.DeepCopy())
			if ingressDefinition.Default {
				if valid, field := validatePatchableStatus(*ingressController, desiredIngressController.Spec); valid {
					// Patch CR
					if field == IngressControllerSelector {
						ingressController.Spec.RouteSelector = desiredIngressController.Spec.RouteSelector
					} else if field == IngressControllerCertificate {
						ingressController.Spec.DefaultCertificate = desiredIngressController.Spec.DefaultCertificate
					}
					err = r.client.Patch(context.TODO(), ingressController, baseToPatch)
					if err != nil {
						return reconcile.Result{}, err
					}
					return reconcile.Result{Requeue: true}, nil
				}
			} else {
				// Patch CR
				if field == IngressControllerSelector {
					ingressController.Spec.RouteSelector = desiredIngressController.Spec.RouteSelector
				} else if field == IngressControllerCertificate {
					ingressController.Spec.DefaultCertificate = desiredIngressController.Spec.DefaultCertificate
				}
				err = r.client.Patch(context.TODO(), ingressController, baseToPatch)
				if err != nil {
					return reconcile.Result{}, err
				}
				return reconcile.Result{Requeue: true}, nil
			}
		}
	}

	cloudPlatform, err := baseutils.GetPlatformType(r.client)
	if err != nil {
		log.Error(err, "Failed to create a Cloud Client")
		return reconcile.Result{}, err
	}
	cloudClient := cloudclient.GetClientFor(r.client, *cloudPlatform)

	// Discard the error since it's just for logging messages.
	// In case of failure, clusterBaseDomain is an empty string.
	clusterBaseDomain, _ := baseutils.GetClusterBaseDomain(r.client)

	if instance.Spec.DefaultAPIServerIngress.Listening == cloudingressv1alpha1.Internal {
		err := cloudClient.SetDefaultAPIPrivate(context.TODO(), r.client, instance)
		if err != nil {
			log.Error(err, fmt.Sprintf("Error updating api.%s alias to internal NLB", clusterBaseDomain))
			return reconcile.Result{}, err
		}
		log.Info(fmt.Sprintf("Update api.%s alias to internal NLB successful", clusterBaseDomain))
		return reconcile.Result{}, nil
	}

	// if CR is wanted the default server API to be internet-facing, we
	// create the external NLB for port 6443/TCP and add api.<cluster-name> DNS record to point to external NLB
	if instance.Spec.DefaultAPIServerIngress.Listening == cloudingressv1alpha1.External {
		err := cloudClient.SetDefaultAPIPublic(context.TODO(), r.client, instance)
		if err != nil {
			log.Error(err, fmt.Sprintf("Error updating api.%s alias to internal NLB", clusterBaseDomain))
			return reconcile.Result{}, err
		}
		log.Info(fmt.Sprintf("Update api.%s alias to internal NLB successful", clusterBaseDomain))
		return reconcile.Result{}, nil
	}
	return reconcile.Result{}, nil
}

// getIngressName takes the domain name and returns the first part
func getIngressName(dnsName string) string {
	firstPeriodIndex := strings.Index(dnsName, ".")
	newIngressName := dnsName[:firstPeriodIndex]
	return newIngressName
}

func generateIngressController(appIngress v1alpha1.ApplicationIngress) *operatorv1.IngressController {
	loadBalancerScope := operatorv1.LoadBalancerScope("")
	switch appIngress.Listening {
	case "internal":
		loadBalancerScope = operatorv1.InternalLoadBalancer
	case "external":
		loadBalancerScope = operatorv1.ExternalLoadBalancer
	default:
		loadBalancerScope = operatorv1.ExternalLoadBalancer
	}

	return &operatorv1.IngressController{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getIngressName(appIngress.DNSName),
			Namespace: ingressControllerNamespace,
			Annotations: map[string]string{
				"Owner": "cloud-ingress-operator",
			},
		},
		Spec: operatorv1.IngressControllerSpec{
			DefaultCertificate: &corev1.LocalObjectReference{
				Name: appIngress.Certificate.Name,
			},
			Domain: appIngress.DNSName,
			EndpointPublishingStrategy: &operatorv1.EndpointPublishingStrategy{
				Type: operatorv1.LoadBalancerServiceStrategyType,
				LoadBalancer: &operatorv1.LoadBalancerStrategy{
					Scope: loadBalancerScope,
				},
			},
			RouteSelector: &metav1.LabelSelector{
				MatchLabels: appIngress.RouteSelector.MatchLabels,
			},
		},
	}
}

func validateStaticStatus(ingressController operatorv1.IngressController, desiredSpec operatorv1.IngressControllerSpec) bool {

	if !(desiredSpec.Domain == ingressController.Status.Domain) {
		return false
	}
	if !(desiredSpec.EndpointPublishingStrategy.LoadBalancer.Scope == ingressController.Status.EndpointPublishingStrategy.LoadBalancer.Scope) {
		return false
	}

	return true
}

func validateStaticSpec(ingressController operatorv1.IngressController, desiredSpec operatorv1.IngressControllerSpec) bool {
	if !(desiredSpec.Domain == ingressController.Spec.Domain) {
		return false
	}
	if !(desiredSpec.EndpointPublishingStrategy.LoadBalancer.Scope == ingressController.Spec.EndpointPublishingStrategy.LoadBalancer.Scope) {
		return false
	}

	return true
}

func validatePatchableStatus(ingressController operatorv1.IngressController, desiredSpec operatorv1.IngressControllerSpec) (bool, patchField) {
	ingressControllerSelector, _ := metav1.ParseToLabelSelector(ingressController.Status.Selector)
	if ingressControllerSelector != nil {
		if !(reflect.DeepEqual(desiredSpec.RouteSelector.MatchLabels, ingressControllerSelector.MatchLabels)) {
			return false, IngressControllerSelector
		}
	} else {
		if desiredSpec.RouteSelector != nil {
			return false, IngressControllerSelector
		}
	}

	return true, ""
}

func validatePatchableSpec(ingressController operatorv1.IngressController, desiredSpec operatorv1.IngressControllerSpec) (bool, patchField) {
	if !(reflect.DeepEqual(desiredSpec.RouteSelector.MatchLabels, ingressController.Spec.RouteSelector.MatchLabels)) {
		return false, IngressControllerSelector
	}

	if !(desiredSpec.DefaultCertificate.Name == ingressController.Spec.DefaultCertificate.Name) {
		return false, IngressControllerCertificate
	}

	return true, ""
}
