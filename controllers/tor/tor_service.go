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

package tor

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	torv1alpha2 "github.com/bugfest/tor-controller/apis/tor/v1alpha2"
)

func (r *TorReconciler) reconcileService(ctx context.Context, tor *torv1alpha2.Tor) error {
	log := log.FromContext(ctx)

	serviceName := tor.ServiceName()
	namespace := tor.Namespace
	if serviceName == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		runtime.HandleError(fmt.Errorf("service name must be specified"))
		return nil
	}

	var service corev1.Service
	err := r.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: namespace}, &service)

	newService := torService(tor)
	if errors.IsNotFound(err) {

		if len(newService.Spec.Ports) == 0 {
			log.Info("No ports enabled, skipping service for this tor instance")
			return nil
		}

		err := r.Create(ctx, newService)
		if err != nil {
			return err
		}
		service = *newService
	} else if err != nil {
		return err
	}

	if !metav1.IsControlledBy(&service.ObjectMeta, tor) {
		log.Info(fmt.Sprintf("Service %s already exists and is not controller by %s", service.Name, tor.Name))
		return nil
	}

	// If the service specs don't match, update
	if !serviceEqual(&service, newService) {
		err := r.Update(ctx, newService)
		if err != nil {
			return fmt.Errorf("filed to update Service %#v", newService)
		}
	}

	return nil
}

func torService(tor *torv1alpha2.Tor) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tor.ServiceName(),
			Namespace: tor.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(tor, schema.GroupVersionKind{
					Group:   torv1alpha2.GroupVersion.Group,
					Version: torv1alpha2.GroupVersion.Version,
					Kind:    "Tor",
				}),
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: tor.ServiceSelector(),
			Ports:    getTorServicePortList(tor),
		},
	}
}

func getTorServicePortList(tor *torv1alpha2.Tor) []corev1.ServicePort {
	ports := []corev1.ServicePort{}

	for _, r := range tor.GetAllPorts() {
		if r.Port.Enable {
			port := corev1.ServicePort{
				Name:       r.Name,
				TargetPort: intstr.FromInt(int(r.Port.Port)),
				Port:       r.Port.Port,
				Protocol:   corev1.Protocol(r.Protocol),
			}
			ports = append(ports, port)
		}
	}

	return ports
}
