package integration

import (
	"context"
	"encoding/json"
	"fmt"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	"github.com/hobbyfarm/gargantua/pkg/vmclaimserver"
	"github.com/hobbyfarm/gargantua/tests/framework/api"
	"github.com/hobbyfarm/gargantua/tests/framework/setup"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

var _ = Describe("Dynamic Provisioning Testing", func() {

	var hf *hfClientset.Clientset
	var err error
	var user *hfv1.User
	sessionSpec := &hfv1.SessionSpec{}
	// default endpoint for API calls
	defaultAddress := fmt.Sprintf("http://localhost:%d", DefaultPort)

	// required VMs for scheduled event
	vmCount := make(map[string]int)
	vmCount["sles-15-sp2-dynamic"] = 2
	envVMMap := make(map[string]map[string]int)
	envVMMap["aws-demo-dynamic"] = vmCount

	dynamicEvent := &hfv1.ScheduledEvent{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dynamic-event",
		},
		Spec: hfv1.ScheduledEventSpec{
			AccessCode:              "dynamic",
			Creator:                 "admin",
			Description:             "dynamic event provisioning",
			OnDemand:                true,
			RestrictedBind:          true,
			RestrictedBindValue:     "dynamic-event",
			Scenarios:               []string{"test-scenario-dynamic"},
			RequiredVirtualMachines: envVMMap,
			StartTime:               time.Now().UTC().Format(time.UnixDate),
			EndTime:                 time.Now().UTC().Add(10 * time.Minute).Format(time.UnixDate),
		},
		Status: hfv1.ScheduledEventStatus{
			Active: true,
		},
	}

	BeforeEach(func() {
		Eventually(func() error {
			err = setup.SetupCommonObjects(context.TODO(), config, "dynamic")
			if err != nil {
				return err
			}

			hf, err = hfClientset.NewForConfig(config)
			if err != nil {
				return err
			}

			_, err = hf.HobbyfarmV1().ScheduledEvents(setup.DefaultNamespace).Create(context.TODO(), dynamicEvent,
				metav1.CreateOptions{})
			if err != nil {
				return err
			}

			//setup a new user for testing out scenarios
			err = api.RegisterUser("dynamic", "dynamic", "dynamic", defaultAddress)
			return err
		}, defaultTimeout, defaultInterval).ShouldNot(HaveOccurred())
	})

	AfterEach(func() {
		Eventually(func() error {
			err = setup.CleanupCommonObjects(context.TODO(), config, "dynamic")
			if err != nil {
				return err
			}

			err = hf.HobbyfarmV1().ScheduledEvents(setup.DefaultNamespace).Delete(context.TODO(), dynamicEvent.Name,
				metav1.DeleteOptions{})
			if err != nil {
				return err
			}

			if user != nil {
				err = hf.HobbyfarmV1().Users(setup.DefaultNamespace).Delete(context.TODO(), user.Name, metav1.DeleteOptions{})
			}

			return err
		}, defaultTimeout, defaultInterval).ShouldNot(HaveOccurred())
	})

	It("Setup Dynamic Scheduled Event", func() {

		By("Create ScheduledEvent And Check AccessCodes")
		{
			Eventually(func() error {
				se, err := hf.HobbyfarmV1().ScheduledEvents(setup.DefaultNamespace).Get(ctx, dynamicEvent.Name,
					metav1.GetOptions{})
				if err != nil {
					return err
				}

				if !se.Status.Active {
					return fmt.Errorf("waiting for scheduled event to be made active")
				}

				code, err := hf.HobbyfarmV1().AccessCodes(setup.DefaultNamespace).Get(ctx, dynamicEvent.Spec.AccessCode,
					metav1.GetOptions{})
				if err != nil {
					return err
				}

				if code.Spec.Code != dynamicEvent.Spec.AccessCode {
					return fmt.Errorf("access code spec does not match dynamic event access code")
				}

				return nil
			}, defaultTimeout, defaultInterval).ShouldNot(HaveOccurred())
		}

		By("Register a user and check creation was successful for static event")
		{
			Eventually(func() error {
				var found bool
				userList, err := hf.HobbyfarmV1().Users(setup.DefaultNamespace).List(context.TODO(), metav1.ListOptions{})
				if err != nil {
					return err
				}

				for _, u := range userList.Items {
					if u.Spec.Email == "dynamic" {
						found = true
					}
				}

				if !found {
					return fmt.Errorf("user with email address dynamic not found")
				}

				return nil
			}, defaultTimeout, defaultInterval).ShouldNot(HaveOccurred())
		}

		By("Create a session for dynamic event")
		{
			Eventually(func() error {
				g, err := api.NewGargClient("dynamic", "dynamic", defaultAddress)
				if err != nil {
					return err
				}
				resp, err := g.StartScenario("test-scenario-dynamic", "dynamic")
				if err != nil {
					return err
				}

				err = json.Unmarshal(resp, sessionSpec)
				if err != nil {
					logrus.Errorf("unable to process response %s", string(resp))
					return err
				}

				if len(sessionSpec.VmClaimSet) == 0 {
					return fmt.Errorf("no vmclaims returned in response")
				}

				return nil
			}, defaultTimeout, defaultInterval).ShouldNot(HaveOccurred())
		}

		By("Query VMClaims until they are ready using CRD")
		{
			Eventually(func() error {
				for _, vmc := range sessionSpec.VmClaimSet {
					v, err := hf.HobbyfarmV1().VirtualMachineClaims(setup.DefaultNamespace).Get(context.TODO(), vmc, metav1.GetOptions{})
					if err != nil {
						return err
					}
					// we wait for VMC's to be ready
					if !v.Status.Ready {
						return fmt.Errorf("VirtualMachineClaim is not yet ready.. will check again shortly")
					}
				}
				return nil
			}, defaultTimeout, defaultInterval).ShouldNot(HaveOccurred())
		}

		By("Query VMClaims until they are ready using api")
		{
			Eventually(func() error {
				g, err := api.NewGargClient("dynamic", "dynamic", defaultAddress)
				if err != nil {
					return err
				}

				for _, vmc := range sessionSpec.VmClaimSet {
					resp, err := g.FindVMClaim(vmc)
					if err != nil {
						return err
					}
					vmcSpec := &vmclaimserver.PreparedVirtualMachineClaim{}
					err = json.Unmarshal(resp, vmcSpec)
					if err != nil {
						return err
					}

					if !vmcSpec.Ready {
						return fmt.Errorf("PreparedVirtualMachineClaim is not yet ready.. will check again shortly")
					}
				}
				return nil
			}, defaultTimeout, defaultInterval).ShouldNot(HaveOccurred())
		}
	})

})
