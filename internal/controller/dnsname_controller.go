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

package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"

	networkingv1alpha1 "github.com/domnikl/pihole-operator/api/v1alpha1"
	"github.com/domnikl/pihole-operator/internal/pihole"
)

// DNSNameReconciler reconciles a DNSName object
type DNSNameReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	PiHole   *pihole.PiHole
}

// +kubebuilder:rbac:groups=networking.liebler.dev,resources=dnsnames,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.liebler.dev,resources=dnsnames/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.liebler.dev,resources=dnsnames/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// The Reconcile function compares the state specified by
// the DNSName object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *DNSNameReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)

	dnsName := &networkingv1alpha1.DNSName{}
	err := r.Get(ctx, req.NamespacedName, dnsName)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("DNSName resource not found. Ignoring since object must be deleted.")

			return ctrl.Result{}, nil
		}

		reqLogger.Error(err, "Failed to get DNSName")
		return ctrl.Result{}, err
	}

	reqLogger.Info("Reconciling DNSName", "Name", dnsName.Name)

	records, err := r.PiHole.GetDNSRecords()
	if err != nil {
		reqLogger.Error(err, "Failed to get DNS records")
		return ctrl.Result{}, err
	}

	for _, record := range records {
		if record.Domain == dnsName.Spec.Domain {
			reqLogger.Info("DNS record already exists")
			return ctrl.Result{}, nil
		}
	}

	err = r.PiHole.CreateDNSRecord(dnsName.Spec.Domain, dnsName.Spec.Target)
	if err != nil {
		reqLogger.Error(err, "Failed to create DNS record")
		return ctrl.Result{}, err
	}

	reqLogger.Info("Successfully created DNS record")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DNSNameReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1alpha1.DNSName{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
