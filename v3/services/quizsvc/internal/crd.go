package quizservice

import (
	"github.com/ebauman/crder"
	v1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/v3/pkg/crd"
	"github.com/hobbyfarm/gargantua/v3/pkg/rbac"
)

// QuizCRDInstaller is a struct that can generate CRDs for quizes and quiz progresses.
// It implements the CrdInstaller interface defined in "github.com/hobbyfarm/gargantua/v3/pkg/microservices"
type QuizCRDInstaller struct{}

func (qi QuizCRDInstaller) GenerateCRDs() []crder.CRD {
	return []crder.CRD{
		crd.HobbyfarmCRD(&v1.Quiz{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v1", &v1.Quiz{}, nil).
				WithNames("quiz", rbac.ResourcePluralQuiz)
		}),
	}
}
