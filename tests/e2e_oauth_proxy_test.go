package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

const (
	oauthProxyURL   = "http://127.0.0.1:19418"
	dexServiceIP    = "10.53.0.60"
	dexImage        = "ghcr.io/dexidp/dex@sha256:bc7cfce7c17f52864e2bb2a4dc1d2f86a41e3019f6d42e81d92a301fad0c8a1d"
	oauthProxyImage = "quay.io/oauth2-proxy/oauth2-proxy@sha256:d62e2d81c6f5048f652f67c302083be1272c181b971fad80e5a30ebe2b8b75d8"
)

func createOAuthProxyResources(ctx context.Context, client kubernetes.Interface, cfg e2eConfig) error {
	namespace := cfg.systemNamespace()
	labels := map[string]string{"app": "keel-oauth-e2e", e2eRunLabel: cfg.runID}
	dexConfig := fmt.Sprintf(`issuer: http://%s:5556/dex
storage:
  type: memory
web:
  http: 0.0.0.0:5556
oauth2:
  skipApprovalScreen: true
staticClients:
- id: keel-e2e
  name: Keel e2e
  secret: keel-e2e-secret
  redirectURIs:
  - http://127.0.0.1:19418/oauth2/callback
enablePasswordDB: true
staticPasswords:
- email: alice@example.test
  hash: '$2a$10$NMYoKasT1XS0eJjSEWD5e.k4lArOO7boaygjnFqk07WdM4koH5DLq'
  username: alice
  userID: 00000000-0000-0000-0000-000000000001
`, dexServiceIP)

	if _, err := client.CoreV1().ConfigMaps(namespace).Create(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "dex", Namespace: namespace, Labels: labels},
		Data:       map[string]string{"config.yaml": dexConfig},
	}, metav1.CreateOptions{}); err != nil {
		return err
	}
	if _, err := client.AppsV1().Deployments(namespace).Create(ctx, dexDeployment(cfg, labels), metav1.CreateOptions{}); err != nil {
		return err
	}
	if _, err := client.CoreV1().Services(namespace).Create(ctx, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "dex", Namespace: namespace, Labels: labels},
		Spec:       corev1.ServiceSpec{ClusterIP: dexServiceIP, Selector: map[string]string{"app": "dex-e2e", e2eRunLabel: cfg.runID}, Ports: []corev1.ServicePort{{Name: "http", Port: 5556}}},
	}, metav1.CreateOptions{}); err != nil {
		return err
	}
	if _, err := client.AppsV1().Deployments(namespace).Create(ctx, oauthKeelDeployment(cfg, labels), metav1.CreateOptions{}); err != nil {
		return err
	}
	_, err := client.CoreV1().Services(namespace).Create(ctx, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "keel-oauth", Namespace: namespace, Labels: labels},
		Spec:       corev1.ServiceSpec{Selector: map[string]string{"app": "keel-oauth-e2e", e2eRunLabel: cfg.runID}, Ports: []corev1.ServicePort{{Name: "http", Port: 4180, TargetPort: intstr.FromString("oauth")}}},
	}, metav1.CreateOptions{})
	return err
}

func dexDeployment(cfg e2eConfig, labels map[string]string) *appsv1.Deployment {
	podLabels := map[string]string{"app": "dex-e2e", e2eRunLabel: cfg.runID}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "dex", Namespace: cfg.systemNamespace(), Labels: labels},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptrTo(int32(1)),
			Selector: &metav1.LabelSelector{MatchLabels: podLabels},
			Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: podLabels}, Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name: "dex", Image: dexImage, Args: []string{"dex", "serve", "/etc/dex/config.yaml"},
					Ports:          []corev1.ContainerPort{{Name: "http", ContainerPort: 5556}},
					VolumeMounts:   []corev1.VolumeMount{{Name: "config", MountPath: "/etc/dex", ReadOnly: true}},
					ReadinessProbe: &corev1.Probe{ProbeHandler: corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{Path: "/dex/.well-known/openid-configuration", Port: intstr.FromString("http")}}, PeriodSeconds: 1, FailureThreshold: 60},
				}},
				Volumes: []corev1.Volume{{Name: "config", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "dex"}}}}},
			}},
		},
	}
}

func oauthKeelDeployment(cfg e2eConfig, labels map[string]string) *appsv1.Deployment {
	podLabels := map[string]string{"app": "keel-oauth-e2e", e2eRunLabel: cfg.runID}
	resources := corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("10m"), corev1.ResourceMemory: resource.MustParse("32Mi")}}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "keel-oauth", Namespace: cfg.systemNamespace(), Labels: labels},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptrTo(int32(1)), Selector: &metav1.LabelSelector{MatchLabels: podLabels},
			Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: podLabels}, Spec: corev1.PodSpec{ServiceAccountName: "keel", Containers: []corev1.Container{
				{Name: "keel", Image: cfg.keelImage, ImagePullPolicy: corev1.PullIfNotPresent, Env: []corev1.EnvVar{{Name: "AUTH_MODE", Value: "external-proxy"}, {Name: "AUTH_PROXY_USER_HEADER", Value: "X-Forwarded-User"}, {Name: "POLL", Value: "false"}}, Ports: []corev1.ContainerPort{{Name: "keel", ContainerPort: 9300}}, Resources: resources},
				{Name: "oauth2-proxy", Image: oauthProxyImage, Args: []string{
					"--http-address=0.0.0.0:4180", "--upstream=http://127.0.0.1:9300", "--provider=oidc", "--oidc-issuer-url=http://" + dexServiceIP + ":5556/dex",
					"--client-id=keel-e2e", "--client-secret=keel-e2e-secret", "--redirect-url=" + oauthProxyURL + "/oauth2/callback", "--cookie-secret=0123456789abcdef0123456789abcdef",
					"--cookie-secure=false", "--email-domain=*", "--scope=openid profile email", "--skip-provider-button=true", "--code-challenge-method=S256", "--reverse-proxy=true", "--set-xauthrequest=true", "--pass-user-headers=true",
					"--prefer-email-to-user=true",
				}, Ports: []corev1.ContainerPort{{Name: "oauth", ContainerPort: 4180}}, ReadinessProbe: &corev1.Probe{ProbeHandler: corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{Path: "/ready", Port: intstr.FromString("oauth")}}, PeriodSeconds: 1, FailureThreshold: 60}, Resources: resources},
			}}},
		},
	}
}

