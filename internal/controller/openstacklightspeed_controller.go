/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package controller implements OpenShift reconciliation loops for OpenStack Lightspeed.
package controller

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/go-logr/logr"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	common_helper "github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	apiv1beta1 "github.com/openstack-k8s-operators/lightspeed-operator/api/v1beta1"
)

// OpenStackLightspeedReconciler reconciles a OpenStackLightspeed object
type OpenStackLightspeedReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Kclient kubernetes.Interface
}

// GetLogger returns a logger object with a prefix of "controller.name" and additional controller context fields
func (r *OpenStackLightspeedReconciler) GetLogger(ctx context.Context) logr.Logger {
	return log.FromContext(ctx).WithName("Controllers").WithName("OpenStackLightspeed")
}

// +kubebuilder:rbac:groups=lightspeed.openstack.org,resources=openstacklightspeeds,verbs=get;list;watch;patch
// +kubebuilder:rbac:groups=lightspeed.openstack.org,resources=openstacklightspeeds/status,verbs=patch
// +kubebuilder:rbac:groups=lightspeed.openstack.org,resources=openstacklightspeeds/finalizers,verbs=update
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,verbs=get;list;watch;create;patch;deletecollection
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=get;list;watch;create;patch;deletecollection
// +kubebuilder:rbac:groups=config.openshift.io,resources=clusterversions,verbs=get;list;watch
// SAR role escalation: the operator creates a ClusterRole granting pull-secret GET,
// so it must hold that permission itself (K8s RBAC escalation prevention).
// +kubebuilder:rbac:groups="",resources=secrets,resourceNames=pull-secret,verbs=get
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get
// +kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,namespace=openstack-lightspeed,verbs=get;list;watch;create;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,namespace=openstack-lightspeed,verbs=get;list;watch;create;patch
// +kubebuilder:rbac:groups="",resources=configmaps,namespace=openstack-lightspeed,verbs=get;list;watch;create;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,namespace=openstack-lightspeed,verbs=get;list;watch;create;patch;delete;deletecollection
// +kubebuilder:rbac:groups="",resources=services,namespace=openstack-lightspeed,verbs=get;list;watch;create;patch
// +kubebuilder:rbac:groups="",resources=serviceaccounts,namespace=openstack-lightspeed,verbs=get;list;watch;create;patch
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,namespace=openstack-lightspeed,verbs=get;list;watch;create;patch

