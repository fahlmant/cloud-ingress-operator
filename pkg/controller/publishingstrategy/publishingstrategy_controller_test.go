package publishingstrategy

import (
	"testing"

	operatorv1 "github.com/openshift/api/operator/v1"
	cloudingressv1alpha1 "github.com/openshift/cloud-ingress-operator/pkg/apis/cloudingress/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func mockIngressControllerList() *operatorv1.IngressControllerList {
	return &operatorv1.IngressControllerList{
		Items: []operatorv1.IngressController{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "default",
				},
				Spec: operatorv1.IngressControllerSpec{
					Domain: "example-domain",
					DefaultCertificate: &corev1.LocalObjectReference{
						Name: "",
					},
				},
				Status: operatorv1.IngressControllerStatus{
					EndpointPublishingStrategy: &operatorv1.EndpointPublishingStrategy{
						Type: operatorv1.LoadBalancerServiceStrategyType,
						LoadBalancer: &operatorv1.LoadBalancerStrategy{
							Scope: operatorv1.LoadBalancerScope("Internal"),
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "non-default",
				},
				Spec: operatorv1.IngressControllerSpec{
					Domain: "example-non-default-domain",
				},
				Status: operatorv1.IngressControllerStatus{
					EndpointPublishingStrategy: &operatorv1.EndpointPublishingStrategy{
						Type: operatorv1.LoadBalancerServiceStrategyType,
						LoadBalancer: &operatorv1.LoadBalancerStrategy{
							Scope: operatorv1.LoadBalancerScope("External"),
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "default",
				},
				Spec: operatorv1.IngressControllerSpec{
					Domain: "example-domain-3",
					DefaultCertificate: &corev1.LocalObjectReference{
						Name: "",
					},
				},
				Status: operatorv1.IngressControllerStatus{
					EndpointPublishingStrategy: &operatorv1.EndpointPublishingStrategy{
						Type: operatorv1.LoadBalancerServiceStrategyType,
						// LoadBalancer: &operatorv1.LoadBalancerStrategy{
						// 	Scope: operatorv1.LoadBalancerScope("Internal"),
						// },
					},
				},
			},
		},
	}
}

func mockDefaultIngressController() *operatorv1.IngressController {
	return &operatorv1.IngressController{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
		},
		Spec: operatorv1.IngressControllerSpec{
			Domain: "example-domain",
			DefaultCertificate: &corev1.LocalObjectReference{
				Name: "",
			},
		},
		Status: operatorv1.IngressControllerStatus{
			EndpointPublishingStrategy: &operatorv1.EndpointPublishingStrategy{
				Type: operatorv1.LoadBalancerServiceStrategyType,
				LoadBalancer: &operatorv1.LoadBalancerStrategy{
					Scope: operatorv1.LoadBalancerScope("Internal"),
				},
			},
		},
	}
}

func mockNonDefaultIngressController() *operatorv1.IngressController {
	return &operatorv1.IngressController{
		ObjectMeta: metav1.ObjectMeta{
			Name: "non-default",
			Annotations: map[string]string{
				"Owner": "cloud-ingress-operator",
			},
		},
		Spec: operatorv1.IngressControllerSpec{
			Domain: "apps2.exaple-nondefault-domain-to-pass-in",
			DefaultCertificate: &corev1.LocalObjectReference{
				Name: "",
			},
		},
		Status: operatorv1.IngressControllerStatus{
			EndpointPublishingStrategy: &operatorv1.EndpointPublishingStrategy{
				Type: operatorv1.LoadBalancerServiceStrategyType,
				LoadBalancer: &operatorv1.LoadBalancerStrategy{
					Scope: operatorv1.LoadBalancerScope("Internal"),
				},
			},
		},
	}
}

func mockNonDefaultIngressNoAnnotation() *operatorv1.IngressController {
	return &operatorv1.IngressController{
		ObjectMeta: metav1.ObjectMeta{
			Name: "non-default-no-annotation",
		},
		Spec: operatorv1.IngressControllerSpec{
			Domain: "apps2.exaple-nondefault-domain-with-no-annotation",
			DefaultCertificate: &corev1.LocalObjectReference{
				Name: "",
			},
		},
		Status: operatorv1.IngressControllerStatus{
			EndpointPublishingStrategy: &operatorv1.EndpointPublishingStrategy{
				Type: operatorv1.LoadBalancerServiceStrategyType,
				LoadBalancer: &operatorv1.LoadBalancerStrategy{
					Scope: operatorv1.LoadBalancerScope("Internal"),
				},
			},
		},
	}
}

func mockApplicationIngress() *cloudingressv1alpha1.ApplicationIngress {
	return &cloudingressv1alpha1.ApplicationIngress{
		Listening: cloudingressv1alpha1.Internal,
		Default:   true,
		DNSName:   "example-domain",
		Certificate: corev1.SecretReference{
			Name: "",
		},
	}
}

func mockApplicationIngressExternal() *cloudingressv1alpha1.ApplicationIngress {
	return &cloudingressv1alpha1.ApplicationIngress{
		Listening: cloudingressv1alpha1.External,
		Default:   true,
		DNSName:   "example-domain-3",
		Certificate: corev1.SecretReference{
			Name: "",
		},
	}
}

func mockApplicationIngressNotOnCluster() *cloudingressv1alpha1.ApplicationIngress {
	return &cloudingressv1alpha1.ApplicationIngress{
		Listening: cloudingressv1alpha1.External,
		Default:   false,
		DNSName:   "example-domain-nondefault",
		Certificate: corev1.SecretReference{
			Name: "example-cert-nondefault",
		},
	}
}

func mockPublishingStrategy() *cloudingressv1alpha1.PublishingStrategy {
	return &cloudingressv1alpha1.PublishingStrategy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "testPublishingStrategy",
		},
		Spec: cloudingressv1alpha1.PublishingStrategySpec{
			DefaultAPIServerIngress: cloudingressv1alpha1.DefaultAPIServerIngress{
				Listening: cloudingressv1alpha1.External,
			},
			ApplicationIngress: []cloudingressv1alpha1.ApplicationIngress{
				{
					Listening: cloudingressv1alpha1.External,
					Default:   true,
					DNSName:   "exaple-domain-to-pass-in",
					Certificate: corev1.SecretReference{
						Name: "example-cert-default",
					},
				},
				{
					Listening: cloudingressv1alpha1.External,
					Default:   false,
					DNSName:   "apps2.exaple-nondefault-domain-to-pass-in",
					Certificate: corev1.SecretReference{
						Name: "example-nondefault-cert-default",
					},
				},
			},
		},
	}
}

func TestGetIngressName(t *testing.T) {

	domainName := "apps2.test.domain_name.org"

	expected := "apps2"
	result := getIngressName(domainName)

	if expected != result {
		t.Errorf("got %s \n, expected %s \n", result, expected)
	}
}

// create new fake k8s client to mock API calls
func newTestReconciler() *ReconcilePublishingStrategy {
	return &ReconcilePublishingStrategy{
		client: fake.NewFakeClient(),
		scheme: scheme.Scheme,
	}
}
