package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/cli-runtime/pkg/printers"
)

func Pod(cpu string, memory string, name string, image string, env map[string]string, args []string) corev1.Pod {

	yes := true
	no := false

	Env := []corev1.EnvVar{}
	for k, v := range env {
		Env = append(Env, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	cpuReq := resource.MustParse(cpu)
	memoryReq := resource.MustParse(memory)

	// FIXME: Add support for setting limits separately in some good way
	cpuLimit := resource.MustParse(cpu)
	memoryLimit := resource.MustParse(memory)
	cpuLimit.Add(cpuLimit)
	memoryLimit.Add(memoryLimit)

	return corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app": name,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  name,
					Image: image,
					Env:   Env,
					Args:  args,
					SecurityContext: &corev1.SecurityContext{
						AllowPrivilegeEscalation: &no,
						Capabilities: &corev1.Capabilities{
							Drop: []corev1.Capability{"ALL"},
						},
						ReadOnlyRootFilesystem: &yes,
						RunAsNonRoot:           &yes,
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    cpuReq,
							corev1.ResourceMemory: memoryReq,
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    cpuLimit,
							corev1.ResourceMemory: memoryLimit,
						},
					},
				},
			},
		},
	}
}

func Deployment(name string, replicas int32, pod corev1.Pod) appsv1.Deployment {
	return appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": name,
					},
				},
				Spec: pod.Spec,
			},
		},
	}
}

func Service(name string, port int32, targetPort int32, selector map[string]string) corev1.Service {
	return corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: corev1.ServiceSpec{
			Selector: selector,
			Ports: []corev1.ServicePort{
				{
					Name:       name,
					Port:       port,
					TargetPort: intstr.FromInt(int(targetPort)),
				},
			},
		},
	}
}

func HostIngress(host string, serviceName string, servicePort int32) netv1.Ingress {

	implementationSpecific := netv1.PathTypeImplementationSpecific
	nginx := "nginx"

	return netv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "networking.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: host,
		},
		Spec: netv1.IngressSpec{
			IngressClassName: &nginx,
			Rules: []netv1.IngressRule{
				{
					Host: host,
					IngressRuleValue: netv1.IngressRuleValue{
						HTTP: &netv1.HTTPIngressRuleValue{
							Paths: []netv1.HTTPIngressPath{
								{
									PathType: &implementationSpecific,
									Path:     "/",
									Backend: netv1.IngressBackend{
										Service: &netv1.IngressServiceBackend{
											Name: serviceName,
											Port: netv1.ServiceBackendPort{
												Number: servicePort,
											},
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
}

func printOrDie(printer printers.ResourcePrinter, obj runtime.Object) {
	fmt.Println("---")

	// FIXME: kubectl-neat only does one object at a time so we need to
	// pipe the output for each print.
	neat := exec.Command("kubectl-neat")
	pipeR, pipeW := io.Pipe()
	neat.Stdin = pipeR
	neat.Stdout = os.Stdout

	neat.Start()

	err := printer.PrintObj(obj, pipeW)
	if err != nil {
		panic(err)
	}
	pipeW.Close()
	neat.Wait()
}

func renderApp(printer printers.ResourcePrinter, servicePort int, replicas int32) {

	echoPort := 3003
	args := []string{"-text=hello", "-listen=:" + strconv.Itoa(echoPort)}
	echo := Pod("100m", "32Mi", "echo", "hashicorp/http-echo:1.0", map[string]string{}, args)

	deployEcho := Deployment("echo", replicas, echo)

	srv := Service("echo", int32(servicePort), int32(echoPort), map[string]string{"app": "echo"})

	printOrDie(printer, &deployEcho)
	printOrDie(printer, &srv)
}

func main() {
	env := flag.String("env", "local", "Deploy environment")
	flag.Parse()

	p := printers.YAMLPrinter{}
	servicePort := 3000
	replicaCount := 3
	withIngress := false

	switch *env {
	case "local":
		replicaCount = 1
		withIngress = true
	case "staging":
		replicaCount = 2
	case "production":
		replicaCount = 3
	default:
		panic("Unsupported environment.")
	}

	renderApp(&p, servicePort, int32(replicaCount))
	if withIngress {
		ingress := HostIngress("echo.local", "echo", int32(servicePort))
		printOrDie(&p, &ingress)
	}
}
