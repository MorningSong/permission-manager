package kube

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
)

var clusterName, clusterControlPlaceAddress string

func init() {
	clusterName = os.Getenv("CLUSTER_NAME")
	if clusterName == "" {
		log.Fatal("CLUSTER_NAME env cannot be empty")
	}

	clusterControlPlaceAddress = os.Getenv("CONTROL_PLANE_ADDRESS")
	if clusterControlPlaceAddress == "" {
		log.Fatal("CONTROL_PLANE_ADDRESS env cannot be empty")
	}
}

// CreateKubeconfigYAML returns a kubeconfig YAML string
func CreateKubeconfigYAML(kc kubernetes.Interface, username string) (kubeconfigYAML string) {
	priv, privPem := createRsaPrivateKeyPem()
	certificatePemBytes := getSignedCertificateForUser(kc, username, priv)

	ca := ""
	/* REFACTOR: read and encode base64 from go */
	if os.Getenv("KUBERNETES_SERVICE_HOST") == "" {
		fp := filepath.Join(os.Getenv("HOME"), ".minikube", "ca.crt")
		s := fmt.Sprintf("cat %s | base64 | tr -d '\n'", fp)
		caBase64, err := exec.Command("sh", "-c", s).Output()
		if err != nil {
			panic(err)
		}
		ca = "certificate-authority-data: " + string(caBase64)
	} else {
		fmt.Println("detected runnig inside cluster")
		s := "cat /var/run/secrets/kubernetes.io/serviceaccount/ca.crt | base64 | tr -d '\n'"
		caBase64, err := exec.Command("sh", "-c", s).Output()
		if err != nil {
			panic(err)
		}
		ca = "certificate-authority-data: " + string(caBase64)
	}

	crtBase64 := base64.StdEncoding.EncodeToString(certificatePemBytes)
	privateKeyBase64 := base64.StdEncoding.EncodeToString(privPem)

	kubeconfigYAML = fmt.Sprintf(`apiVersion: v1
kind: Config
preferences:
    colors: true
current-context: %s
clusters:
  - name: %s
    cluster:
      server: %s
      %s
contexts:
  - context:
      cluster: %s
      user: %s
    name: %s
users:
  - name: %s
    user:
      client-certificate-data: %s
      client-key-data: %s`,
		clusterName, clusterName, clusterControlPlaceAddress, ca, clusterName, username, clusterName, username, crtBase64, privateKeyBase64)

	return kubeconfigYAML
}
