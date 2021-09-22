/*
Copyright 2021.

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

package controllers

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	appsv1 "k8s.io/api/apps/v1"

	batchv1 "scratchkb.example.com/api/v1"
)

// DemoReconciler reconciles a Demo object
type DemoReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=batch.scratchkb.example.com,resources=demoes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch.scratchkb.example.com,resources=demoes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=batch.scratchkb.example.com,resources=demoes/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Demo object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *DemoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Started reconciler")

	// get the CRD info
	var demo batchv1.Demo
	if err := r.Get(ctx, req.NamespacedName, &demo); err != nil {
		log.Error(err, "cannot get the demo crd")
	}

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:            demo.Name + "-deployment",
			Namespace:       demo.Namespace,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(&demo, batchv1.GroupVersion.WithKind("Demo"))},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"deployment": demo.Spec.Label + "-deployment"},
			},
			Replicas: &demo.Spec.Replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"deployment": demo.Spec.Label + "-deployment"}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  demo.Spec.ContainerName,
							Image: demo.Spec.ContainerImage,
							Ports: []corev1.ContainerPort{{
								ContainerPort: demo.Spec.ContainerPort,
							}},
						},
						{
							Name:  demo.Spec.ProxyName,
							Image: demo.Spec.ProxyImage,
							Ports: []corev1.ContainerPort{{
								ContainerPort: demo.Spec.ProxyPort,
							}},
						},
					},
				},
			},
		},
	}

	// Check if the Deployment already exists
	found := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: deploy.Name, Namespace: deploy.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Setting controller reference for deployment")
		if err := ctrl.SetControllerReference(&demo, deploy, r.Scheme); err != nil {
			log.Error(err, "unable to set the controller reference")
		}
		log.Info(fmt.Sprintf("Creating Deployment %s/%s\n", deploy.Namespace, deploy.Name))
		err := r.Create(ctx, deploy)
		if err != nil {
			log.Error(err, fmt.Sprintf("Cannot create the Deployment %s/%s\n", deploy.Namespace, deploy.Name))
			return reconcile.Result{}, err
		}
	} else { // the deployment exists
		log.Info(fmt.Sprintf("Found the Deployment %s/%s\n", deploy.Namespace, deploy.Name))

		// examine DeletionTimestamp to determine if object is under deletion
		//if demo.ObjectMeta.DeletionTimestamp.IsZero() {
		// reacting to changing replicas in the demo CR
		if demo.Spec.Replicas != batchv1.DefaultReplicasNumber {
			// set the new number of replicas to the deployment
			deploy.Spec.Replicas = &demo.Spec.Replicas
			if err := r.Update(ctx, deploy); err != nil {
				return reconcile.Result{}, err
			}
			log.Info(fmt.Sprintf("Set the replicas number to %d", &demo.Spec.Replicas))
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DemoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1.Demo{}).
		Complete(r)
}
