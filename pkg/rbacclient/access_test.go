package rbacclient

import (
	"fmt"
	"testing"
)

func Test_Grants(t *testing.T) {
	accessSet := AccessSet{
		Subject: "fake@fake.com",
		Access: map[string]bool{
			"/hobbyfarm.io/scheduledevents/list": true,
			"/hobbyfarm.io/scenarios/list":       true,
		},
	}

	sePerms := RbacRequest().HobbyfarmPermission("scheduledevents", "list").GetPermissions()
	sPerms := RbacRequest().HobbyfarmPermission("scenarios", "list").GetPermissions()

	nPerms := RbacRequest().HobbyfarmPermission("courses", "list").GetPermissions()

	for _, p := range append(sePerms, sPerms...) {
		t.Run(fmt.Sprintf("should allow %s/%s/%s", p.GetAPIGroup(), p.GetResource(), p.GetVerb()), func(t *testing.T) {
			if !accessSet.Grants(p) {
				t.Error("accessset should have granted, did not")
			}
		})
	}

	for _, p := range nPerms {
		t.Run(fmt.Sprintf("should not allow %s/%s/%s", p.GetAPIGroup(), p.GetResource(), p.GetVerb()), func(t *testing.T) {
			if accessSet.Grants(p) {
				t.Error("accessset granted, should not have")
			}
		})
	}
}
