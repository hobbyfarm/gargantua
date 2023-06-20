package preinstall

import (
	"context"
	"github.com/golang/glog"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	"github.com/hobbyfarm/gargantua/pkg/preinstall/settings"
	"sync"
)

func Preinstall(ctx context.Context, client *hfClientset.Clientset) {
	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		err := settings.InstallResources(ctx, client, &wg)
		if err != nil {
			glog.Errorf("error installing resources: %s", err.Error())
		}
	}()

	wg.Wait()
}
