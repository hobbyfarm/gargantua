package integration

import (
	"context"
	"github.com/hobbyfarm/gargantua/pkg/bootstrap"
	_ "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned/scheme"
	"github.com/hobbyfarm/gargantua/pkg/crd"
	"github.com/hobbyfarm/gargantua/pkg/signals"
	"github.com/hobbyfarm/gargantua/tests/framework/cluster"
	"github.com/hobbyfarm/gargantua/tests/framework/setup"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	_ "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"testing"
	"time"
)

var (
	config *rest.Config
	ctx    context.Context
	cancel context.CancelFunc
)

const (
	timeout      = time.Second * 10
	duration     = time.Second * 10
	setupTimeout = 1200
)

func TestAPI(t *testing.T) {
	defer GinkgoRecover()
	RegisterFailHandler(Fail)
	RunSpecs(t, "gargantua integration")
}

var _ = BeforeSuite(func(done Done) {
	defer close(done)
	var err error
	By("starting test cluster")
	ctx, cancel = context.WithCancel(context.TODO())
	c, err := cluster.Setup(ctx)
	Expect(err).NotTo(HaveOccurred())
	Expect(c).NotTo(BeNil())
	config, err = c.Startup(ctx)
	Expect(err).NotTo(HaveOccurred())
	Expect(config).NotTo(BeNil())

	k8sClient, err := kubernetes.NewForConfig(config)
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	logrus.Info("waiting for nodes to be ready")
	Eventually(func() bool {
		nodeList, err := k8sClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if err != nil {
			logrus.Error(err)
			return false
		}

		var ready bool
		for _, node := range nodeList.Items {
			logrus.Infof("querying node %s", node.Name)
			for _, condition := range node.Status.Conditions {
				if condition.Type == corev1.NodeReady {
					logrus.Tracef("current node status is %s", condition.Status)
					if condition.Status == corev1.ConditionTrue {
						ready = true
					}
				}
			}
		}

		return ready
	}, 30*time.Second, 5*time.Second).Should(BeTrue())

	logrus.Info("Setting up hobbyfarm namespace")
	Eventually(func() error {
		_, err = k8sClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: setup.DefaultNamespace,
			},
		}, metav1.CreateOptions{})
		return err
	}, 30*time.Second, 5*time.Second).ShouldNot(HaveOccurred())

	logrus.Info("Installing CRDs")
	Eventually(func() error {
		err = crd.Create(context.Background(), config)
		return err
	}).ShouldNot(HaveOccurred())
	stopCh := signals.SetupSignalHandler()

	go func() {
		defer GinkgoRecover()
		g := bootstrap.NewServer(config, false, false, false, 8080, setup.DefaultNamespace)

		g.Start(context.Background(), stopCh)
	}()

}, setupTimeout)

var _ = AfterSuite(func(done Done) {
	defer close(done)
	cancel()
	var err error
	By("shutting down test cluster")
	c, err := cluster.Setup(context.Background())
	Expect(err).NotTo(HaveOccurred())
	Expect(c).NotTo(BeNil())
	err = c.Shutdown(context.Background())
	Expect(err).NotTo(HaveOccurred())
}, setupTimeout)
