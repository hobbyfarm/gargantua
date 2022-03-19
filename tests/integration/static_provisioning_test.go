package integration

import (
	"context"
	"encoding/json"
	"fmt"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	"github.com/hobbyfarm/gargantua/pkg/controllers/scheduledevent"
	"github.com/hobbyfarm/gargantua/pkg/vmclaimserver"
	"github.com/hobbyfarm/gargantua/tests/framework/api"
	"github.com/hobbyfarm/gargantua/tests/framework/setup"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

var _ = Describe("Static Provisioning Testing", func() {

	var hf *hfClientset.Clientset
	var err error
	var user *hfv1.User
	vmRequested := 2
	sessionSpec := &hfv1.SessionSpec{}
	// default endpoint for API calls
	defaultAddress := fmt.Sprintf("http://localhost:%d", DefaultPort)

	// required VMs for scheduled event
	vmCount := make(map[string]int)
	vmCount["sles-15-sp2-static"] = vmRequested
	envVMMap := make(map[string]map[string]int)
	envVMMap["aws-demo-static"] = vmCount

	staticEvent := &hfv1.ScheduledEvent{
		ObjectMeta: metav1.ObjectMeta{
			Name: "static-event",
		},
		Spec: hfv1.ScheduledEventSpec{
			AccessCode:              "static",
			Creator:                 "admin",
			Description:             "static event provisioning",
			OnDemand:                false,
			RestrictedBind:          true,
			RestrictedBindValue:     "static-event",
			Scenarios:               []string{"test-scenario-static"},
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
			err = setup.SetupCommonObjects(context.TODO(), config, "static")
			if err != nil {
				return err
			}

			hf, err = hfClientset.NewForConfig(config)
			if err != nil {
				return err
			}

			_, err = hf.HobbyfarmV1().ScheduledEvents(setup.DefaultNamespace).Create(context.TODO(), staticEvent,
				metav1.CreateOptions{})
			if err != nil {
				return err
			}

			//setup a new user for testing out scenarios
			err = api.RegisterUser("static", "static", "static", defaultAddress)
			return err
		}, defaultTimeout, defaultInterval).ShouldNot(HaveOccurred())
	})

	AfterEach(func() {
		Eventually(func() error {
			err = setup.CleanupCommonObjects(context.TODO(), config, "dynamic")
			if err != nil {
				return err
			}

			err = hf.HobbyfarmV1().ScheduledEvents(setup.DefaultNamespace).Delete(context.TODO(), staticEvent.Name,
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

	It("Setup Static Scheduled Event", func() {

		By("Create ScheduledEvent And Check AccessCodes")
		{
			Eventually(func() error {
				se, err := hf.HobbyfarmV1().ScheduledEvents(setup.DefaultNamespace).Get(ctx, staticEvent.Name,
					metav1.GetOptions{})
				if err != nil {
					return err
				}

				if !se.Status.Active {
					return fmt.Errorf("waiting for scheduled event to be made active")
				}

				code, err := hf.HobbyfarmV1().AccessCodes(setup.DefaultNamespace).Get(ctx, staticEvent.Spec.AccessCode,
					metav1.GetOptions{})
				if err != nil {
					return err
				}

				if code.Spec.Code != staticEvent.Spec.AccessCode {
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
					if u.Spec.Email == "static" {
						found = true
					}
				}

				if !found {
					return fmt.Errorf("user with email address static not found")
				}

				return nil
			}, defaultTimeout, defaultInterval).ShouldNot(HaveOccurred())
		}

		By("Query VMSet Object for ScheduledEvent")
		{
			Eventually(func() error {
				vmset, err := hf.HobbyfarmV1().VirtualMachineSets(setup.DefaultNamespace).List(context.TODO(),
					metav1.ListOptions{LabelSelector: scheduledevent.ScheduledEventLabel + "=" + staticEvent.Name})
				if err != nil {
					return err
				}
				if len(vmset.Items) != 1 {
					return fmt.Errorf("vmset list should have had 1 vmset but found %d", len(vmset.Items))
				}
				return nil
			}, defaultTimeout, defaultInterval).ShouldNot(HaveOccurred())
		}

		By("Query VM's have been created and are running")
		{
			Eventually(func() error {
				vmList, err := hf.HobbyfarmV1().VirtualMachines(setup.DefaultNamespace).List(context.TODO(),
					metav1.ListOptions{LabelSelector: scheduledevent.ScheduledEventLabel + "=" + staticEvent.Name})
				if err != nil {
					return err
				}
				if len(vmList.Items) != vmRequested {
					return fmt.Errorf("Should have found %d VMs but found %d", vmRequested, len(vmList.Items))
				}

				runningCount := 0
				for _, vm := range vmList.Items {
					if vm.Status.Status == hfv1.VmStatusRunning {
						runningCount++
					}
				}

				if runningCount != vmRequested {
					return fmt.Errorf("Should have found %d VMs running but only %d are", vmRequested, runningCount)
				}
				return nil
			}, 5*defaultTimeout, defaultInterval).ShouldNot(HaveOccurred())
		}

		By("Create a session for static event")
		{
			Eventually(func() error {
				g, err := api.NewGargClient("static", "static", defaultAddress)
				if err != nil {
					return err
				}
				resp, err := g.StartScenario("test-scenario-static", "static")
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
				g, err := api.NewGargClient("static", "static", defaultAddress)
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
