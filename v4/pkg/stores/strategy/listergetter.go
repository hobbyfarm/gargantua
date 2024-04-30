package strategy

import "github.com/hobbyfarm/mink/pkg/strategy"

type ListerGetter interface {
	strategy.Lister
	strategy.Getter
}
