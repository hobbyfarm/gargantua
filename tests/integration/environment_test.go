package integration

import (
	"context"
	"fmt"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	"github.com/hobbyfarm/gargantua/tests/framework/setup"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Install Environment", func() {

	var hf *hfClientset.Clientset
	var err error

	BeforeEach(func() {
		Eventually(func() error {
			return setup.SetupCommonObjects(context.TODO(), config)
		}, 5, 60).ShouldNot(HaveOccurred())
	})

	AfterEach(func() {
		Eventually(func() error {
			return setup.CleanupCommonObjects(context.TODO(), config)
		}, 5, 60).ShouldNot(HaveOccurred())
	})

	It("Query the environment", func() {
		hf, err = hfClientset.NewForConfig(config)
		Expect(err).NotTo(HaveOccurred())
		Expect(hf).ToNot(BeNil())

		Eventually(func() error {
			envList, err := hf.HobbyfarmV1().Environments(setup.DefaultNamespace).List(ctx, metav1.ListOptions{})
			if err != nil {
				return err
			}
			if len(envList.Items) == 0 {
				return fmt.Errorf("no envs found yet")
			}

			logrus.Infof("found %d environments", len(envList.Items))
			return nil
		}, 5, 60).ShouldNot(HaveOccurred())
	})
})
