package kubeconfig_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	certificatesv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"math/big"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	kubecfg "github.com/openshift-assisted/cluster-api-agent/pkg/kubeconfig"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

var _ = Describe("GetClientCertDates", func() {
	var (
		clientCert []byte
		clientKey  []byte
		kubeconfig []byte
		notBefore  time.Time
		notAfter   time.Time
		username   string
	)

	BeforeEach(func() {
		var err error
		username = "test-user"

		// Generate client key pair
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		Expect(err).NotTo(HaveOccurred())

		// Set cert validity times
		notBefore = time.Now().Add(-1 * time.Hour) // Valid from 1 hour ago
		notAfter = time.Now().Add(24 * time.Hour)  // Valid for 24 hours

		// Create client cert
		template := &x509.Certificate{
			SerialNumber: big.NewInt(1),
			Subject: pkix.Name{
				CommonName: username,
			},
			NotBefore:   notBefore,
			NotAfter:    notAfter,
			KeyUsage:    x509.KeyUsageDigitalSignature,
			ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		}

		// Self-sign for test purposes
		certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
		Expect(err).NotTo(HaveOccurred())

		// Encode cert and key in PEM format
		clientCert = pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: certDER,
		})

		clientKey = pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		})

		// Create test kubeconfig
		config := clientcmdapi.NewConfig()
		config.AuthInfos[username] = &clientcmdapi.AuthInfo{
			ClientCertificateData: clientCert,
			ClientKeyData:         clientKey,
		}

		// Add a context using this user
		config.Contexts["test-context"] = &clientcmdapi.Context{
			Cluster:  "test-cluster",
			AuthInfo: username,
		}
		config.CurrentContext = "test-context"

		// Convert to bytes
		kubeconfig, err = clientcmd.Write(*config)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("GetClientCertDates", func() {
		It("should return correct certificate dates", func() {
			gotNotBefore, gotNotAfter, err := kubecfg.GetClientCertDates(kubeconfig, username)
			Expect(err).NotTo(HaveOccurred())
			Expect(gotNotBefore.Unix()).To(Equal(notBefore.Unix()))
			Expect(gotNotAfter.Unix()).To(Equal(notAfter.Unix()))
		})

		It("should return error for non-existent user", func() {
			_, _, err := kubecfg.GetClientCertDates(kubeconfig, "nonexistent")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})

		It("should return error for invalid kubeconfig", func() {
			_, _, err := kubecfg.GetClientCertDates([]byte("invalid kubeconfig"), username)
			Expect(err).To(HaveOccurred())
		})

		Context("with missing certificate data", func() {
			BeforeEach(func() {
				config, err := clientcmd.Load(kubeconfig)
				Expect(err).NotTo(HaveOccurred())
				config.AuthInfos[username].ClientCertificateData = []byte("")
				kubeconfig, err = clientcmd.Write(*config)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return error", func() {
				_, _, err := kubecfg.GetClientCertDates(kubeconfig, username)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no client certificate data found"))
			})
		})

		Context("with invalid certificate data", func() {
			BeforeEach(func() {
				config, err := clientcmd.Load(kubeconfig)
				Expect(err).NotTo(HaveOccurred())
				config.AuthInfos[username].ClientCertificateData = []byte("invalid cert data")
				kubeconfig, err = clientcmd.Write(*config)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return error", func() {
				_, _, err := kubecfg.GetClientCertDates(kubeconfig, username)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to parse"))
			})
		})
	})
})

