package secret

import (
	"context"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	zap "sigs.k8s.io/controller-runtime/pkg/log/zap"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestSecretControllerShouldUpdateSecrets(t *testing.T) {
	logf.SetLogger(zap.Logger(true))
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
			Annotations: map[string]string{
				"hash": "",
			},
			Labels: map[string]string{
				"sso.gable.dev/secret": "true",
			},
		},
		Data: map[string][]byte{
			"test": []byte("aW5pdGlhbA=="),
		},
	}

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
			Labels: map[string]string{
				"sso.gable.dev/secret": "test-secret",
				"updatedSecretAt":      "0",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "myapp",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
					Labels: map[string]string{
						"app":             "my-app",
						"updatedSecretAt": "0",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						corev1.Container{
							Name:  "test-container",
							Image: "busybox",
							Command: []string{
								"sh",
								"-c",
								"echo Hello Kubernetes! && sleep 3600",
							},
							Env: []corev1.EnvVar{
								corev1.EnvVar{
									Name: "secret-value",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "test-secret",
											},
											Key: "test",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	objs := []runtime.Object{secret, deployment}

	s := scheme.Scheme
	s.AddKnownTypes(appsv1.SchemeGroupVersion, deployment)
	s.AddKnownTypes(corev1.SchemeGroupVersion, secret)
	cl := fake.NewFakeClient(objs...)

	opt := client.MatchingLabels(map[string]string{"sso.gable.dev/secret": "test-secret"})
	secretList := &corev1.SecretList{}
	err := cl.List(context.TODO(), secretList, opt)
	if err != nil {
		t.Fatalf("list secrets: (%v)", err)
	}

	r := &ReconcileSecret{client: cl, scheme: s}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-secret",
			Namespace: "default",
		},
	}

	res, err := r.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	} else {
		t.Log(res)
	}
	testSecret := &corev1.Secret{}
	err = r.client.Get(context.TODO(), req.NamespacedName, testSecret)
	if err != nil {
		t.Fatalf("get secret: (%v)", err)
	}
	testSecret.Data["test"] = []byte("c3VjY2Vzcw==")
	err = r.client.Update(context.TODO(), testSecret)
	updateSecretRes, err := r.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}

	if !updateSecretRes.Requeue {
		t.Log("reconcile did not requeue request as expected")
	}
	if err != nil {
		t.Fatalf("Failed to update secret: (%v)", err)
	}
	deploymentReq := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-deployment",
			Namespace: "default",
		},
	}

	deployRes, err := r.Reconcile(deploymentReq)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}

	testDeployment := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), deploymentReq.NamespacedName, testDeployment)
	if err != nil {
		t.Fatalf("Failed to get deployment: (%v)", err)
	}
	if !deployRes.Requeue {
		t.Log("reconcile did not requeue request as expected")
	}
	if testDeployment.Labels["updatedSecretAt"] == "0" {
		t.Error("Deployment was not updated by controller")
	}
}
