package kubeconfig

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	certificatesv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	privateKeyBits              = 2048
	privateKeyType              = "RSA PRIVATE KEY"
	certificateRequestType      = "CERTIFICATE REQUEST"
	clientCertificateSignerName = "kubernetes.io/kube-apiserver-client"
)

// GetClientCertDates returns the NotBefore and NotAfter dates of the client certificate in the kubeconfig
func GetClientCertDates(kubeconfigData []byte, username string) (notBefore, notAfter time.Time, err error) {
	config, err := clientcmd.Load(kubeconfigData)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("failed to load kubeconfig: %v", err)
	}

	userInfo, exists := config.AuthInfos[username]
	if !exists {
		return time.Time{}, time.Time{}, fmt.Errorf("user %q not found in kubeconfig", username)
	}

	if len(userInfo.ClientCertificateData) == 0 {
		return time.Time{}, time.Time{}, fmt.Errorf("no client certificate data found for user %q", username)
	}

	// Decode current client cert
	currentCert, err := tls.X509KeyPair(userInfo.ClientCertificateData, userInfo.ClientKeyData)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("failed to parse client certificate: %v", err)
	}

	x509Cert, err := x509.ParseCertificate(currentCert.Certificate[0])
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("failed to parse x509 certificate: %v", err)
	}

	return x509Cert.NotBefore, x509Cert.NotAfter, nil
}

// ErrCSRProcessing describes when the CSR is being created or approved. Useful to reconcile on this error
var ErrCSRProcessing = errors.New("CSR is still processing")

// RegenerateClientCertWithCSR will return a kubeconfig with a regenerated client-certificate
// returns ErrCSRProcessing if the CSR has been created but not processed yet. We will need to requeue and reconcile if
// the returned error is this.
func RegenerateClientCertWithCSR(ctx context.Context, k8sClient client.Client, kubeconfig []byte, username string) ([]byte, error) {
	config, err := clientcmd.Load(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("error loading kubeconfig: %v", err)
	}
	if _, ok := config.AuthInfos[username]; !ok {
		return nil, fmt.Errorf("user %s not present in kubeconfig", username)
	}
	privateKey, err := rsa.GenerateKey(rand.Reader, privateKeyBits)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %v", err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  privateKeyType,
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	csrName := fmt.Sprintf("%s-%s", username, ShortHash(kubeconfig))

	// if CSR already present, check if it's approved
	var signedCert []byte
	csr := &certificatesv1.CertificateSigningRequest{}
	if err := k8sClient.Get(ctx, client.ObjectKey{Name: csrName}, csr); err != nil {
		// if CSR not created, create it and return

		csr, err = newCSR(username, privateKey, csrName)
		if err != nil {
			return nil, err
		}

		if err := k8sClient.Create(ctx, csr); err != nil {
			return nil, fmt.Errorf("failed to create CSR: %v", err)
		}

		// CSR is now created, but it needs auto-approval
		return nil, errors.Join(ErrCSRProcessing, fmt.Errorf("CSR created but not auto-approved"))
	}

	// CSR is found, check if auto-approved
	if condition := getCSRStatusCondition(csr.Status.Conditions, certificatesv1.CertificateApproved, corev1.ConditionTrue); condition != nil {
		if csr.Status.Certificate == nil {
			return nil, errors.Join(ErrCSRProcessing, fmt.Errorf("certificate is auto-approved, but not generated yet"))
		}
		signedCert = csr.Status.Certificate
		return newKubeconfig(config, username, signedCert, keyPEM)

	}

	// CSR is not auto-approved, auto-approve it
	csr.Status.Conditions = append(csr.Status.Conditions, autoApproveCSR())
	if err := k8sClient.Update(ctx, csr); err != nil {
		return nil, fmt.Errorf("cannot auto-approve CSR: %v", err)
	}
	return nil, errors.Join(ErrCSRProcessing, fmt.Errorf("certificate not auto-approved yet"))
}

// Create a new CSR object
func newCSR(username string, privateKey *rsa.PrivateKey, csrName string) (*certificatesv1.CertificateSigningRequest, error) {
	// Create CSR template
	template := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName: username,
		},
	}

	csrDER, err := x509.CreateCertificateRequest(rand.Reader, template, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create CSR: %v", err)
	}

	csrPEM := pem.EncodeToMemory(&pem.Block{
		Type:  certificateRequestType,
		Bytes: csrDER,
	})

	// Create Kubernetes CSR object
	csr := &certificatesv1.CertificateSigningRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: csrName,
		},
		Spec: certificatesv1.CertificateSigningRequestSpec{
			Request:    csrPEM,
			SignerName: clientCertificateSignerName,
			Usages: []certificatesv1.KeyUsage{
				certificatesv1.UsageClientAuth,
				certificatesv1.UsageDigitalSignature,
				certificatesv1.UsageKeyEncipherment,
			},
		},
	}
	return csr, nil
}

// Generate condition to approve CSR
func autoApproveCSR() certificatesv1.CertificateSigningRequestCondition {
	return certificatesv1.CertificateSigningRequestCondition{
		Type:    certificatesv1.CertificateApproved,
		Status:  corev1.ConditionTrue,
		Reason:  "AutoApproved",
		Message: "Auto-approved by client certificate rotation",
	}
}

// Helper to retrieve a given condition in a given status
func getCSRStatusCondition(conditions []certificatesv1.CertificateSigningRequestCondition, conditionType certificatesv1.RequestConditionType, conditionStatus corev1.ConditionStatus) *certificatesv1.CertificateSigningRequestCondition {
	for _, condition := range conditions {
		if condition.Type == conditionType && condition.Status == conditionStatus {
			return &condition
		}
	}
	return nil

}

// Create a new kubeconfig with the new cert and key for a given username
func newKubeconfig(kubeconfig *clientcmdapi.Config, username string, signedCert, keyPEM []byte) ([]byte, error) {
	newConfig := kubeconfig.DeepCopy()
	newConfig.AuthInfos[username].ClientCertificateData = signedCert
	newConfig.AuthInfos[username].ClientKeyData = keyPEM

	return clientcmd.Write(*newConfig)
}

// ShortHash is a utility to generate consistent naming for CSR
func ShortHash(data []byte) string {
	sum := sha256.Sum256(data)
	return base64.URLEncoding.EncodeToString(sum[:6])
}
