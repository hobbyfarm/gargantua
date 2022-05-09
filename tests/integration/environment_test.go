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

var _ = Describe("Setup Common Environment Infra", func() {

	var hf *hfClientset.Clientset
	var err error

	BeforeEach(func() {
		Eventually(func() error {
			return setup.SetupCommonObjects(context.TODO(), config, "environment")
		}, 5, 60).ShouldNot(HaveOccurred())
	})

	AfterEach(func() {
		Eventually(func() error {
			return setup.CleanupCommonObjects(context.TODO(), config, "environment")
		}, defaultTimeout, defaultInterval).ShouldNot(HaveOccurred())
	})

	It("Query the common setup", func() {
		hf, err = hfClientset.NewForConfig(config)
		Expect(err).NotTo(HaveOccurred())
		Expect(hf).ToNot(BeNil())

		By("Check if Environment exists")
		{
			Eventually(func() error {
				envList, err := hf.HobbyfarmV1().Environments(setup.DefaultNamespace).List(ctx, metav1.ListOptions{})
				if err != nil {
					return err
				}
				if len(envList.Items) == 0 {
					return fmt.Errorf("no envs found yet")
				}

				logrus.Tracef("found %d environments", len(envList.Items))
				return nil
			}, defaultTimeout, defaultInterval).ShouldNot(HaveOccurred())
		}

		By("Check if Scenario exists")
		{
			Eventually(func() error {
				scenarioList, err := hf.HobbyfarmV1().Scenarios(setup.DefaultNamespace).List(ctx, metav1.ListOptions{})
				if err != nil {
					return err
				}
				if len(scenarioList.Items) == 0 {
					return fmt.Errorf("no scenarios found yet")
				}

				logrus.Tracef("found %d scenarios", len(scenarioList.Items))
				return nil
			}, defaultTimeout, defaultInterval).ShouldNot(HaveOccurred())
		}

		By("Check if VirtualMachineTemplates exist")
		{
			Eventually(func() error {
				vmtList, err := hf.HobbyfarmV1().VirtualMachineTemplates(setup.DefaultNamespace).List(ctx, metav1.ListOptions{})
				if err != nil {
					return err
				}
				if len(vmtList.Items) == 0 {
					return fmt.Errorf("no virtualmachinetemplates found yet")
				}

				logrus.Tracef("found %d environments", len(vmtList.Items))
				return nil
			}, defaultTimeout, defaultInterval).ShouldNot(HaveOccurred())
		}
	})
})
