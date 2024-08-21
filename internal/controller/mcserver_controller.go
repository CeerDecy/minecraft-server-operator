/*
Copyright 2024.

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
	"io"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"os"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	minecraftserverv1 "ceerdecy.com/minecraft-server-operator/api/v1"
)

var PersistentVolumeFilesystem = corev1.PersistentVolumeFilesystem

// McServerReconciler reconciles a McServer object
type McServerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=minecraft-server.ceerdecy.com,resources=mcservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=minecraft-server.ceerdecy.com,resources=mcservers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=minecraft-server.ceerdecy.com,resources=mcservers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the McServer object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/reconcile
func (r *McServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("start reconcile")

	var mcServer minecraftserverv1.McServer
	var result ctrl.Result

	err := r.Get(ctx, req.NamespacedName, &mcServer)
	if err != nil {
		logger.Error(err, "failed to get mcServer")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	mcServer.Status.Version = mcServer.Spec.Version
	mcServer.Status.State = "Running"

	err = r.Status().Update(ctx, &mcServer)
	if err != nil {
		logger.Error(err, "failed to get mcServer")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mc-server-default-properties",
		},
	}

	_, err = ctrl.CreateOrUpdate(ctx, r.Client, cm, func() error {
		buildDefaultConfigMap(&mcServer, cm)
		return ctrl.SetControllerReference(&mcServer, cm, r.Scheme)
	})

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mcServer.Name,
			Namespace: mcServer.Namespace,
		},
	}

	_, err = ctrl.CreateOrUpdate(ctx, r.Client, sts, func() error {
		buildStatefulSet(&mcServer, sts)
		return ctrl.SetControllerReference(&mcServer, sts, r.Scheme)
	})

	svc := buildService(&mcServer)

	_, err = ctrl.CreateOrUpdate(ctx, r.Client, svc, func() error {
		return ctrl.SetControllerReference(&mcServer, svc, r.Scheme)
	})

	logger.Info("end reconcile")

	return result, nil
}

func buildDefaultConfigMap(mc *minecraftserverv1.McServer, cm *corev1.ConfigMap) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	configPath := filepath.Join(dir, "config/minecraft/server.properties")
	bytes, err := os.Open(configPath)
	if err != nil {
		return err
	}
	properties, err := io.ReadAll(bytes)
	cm.Data["server.properties"] = string(properties)
	return nil
}

func buildService(mc *minecraftserverv1.McServer) *corev1.Service {
	var svc corev1.Service

	return &svc
}

func buildStatefulSet(mc *minecraftserverv1.McServer, sts *appsv1.StatefulSet) *appsv1.StatefulSet {
	labels := mc.NewLabels()
	podLabels := make(map[string]string, len(labels)+len(mc.Labels))

	for k, v := range mc.Labels {
		podLabels[k] = v
	}

	sts.Spec = appsv1.StatefulSetSpec{
		ServiceName: mc.Name,
		Replicas:    ptr.To(int32(1)),
		Selector: &metav1.LabelSelector{
			MatchLabels: mc.Labels,
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: mc.Annotations,
				Labels:      podLabels,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:      "mc-server",
						Image:     mc.Spec.Image,
						Resources: mc.Spec.Resources,
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "world",
								MountPath: "/server/world",
							},
						},
						Ports: []corev1.ContainerPort{
							{ContainerPort: 25565},
						},
					},
					{
						Name: "mc-status",
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("100m"),
								corev1.ResourceMemory: resource.MustParse("100Mi"),
							},
						},
						Image: "registry.cn-hangzhou.aliyuncs.com/ceerdecy/minecraft-server:statuser-1.0.0",
						Ports: []corev1.ContainerPort{
							{ContainerPort: 25566},
						},
					},
				},
			},
		},
		VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "world",
					Namespace: mc.Namespace,
					Labels:    labels,
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: mc.Spec.StorageSize,
						},
					},
					StorageClassName: &mc.Spec.StorageClassName,
					VolumeMode:       &PersistentVolumeFilesystem,
				},
			},
		},
	}

	return sts
}

// SetupWithManager sets up the controller with the Manager.
func (r *McServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&minecraftserverv1.McServer{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