func (s *E2ESuite) TestExternalOAuthProxyAdminFlow() {
	noRedirect := &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	req, err := http.NewRequest(http.MethodGet, oauthProxyURL+"/v1/resources", nil)
	s.Require().NoError(err)
	req.Header.Set("X-Forwarded-User", "spoofed@example.test")
	response, err := noRedirect.Do(req)
	s.Require().NoError(err)
	_ = response.Body.Close()
	s.Require().Equal(http.StatusFound, response.StatusCode, "unauthenticated spoofed identity must be redirected")
	s.Require().NotEmpty(response.Header.Get("Location"), "unauthenticated request must be redirected into the OIDC flow")

	pods, err := s.client.CoreV1().Pods(s.cfg.systemNamespace()).List(context.Background(), metav1.ListOptions{LabelSelector: "app=keel-oauth-e2e"})
	s.Require().NoError(err)
	s.Require().Len(pods.Items, 1)
	connection, err := net.DialTimeout("tcp", net.JoinHostPort(pods.Items[0].Status.PodIP, "9300"), time.Second)
	if err == nil {
		_ = connection.Close()
	}
	s.Require().Error(err, "Keel port 9300 must not accept direct Pod-IP connections")

	jar, err := cookiejar.New(nil)
	s.Require().NoError(err)
	client := &http.Client{Timeout: 15 * time.Second, Jar: jar}
	response, err = client.Get(oauthProxyURL + "/")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, response.StatusCode)
	loginPage, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	s.Require().NoError(err)
	action, fields, err := dexLoginForm(loginPage, response.Request.URL)
	s.Require().NoError(err)
	fields.Set("login", "alice@example.test")
	fields.Set("password", "password")
	response, err = client.PostForm(action, fields)
	s.Require().NoError(err)
	index, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	s.Require().NoError(err)
	if !strings.Contains(string(index), `<div id="root"></div>`) {
		approvalAction, approvalFields, formErr := dexLoginForm(index, response.Request.URL)
		s.Require().NoError(formErr, "expected Dex approval form after password login")
		response, err = client.PostForm(approvalAction, approvalFields)
		s.Require().NoError(err)
		index, err = io.ReadAll(response.Body)
		_ = response.Body.Close()
		s.Require().NoError(err)
	}
	s.Require().Equal(http.StatusOK, response.StatusCode)
	s.Require().Contains(string(index), `<div id="root"></div>`, "authenticated Admin UI index must load")

	response, err = client.Get(oauthProxyURL + "/v1/auth/user")
	s.Require().NoError(err)
	var user struct {
		Name     string `json:"name"`
		AuthMode string `json:"auth_mode"`
	}
	s.Require().NoError(json.NewDecoder(response.Body).Decode(&user))
	_ = response.Body.Close()
	s.Require().Equal("alice@example.test", user.Name)
	s.Require().Equal("external-proxy", user.AuthMode)

	response, err = client.Get(oauthProxyURL + "/v1/resources")
	s.Require().NoError(err)
	resourcesBody, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, response.StatusCode)
	s.Require().NotEmpty(resourcesBody, "meaningful authenticated Admin API read must return JSON")

	response, err = client.Post(oauthProxyURL+"/v1/auth/login", "application/json", strings.NewReader(`{"username":"admin","password":"secret"}`))
	s.Require().NoError(err)
	_ = response.Body.Close()
	s.Require().Equal(http.StatusNotFound, response.StatusCode, "local Keel login must remain disabled")

	response, err = client.Get(oauthProxyURL + "/oauth2/sign_out?rd=/")
	s.Require().NoError(err)
	_ = response.Body.Close()
	response, err = noRedirect.Get(oauthProxyURL + "/v1/resources")
	s.Require().NoError(err)
	_ = response.Body.Close()
	s.Require().Equal(http.StatusFound, response.StatusCode, "logged-out session must be protected")
}

func dexLoginForm(document []byte, base *url.URL) (string, url.Values, error) {
	root, err := html.Parse(strings.NewReader(string(document)))
	if err != nil {
		return "", nil, err
	}
	var form *html.Node
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if form != nil {
			return
		}
		if node.Type == html.ElementNode && node.Data == "form" {
			form = node
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
	if form == nil {
		return "", nil, fmt.Errorf("Dex login form not found in %s", base)
	}
	action := base.String()
	fields := url.Values{}
	for _, attr := range form.Attr {
		if attr.Key == "action" {
			parsed, err := base.Parse(attr.Val)
			if err != nil {
				return "", nil, err
			}
			action = parsed.String()
		}
	}
	var inputs func(*html.Node)
	inputs = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "input" {
			name, value := "", ""
			for _, attr := range node.Attr {
				if attr.Key == "name" {
					name = attr.Val
				}
				if attr.Key == "value" {
					value = attr.Val
				}
			}
			if name != "" {
				fields.Set(name, value)
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			inputs(child)
		}
	}
	inputs(form)
	return action, fields, nil
}