// Reconcile reads the state of the cluster for a OpenStackLightspeed object and makes changes towards the state defined in the spec.
func (r *OpenStackLightspeedReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, e error) {
	Log := r.GetLogger(ctx)
	Log.Info("OpenStackLightspeed Reconciling")

	instance, err := r.GetOpenStackLightspeed(ctx, req)
	if err != nil {
		Log.Error(err, "Cannot reconcile OpenStackLightspeed")
		return ctrl.Result{}, err
	}
	if instance == nil {
		Log.Info("No OpenStackLightspeed CR matches the reconcile request", "name", req.Name, "namespace", req.Namespace)
		return ctrl.Result{}, nil
	}

	helper, err := common_helper.NewHelper(
		instance,
		r.Client,
		r.Kclient,
		r.Scheme,
		Log,
	)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Save a copy of the conditions so that we can restore the LastTransitionTime
	// when a condition's state doesn't change.
	savedConditions := instance.Status.Conditions.DeepCopy()

	// Always patch the instance status when exiting this function so we can persist any changes.
	defer func() {
		// Don't update the status, if reconciler Panics
		if r := recover(); r != nil {
			Log.Info(fmt.Sprintf("panic during reconcile %v\n", r))
			panic(r)
		}

		// 1) Restore subconditions first so Mirror tie-breaks on real LTTs
		condition.RestoreLastTransitionTimes(&instance.Status.Conditions, savedConditions)

		if instance.Status.Conditions.AllSubConditionIsTrue() {
			instance.Status.Conditions.MarkTrue(
				condition.ReadyCondition, condition.ReadyMessage)
		} else {
			// something is not ready so reset the Ready condition
			instance.Status.Conditions.MarkUnknown(
				condition.ReadyCondition, condition.InitReason, condition.ReadyInitMessage)
			// and recalculate it based on the state of the rest of the conditions
			instance.Status.Conditions.Set(
				instance.Status.Conditions.Mirror(condition.ReadyCondition))
		}

		// 2) Restore again so Ready LTT is preserved after aggregation
		condition.RestoreLastTransitionTimes(&instance.Status.Conditions, savedConditions)

		err := helper.PatchInstance(ctx, instance)
		if err != nil {
			return
		}

	}()

	cl := condition.CreateList(
		condition.UnknownCondition(
			apiv1beta1.OpenStackLightspeedReadyCondition,
			condition.InitReason,
			apiv1beta1.OpenStackLightspeedReadyInitMessage,
		),
	)

	instance.Status.Conditions.Init(&cl)
	instance.Status.ObservedGeneration = instance.Generation

	if !instance.DeletionTimestamp.IsZero() {
		if err := r.reconcileDelete(ctx, helper, instance); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if instance.DeletionTimestamp.IsZero() && controllerutil.AddFinalizer(instance, helper.GetFinalizer()) {
		return ctrl.Result{}, nil
	}

	if err := validateModelSelectionAndSetDefaults(Log, instance); err != nil {
		return ctrl.Result{}, err
	}

	// Log dev config parse errors so misconfigurations don't silently disable features.
	if _, err := instance.ParseDevConfig(); err != nil {
		Log.Error(err, "failed to parse dev config, ignoring")
	}

	reconcileTasks := []ReconcileTask{
		{Name: "PostgresResources", Task: ReconcilePostgresResources},
		{Name: "PostgresDeployment", Task: ReconcilePostgresDeployment},
		{Name: "OKPDeployment", Task: ReconcileOKPDeployment},
		{Name: "LCoreResources", Task: ReconcileLCoreResources},
		{Name: "LCoreDeployment", Task: ReconcileLCoreDeployment},
	}

	if err := ReconcileTasks(ctx, helper, instance, reconcileTasks); err != nil {
		Log.Error(err, "reconcile tasks failed")
		instance.Status.Conditions.Set(condition.FalseCondition(
			apiv1beta1.OpenStackLightspeedReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			apiv1beta1.DeploymentCheckFailedMessage,
		))
		return ctrl.Result{}, err
	}

	return r.reconcileStatus(ctx, helper, instance)
}

// reconcileDelete reconciles the deletion of OpenStackLightspeed instance
func validateModelSelectionAndSetDefaults(Log logr.Logger, instance *apiv1beta1.OpenStackLightspeed) error {
	modelNames := map[string]struct{}{}
	validModelNames := make([]string, 0, len(instance.Spec.Models))
	for i := range instance.Spec.Models {
		if instance.Spec.Models[i].MaxTokensForResponse == 0 {
			instance.Spec.Models[i].MaxTokensForResponse = apiv1beta1.OpenStackLightspeedDefaultValues.MaxTokensForResponse
		}
		modelName := instance.Spec.Models[i].Name
		modelNames[modelName] = struct{}{}
		validModelNames = append(validModelNames, modelName)
	}

	if _, ok := modelNames[instance.Spec.DefaultModel]; !ok {
		sort.Strings(validModelNames)
		err := fmt.Errorf(
			"spec.defaultModel %q must match one of spec.models[].name: [%s]",
			instance.Spec.DefaultModel,
			strings.Join(validModelNames, ", "),
		)
		Log.Error(err, "invalid model configuration")
		instance.Status.Conditions.Set(condition.FalseCondition(
			apiv1beta1.OpenStackLightspeedReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			"%s",
			err.Error(),
		))
		return err
	}

	return nil
}

func (r *OpenStackLightspeedReconciler) reconcileDelete(
	ctx context.Context,
	helper *common_helper.Helper,
	instance *apiv1beta1.OpenStackLightspeed,
) error {
	Log := r.GetLogger(ctx)
	Log.Info("OpenStackLightspeed Reconciling Delete")

	// Delete cluster-scoped resources using fail-fast pattern
	deletionTasks := []ReconcileTask{
		{Name: "DeleteSARClusterRoleBinding", Task: reconcileDeleteClusterRoleBindingByLabels},
		{Name: "DeleteSARClusterRole", Task: reconcileDeleteClusterRoleByLabels},
	}

	// Execute deletion tasks in order (fail-fast: stop on first error)
	if err := ReconcileTasksFailFast(ctx, helper, instance, deletionTasks); err != nil {
		Log.Error(err, "failed to delete cluster-scoped resources")
		return err
	}

	controllerutil.RemoveFinalizer(instance, helper.GetFinalizer())

	Log.Info("OpenStackLightspeed Reconciling Delete completed")
	return nil
}

func (r *OpenStackLightspeedReconciler) reconcileStatus(
	ctx context.Context,
	helper *common_helper.Helper,
	instance *apiv1beta1.OpenStackLightspeed,
) (ctrl.Result, error) {
	Log := r.GetLogger(ctx)
	deployments := []string{
		PostgresDeploymentName,
		OKPDeploymentName,
		LCoreDeploymentName,
	}
	for _, deploymentName := range deployments {
		deployment, err := getDeployment(ctx, helper, deploymentName, instance.Namespace)
		if err != nil {
			if k8s_errors.IsNotFound(err) {
				// Deployment not created yet, e.g. LCore waiting on its
				// Postgres/OKP dependencies. Treat the same as not-ready.
				instance.Status.Conditions.Set(condition.FalseCondition(
					apiv1beta1.OpenStackLightspeedReadyCondition,
					condition.RequestedReason,
					condition.SeverityInfo,
					apiv1beta1.DeploymentsNotReadyMessage,
					deploymentName,
				))
				return ctrl.Result{RequeueAfter: getResourcePollInterval(instance)}, nil
			}
			Log.Error(err, "failed to get deployment", "deployment", deploymentName)
			instance.Status.Conditions.Set(condition.FalseCondition(
				apiv1beta1.OpenStackLightspeedReadyCondition,
				condition.ErrorReason,
				condition.SeverityWarning,
				apiv1beta1.DeploymentCheckFailedMessage,
			))
			return ctrl.Result{}, err
		}

		if !isDeploymentReady(deployment) {
			instance.Status.Conditions.Set(condition.FalseCondition(
				apiv1beta1.OpenStackLightspeedReadyCondition,
				condition.RequestedReason,
				condition.SeverityInfo,
				apiv1beta1.DeploymentsNotReadyMessage,
				deploymentName,
			))
			return ctrl.Result{RequeueAfter: getResourcePollInterval(instance)}, nil
		}
	}

	instance.Status.Conditions.MarkTrue(
		apiv1beta1.OpenStackLightspeedReadyCondition,
		apiv1beta1.OpenStackLightspeedReadyMessage,
	)

	helper.GetLogger().Info("OpenStackLightspeed Reconciled successfully")

	return ctrl.Result{}, nil
}

// GetOpenStackLightspeed returns the instance matching the request, or nil if none matches.
// It returns an error if listing fails or more than one instance exists in the namespace.
func (r *OpenStackLightspeedReconciler) GetOpenStackLightspeed(ctx context.Context, req ctrl.Request) (*apiv1beta1.OpenStackLightspeed, error) {
	instances := &apiv1beta1.OpenStackLightspeedList{}
	if err := r.List(ctx, instances, client.InNamespace(req.Namespace)); err != nil {
		return nil, err
	}
	if len(instances.Items) > 1 {
		return nil, fmt.Errorf("only one OpenStackLightspeed instance per namespace is allowed; found %d in namespace %q",
			len(instances.Items), req.Namespace)
	}
	if len(instances.Items) == 0 || instances.Items[0].Name != req.Name {
		return nil, nil
	}
	return &instances.Items[0], nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenStackLightspeedReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := initClusterClient(mgr); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1beta1.OpenStackLightspeed{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&rbacv1.ClusterRole{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Watches(
			&corev1.PersistentVolumeClaim{},
			handler.EnqueueRequestsFromMapFunc(r.NotifyAllOpenStackLightspeeds),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Watches(
			&corev1.ConfigMap{},
			handler.EnqueueRequestsFromMapFunc(r.NotifyOpenStackLightspeedsByCAConfigMap),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}

// NotifyOpenStackLightspeedsByCAConfigMap watches ConfigMaps and triggers reconciliation when
// a user-provided CA ConfigMap (referenced by an OpenStackLightspeed CR) changes.
func (r *OpenStackLightspeedReconciler) NotifyOpenStackLightspeedsByCAConfigMap(ctx context.Context, obj client.Object) []ctrl.Request {
	var lightspeedList apiv1beta1.OpenStackLightspeedList
	if err := r.List(ctx, &lightspeedList, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}

	var requests []ctrl.Request
	for _, item := range lightspeedList.Items {
		if item.Spec.TLSCACertBundle == obj.GetName() {
			requests = append(requests, ctrl.Request{
				NamespacedName: client.ObjectKey{
					Namespace: item.GetNamespace(),
					Name:      item.GetName(),
				},
			})
		}
	}
	return requests
}

// NotifyAllOpenStackLightspeeds returns a list of reconcile requests for all OpenStackLightspeed objects.
// For namespace-scoped resources (like InstallPlan), it lists in the same namespace as the triggering object.
// For cluster-scoped resources (like ClusterVersion), it lists in all namespaces the operator can access.
func (r *OpenStackLightspeedReconciler) NotifyAllOpenStackLightspeeds(ctx context.Context, obj client.Object) []ctrl.Request {
	var lightspeedList apiv1beta1.OpenStackLightspeedList
	var err error

	// For cluster-scoped resources (no namespace), list without namespace filter
	// The operator's cache is already restricted to the watch namespace, so this is safe
	if obj.GetNamespace() == "" {
		err = r.List(ctx, &lightspeedList)
	} else {
		err = r.List(ctx, &lightspeedList, client.InNamespace(obj.GetNamespace()))
	}

	if err != nil {
		return nil
	}

	requests := make([]ctrl.Request, 0, len(lightspeedList.Items))
	for _, item := range lightspeedList.Items {
		requests = append(requests, ctrl.Request{
			NamespacedName: client.ObjectKey{
				Namespace: item.GetNamespace(),
				Name:      item.GetName(),
			},
		})
	}

	return requests
}