var _ = Describe("RegenerateClientCertWithCSR", func() {
	var (
		ctx            context.Context
		k8sClient      client.Client
		kubeconfigData []byte
		username       string
		scheme         *runtime.Scheme
	)

	BeforeEach(func() {
		ctx = context.Background()
		scheme = runtime.NewScheme()
		Expect(certificatesv1.AddToScheme(scheme)).To(Succeed())

		k8sClient = fake.NewClientBuilder().
			WithScheme(scheme).
			Build()

		username = "admin"
		kubeconfigData = []byte(`apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURRRENDQWlpZ0F3SUJBZ0lJUy93MVJnaDhDTWN3RFFZSktvWklodmNOQVFFTEJRQXdQakVTTUJBR0ExVUUKQ3hNSmIzQmxibk5vYVdaME1TZ3dKZ1lEVlFRREV4OXJkV0psTFdGd2FYTmxjblpsY2kxc2IyTmhiR2h2YzNRdApjMmxuYm1WeU1CNFhEVEkxTURNd05URTNNak0xTWxvWERUTTFNRE13TXpFM01qTTFNbG93UGpFU01CQUdBMVVFCkN4TUpiM0JsYm5Ob2FXWjBNU2d3SmdZRFZRUURFeDlyZFdKbExXRndhWE5sY25abGNpMXNiMk5oYkdodmMzUXQKYzJsbmJtVnlNSUlCSWpBTkJna3Foa2lHOXcwQkFRRUZBQU9DQVE4QU1JSUJDZ0tDQVFFQXVRN3lsZWR2cUVCRAorV0QvWmJCbHZKN0MrYm40Ynh2MmtieFYrT1BoVXp2MGxsaStKV1poRGhXS2M5TDZoYXdCWGJHaHdXbTFkNkpOClppYkp2NndtMXFaYWdqTXluNlovVDFIZ0d5ekNCVTlKWUc4bzhHZ3JnN3lLbmVPSlVYblEyUzhIUGxva2FENm0KMzZjYVlMWHE1SVdTSGpucWwveS9IRllHUW5YeFprKzZQMURGNHZnckpHV2htWG5teGFjVTBnYzA5YmJwVTl1awphRm9GSlFMMzJKTkxjL0RyTmx6b3FTQWRuSXE4Z25RWGdLQVZWaXVoN0hScjM0aEtINDR5Qm9WYW4wM0JXZm9qCmlJdHRRUmJsUGNudTM5dCtqWnNqbmFCbnBlNTFvdThqNEVNeElIVVZILzgycFNDbU5lL0ZZcEwzQ0tWbXIyUFkKWStCMEtmblkvUUlEQVFBQm8wSXdRREFPQmdOVkhROEJBZjhFQkFNQ0FxUXdEd1lEVlIwVEFRSC9CQVV3QXdFQgovekFkQmdOVkhRNEVGZ1FVWklBVjBROHlaTUV1S2ZrcWdubmdSKzM0ZHl3d0RRWUpLb1pJaHZjTkFRRUxCUUFECmdnRUJBRDA2cE1sZUNMRDRMSmttbzNVdWpvamlyM2IvbzlKbFFUTVNFNFVTQlNLWGJzeS9WOUF3c2ZlV0M3TUYKdUd3MjBWOXFza1dKSVNyWE9DQTM4OGsvcVM0LzFyTEI5RGY4OUdLS09QbzdpbUFRVUkvWGk3bUh0RWlWb1FWbQoreGtkWjRYbFZLa2tCNE5jUWQxdjJsVW4vV2hCWWxYajczQ1dmZXZuVzVCbTBSdFBHTlVtcy9JVUM1a3VoeVMyCk9tM05ucnpHcVdiRCsvcXZNS25naHZaTmk5VmtCMEd3c3QvRlpyNnBwQWFxS2FJTC9malRIUG42L0tGWmRqdW0Kb2UxYXhCQy9LZkVOQzlJY1NCM1JXNjBUaU9EV0xGdzNEUnpMMXAxaU5qcXZVY0xxNXpLSFZZMGpCTWhzK2xvaQo5RUpNMnVHTEp4UWdJTytmSVlIZGhoMHRSaEE9Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
    server: https://api.test-multinode.lab.home:6443
  name: test-multinode
contexts:
- context:
    cluster: test-multinode
    user: admin
  name: admin
current-context: admin
preferences: {}
users:
- name: admin
  user:
    client-certificate-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURaekNDQWsrZ0F3SUJBZ0lJZlQzQVJxS1VtM0F3RFFZSktvWklodmNOQVFFTEJRQXdOakVTTUJBR0ExVUUKQ3hNSmIzQmxibk5vYVdaME1TQXdIZ1lEVlFRREV4ZGhaRzFwYmkxcmRXSmxZMjl1Wm1sbkxYTnBaMjVsY2pBZQpGdzB5TlRBek1EVXhOekl6TlRGYUZ3MHpOVEF6TURNeE56SXpOVEZhTURBeEZ6QVZCZ05WQkFvVERuTjVjM1JsCmJUcHRZWE4wWlhKek1SVXdFd1lEVlFRREV3eHplWE4wWlcwNllXUnRhVzR3Z2dFaU1BMEdDU3FHU0liM0RRRUIKQVFVQUE0SUJEd0F3Z2dFS0FvSUJBUURId0RERUM5QzZsNXZxYWlVRFNtMVU3clFwU1AxNTJxVmRBVUkwWVVVTgpZSVFCMHdtb2xRZmhWNmZkRi9xTC8zeTluNExyQ2NJRjFmbWs5c2dRa2FNQlBIL2lXdUphSllLNGJ2bFEvWi9WCkhQRGU2MEZHdHRSQzQ0VC9XdGRQaExTQTRZWWxyMzg0VU4yOWJSdXZ3MHFudGk5M2RCWVlaNjg3dzlrVkpJRE0KSW9maVQ2ZjYxZ2I2QlVqNE9kdjNHL1NZUTdZZG1TK3hUNThrczE4YTN1TU1ySUxHK2pFNFlnUStxMGNkVFZKZgpnM1NVejd6c2R6dDVNeHJuWFlNQnVUS1FHREFvbzlhMEhvRHBRZDd3ZnRPajlaMEhLVWpZWXdiTHd4SkJYd1NUCmVjWGY0b0JJKzZCNjJsNVRlRFh3RStHTHRUTFdPaUIwaXNLaTlYVTFxMlBmQWdNQkFBR2pmekI5TUE0R0ExVWQKRHdFQi93UUVBd0lGb0RBZEJnTlZIU1VFRmpBVUJnZ3JCZ0VGQlFjREFRWUlLd1lCQlFVSEF3SXdEQVlEVlIwVApBUUgvQkFJd0FEQWRCZ05WSFE0RUZnUVVBMG0zV0M3SGpsYjkvQVRYSGxiOXROZTJzTm93SHdZRFZSMGpCQmd3CkZvQVVuUEo3c1dBS015Q29obENxc29vN1E4ZUZJLzB3RFFZSktvWklodmNOQVFFTEJRQURnZ0VCQUordzh1OEwKWmREbzl2ZStCN0trSCtmL0FLNy9SZDJEbmo3TVdiSjY2Rk1RelZqV0hONllCeUJEdHFlZ2c4NktiQmV3QzBDSAphSTVHQ1Q5NzFOQlU1WllUai95N2hOZzdiZEtOQjJXWE9jY3dVM0FwOUhxZDVTMExBQmk4alNEVTBRTUdWT1lzCm02TXV1QThCQUl0YktvZlM3UDdBUGJLZlc0OWVUWU9nTDZBZnpBTjFyZWhxL1lwclpqZE1MUW5Hb3MrdjYyVE0KUy8wSGMzN3hva0htNGMyckFwazE0djFKK0h3eDUxOEE1ZElrK3VzVk1CdllUZXJWTXhYcGlPNkowK2VOeGlqYgpMMDMwRmdXTWdhWi9WdEgzeW9WLzNhbGsxNDdCQnkrdzVGN1pWN2pYcEFoMUs3WmZOeTNZQy9zRnY5cXd6RHBtCkZxN2FFR0k0d0N1c1lKVT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=
    client-key-data: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFcFFJQkFBS0NBUUVBeDhBd3hBdlF1cGViNm1vbEEwcHRWTzYwS1VqOWVkcWxYUUZDTkdGRkRXQ0VBZE1KCnFKVUg0VmVuM1JmNmkvOTh2WitDNnduQ0JkWDVwUGJJRUpHakFUeC80bHJpV2lXQ3VHNzVVUDJmMVJ6dzN1dEIKUnJiVVF1T0UvMXJYVDRTMGdPR0dKYTkvT0ZEZHZXMGJyOE5LcDdZdmQzUVdHR2V2TzhQWkZTU0F6Q0tINGsrbgordFlHK2dWSStEbmI5eHYwbUVPMkhaa3ZzVStmSkxOZkd0N2pES3lDeHZveE9HSUVQcXRISFUxU1g0TjBsTSs4CjdIYzdlVE1hNTEyREFia3lrQmd3S0tQV3RCNkE2VUhlOEg3VG8vV2RCeWxJMkdNR3k4TVNRVjhFazNuRjMrS0EKU1B1Z2V0cGVVM2cxOEJQaGk3VXkxam9nZElyQ292VjFOYXRqM3dJREFRQUJBb0lCQVFDREFoczV5V0lKcnp2VQpYKytNbS9qZkZudlZCQWt6TFdMOWY4RFRKK1NwSkY4UDcwRExiNHN1a1ZZSVhSeTNTMGFkKzR0YTZoaDF5V1FsCmZMRzBwRUFicEhsZmxTb1Y0N283aXBVOE9Fdm04MGRMZlZKZnRiTzdkd3VZaXhUaUUzQnJndjUvb3YyMml0c1QKelFhMm5VaE9mTi9lNGFWSU5tQ291d2VhcFVsUUdEOHBFK0pHZjcyYUZwaU5uazZDRTc3SWh3RmYzMUdLeGVoUgp1dSt1OUR0RGdwakRGOTBjdGR5akgvNnE5c1BXaDNjaXdEQ0RrS0dFL0ZlR0MvYUQvUEYyajVBWVpKYnZ0aW00ClkvcEIvbzYwUnhXbjRZVlQ4WDh1L1dHaFN6dFMwb3llTGVpaFVWMTE5VGxmbWlmSmJ3RVBrdnhkWHJLWjI1NzkKWi9zbTRhNlJBb0dCQU1uVWRPekFGcXYzdnhSd3pOZWdPZ01tTndoK3dXV3MxMWgxQ3JXSXhMUFptS0JIUWZLRApRWE9IU2g4TGZWWjErbzZ2U2I0dUNvQm93TzdEa2h2SDhOb1k1UVZjaXp0c0FydEpYSy90VlYvWi9vT1I0aVlXCm1rakJKUzZqTWhXYnJrOXNZVUMrNnRkUjBob1BLbzdMdVFsT1lpaUtVWndGdEc2UXhEOVpDeW9wQW9HQkFQMWMKNEZQbnZhVkVjVVltckl3VG83MjhPWS9nc3RHWlJKYzhNb1VTN3dKWS9NbFFRUDFwQXY0U1Z0ODhYc3psSXpYQQpVTGwyQTBJWTd4TklwWHFoVVBWQURIRy8zVnloWDhMa3ZacHJrYWJOUEhxOXRqZUtROHVXanM0a25ocEwzOG1SCkRIWVIxY0k4MStLb1ZCellCU01jNnhsbC9wYi8vZUJJaWhxZTRHN0hBb0dCQUo5VEp3WXAwUHZwOUI1WHVXelMKWUZsU0ZvbVBQbTVjRmhjUE5lZitVb0ZEV2JmVTZKdGZ3QkJLRVZvV2dOZjdCRk1VenRyaGo0cTBwdkVVMDhjNAplOG8vY3JOYnpkR1h2MFJIY25LeW9QMnNvYjBOTVlBdHdaZURXUzNLeUdQRVpNTHY1SW51N0lZVFlnOE9QK00vCnNROUdvRGd1a0tQZzRRR1RLRWgxcTFtWkFvR0FJRVBscFluTGt2Sm1Zb0ttVXFobG45SUttcElJODd4TENkOTIKcDQvRHRFN1UwbVpRQUhXUkZmNEw1aDN3RExQWmlnelZ1dWlXZmFKalA5ZHVpM0ZqdC9mU0hlSkxOSEt3bVVjOApCaUJReWljMDNvU3VZZUJQeGV1RWdDZ1ZvaytyVGlZVXFpeVhSa2N0VHdZVXdCK2FkK0JFNkZVZTJPZjgwc1VxCklUMitZeDBDZ1lFQWllSjhHNTNTL2I4M1N2aDEreFlpZzlKQjlCeE8wNjdCREJhV0tnOFA3VHhUWCttNEdra04KaGR1RElzbXVNMHpTSnZrMFp0bHptbmJUZVdmdzZqZW1uakRNQ2xkWWpZdjN4ZWMrRFB0TVkyME50SjNFUk9paAp3ZmFKbHJXQnpDeGwxNFByZXo4RmRFb1ZOYzcrZ3pQdkNtaTZoM0FaUXhkRlUzV1pTTDZiOGo4PQotLS0tLUVORCBSU0EgUFJJVkFURSBLRVktLS0tLQo=
`)
	})

	It("should return ErrCSRProcessing when CSR is created but not approved", func() {
		newKubeconfig, err := kubecfg.RegenerateClientCertWithCSR(ctx, k8sClient, kubeconfigData, username)
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, kubecfg.ErrCSRProcessing)).To(BeTrue())
		Expect(newKubeconfig).To(BeNil())

		// Verify CSR was created
		csrName := fmt.Sprintf("%s-%s", username, kubecfg.ShortHash(kubeconfigData))
		var csr certificatesv1.CertificateSigningRequest
		err = k8sClient.Get(ctx, client.ObjectKey{Name: csrName}, &csr)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should return ErrCSRProcessing when CSR is approved but certificate not generated", func() {
		// First call to create CSR
		_, err := kubecfg.RegenerateClientCertWithCSR(ctx, k8sClient, kubeconfigData, username)
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, kubecfg.ErrCSRProcessing)).To(BeTrue())

		// Auto-approve CSR
		csrName := fmt.Sprintf("%s-%s", username, kubecfg.ShortHash(kubeconfigData))
		var csr certificatesv1.CertificateSigningRequest
		Expect(k8sClient.Get(ctx, client.ObjectKey{Name: csrName}, &csr)).To(Succeed())

		csr.Status.Conditions = []certificatesv1.CertificateSigningRequestCondition{
			{
				Type:   certificatesv1.CertificateApproved,
				Status: corev1.ConditionTrue,
			},
		}
		Expect(k8sClient.Status().Update(ctx, &csr)).To(Succeed())

		// Second call should still return ErrCSRProcessing
		newKubeconfig, err := kubecfg.RegenerateClientCertWithCSR(ctx, k8sClient, kubeconfigData, username)
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, kubecfg.ErrCSRProcessing)).To(BeTrue())
		Expect(newKubeconfig).To(BeNil())
	})

	It("should return new kubeconfig when CSR is approved and certificate is generated", func() {
		// First call to create CSR
		_, err := kubecfg.RegenerateClientCertWithCSR(ctx, k8sClient, kubeconfigData, username)
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, kubecfg.ErrCSRProcessing)).To(BeTrue())

		// Auto-approve CSR and add certificate
		csrName := fmt.Sprintf("%s-%s", username, kubecfg.ShortHash(kubeconfigData))
		var csr certificatesv1.CertificateSigningRequest
		Expect(k8sClient.Get(ctx, client.ObjectKey{Name: csrName}, &csr)).To(Succeed())

		csr.Status.Conditions = []certificatesv1.CertificateSigningRequestCondition{
			{
				Type:   certificatesv1.CertificateApproved,
				Status: corev1.ConditionTrue,
			},
		}
		csr.Status.Certificate = []byte("test-certificate-data")
		Expect(k8sClient.Status().Update(ctx, &csr)).To(Succeed())

		// Second call should return new kubeconfig
		newKubeconfig, err := kubecfg.RegenerateClientCertWithCSR(ctx, k8sClient, kubeconfigData, username)
		Expect(err).NotTo(HaveOccurred())
		Expect(newKubeconfig).NotTo(BeNil())

		// Verify new kubeconfig
		config, err := clientcmd.Load(newKubeconfig)
		Expect(err).NotTo(HaveOccurred())
		Expect(config.AuthInfos[username].ClientCertificateData).To(Equal([]byte("test-certificate-data")))
		Expect(config.AuthInfos[username].ClientKeyData).NotTo(BeEmpty())
	})

	It("should fail for non-existent user", func() {
		newKubeconfig, err := kubecfg.RegenerateClientCertWithCSR(ctx, k8sClient, kubeconfigData, "nonexistent")
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError("user nonexistent not present in kubeconfig"))
		Expect(newKubeconfig).To(BeNil())
	})

	It("should fail for invalid kubeconfig", func() {
		newKubeconfig, err := kubecfg.RegenerateClientCertWithCSR(ctx, k8sClient, []byte("invalid"), username)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("error loading kubeconfig"))
		Expect(newKubeconfig).To(BeNil())
	})
})
