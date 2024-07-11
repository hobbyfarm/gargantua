package rbac

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	hflabels "github.com/hobbyfarm/gargantua/v3/pkg/labels"
	"github.com/hobbyfarm/gargantua/v3/pkg/rbac"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	rbacpb "github.com/hobbyfarm/gargantua/v3/protos/rbac"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/emptypb"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

func (rs *GrpcRbacServer) CreateRolebinding(c context.Context, cr *rbacpb.RoleBinding) (*emptypb.Empty, error) {
	rolebinding, err := rs.marshalRolebinding(c, cr)
	if err != nil {
		glog.Errorf("invalid rolebinding: %v", err)
		return &emptypb.Empty{}, hferrors.GrpcError(
			codes.InvalidArgument,
			"invalid rolebinding",
			cr,
		)
	}

	_, err = rs.roleBindingClient.Create(c, rolebinding, metav1.CreateOptions{})
	if err != nil {
		glog.Errorf("error creating rolebinding in kubernetes: %v", err)
		return &emptypb.Empty{}, hferrors.GrpcError(
			codes.Internal,
			"error creating rolebinding",
			cr,
		)
	}

	return &emptypb.Empty{}, nil
}

func (rs *GrpcRbacServer) GetRolebinding(c context.Context, gr *generalpb.GetRequest) (*rbacpb.RoleBinding, error) {
	rolebinding, err := rs.getRolebinding(c, gr)
	if err != nil {
		return nil, err
	}
	return unmarshalRolebinding(rolebinding), nil
}

func (rs *GrpcRbacServer) UpdateRolebinding(c context.Context, ur *rbacpb.RoleBinding) (*emptypb.Empty, error) {
	inputRolebinding, err := rs.marshalRolebinding(c, ur)
	if err != nil {
		glog.Errorf("invalid role: %v", err)
		return &emptypb.Empty{}, hferrors.GrpcError(
			codes.InvalidArgument,
			"invalid role",
			ur,
		)
	}

	k8sRolebinding, err := rs.getRolebinding(c, &generalpb.GetRequest{Id: ur.GetName()})
	if err != nil {
		return &emptypb.Empty{}, err
	}

	k8sRolebinding.RoleRef = inputRolebinding.RoleRef
	k8sRolebinding.Subjects = inputRolebinding.Subjects

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_, err := rs.roleBindingClient.Update(c, k8sRolebinding, metav1.UpdateOptions{})
		if err != nil {
			glog.Errorf("error while updating rolebinding in kubernetes: %v", err)
			return hferrors.GrpcError(
				codes.Internal,
				"error while updating rolebinding",
				ur,
			)
		}
		return nil
	})

	if retryErr != nil {
		return &emptypb.Empty{}, retryErr
	}

	return &emptypb.Empty{}, nil
}

func (rs *GrpcRbacServer) DeleteRolebinding(ctx context.Context, req *generalpb.ResourceId) (*emptypb.Empty, error) {
	// we want to get the rolebinding first as this allows us to run it through the various checks before we attempt deletion
	// most important of which is checking that we have labeled it correctly
	// but it doesn't hurt to check if it exists before
	_, err := rs.getRolebinding(ctx, &generalpb.GetRequest{Id: req.GetId()})
	if err != nil {
		return &emptypb.Empty{}, err
	}

	return util.DeleteHfResource(ctx, req, rs.roleBindingClient, "rolebinding")
}

func (rs *GrpcRbacServer) ListRolebinding(ctx context.Context, listOptions *generalpb.ListOptions) (*rbacpb.RoleBindings, error) {
	doLoadFromCache := listOptions.GetLoadFromCache()
	var rolebindings []rbacv1.RoleBinding
	var err error
	if !doLoadFromCache {
		var rolebindingList *rbacv1.RoleBindingList
		rolebindingList, err = util.ListByHfClient(ctx, listOptions, rs.roleBindingClient, "rolebindings")
		if err == nil {
			rolebindings = rolebindingList.Items
		}
	} else {
		rolebindings, err = util.ListByCache(listOptions, rs.roleBindingLister, "rolebindings", rs.roleBindingSynced())
	}
	if err != nil {
		glog.Error(err)
		return &rbacpb.RoleBindings{}, err
	}

	var preparedRolebindings = make([]*rbacpb.RoleBinding, 0)
	for _, r := range rolebindings {
		pr := unmarshalRolebinding(&r)
		preparedRolebindings = append(preparedRolebindings, pr)
	}

	return &rbacpb.RoleBindings{Rolebindings: preparedRolebindings}, nil
}

func (rs *GrpcRbacServer) marshalRolebinding(ctx context.Context, preparedRoleBinding *rbacpb.RoleBinding) (*rbacv1.RoleBinding, error) {
	// first validation, the role it is referencing has to exist
	role, err := rs.roleClient.Get(ctx, preparedRoleBinding.GetRole(), metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("invalid role ref")
	}

	rb := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      preparedRoleBinding.GetName(),
			Namespace: util.GetReleaseNamespace(),
			Labels: map[string]string{
				hflabels.RBACManagedLabel: "true",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "rbac.authorization.k8s.io/v1",
					Kind:       "Role",
					Name:       role.Name,
					UID:        role.UID,
				},
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbac.RbacGroup,
			Name:     preparedRoleBinding.Role,
			Kind:     "Role",
		},
		Subjects: []rbacv1.Subject{},
	}

	for _, s := range preparedRoleBinding.GetSubjects() {
		if s.GetKind() != "Group" && s.GetKind() != "User" {
			return nil, fmt.Errorf("invalid subject kind")
		}

		rb.Subjects = append(rb.Subjects, rbacv1.Subject{
			Kind:     s.GetKind(),
			Name:     s.GetName(),
			APIGroup: rbac.RbacGroup,
		})
	}

	return &rb, nil
}

func unmarshalRolebinding(roleBinding *rbacv1.RoleBinding) *rbacpb.RoleBinding {
	prb := &rbacpb.RoleBinding{
		Name:     roleBinding.Name,
		Role:     roleBinding.RoleRef.Name,
		Subjects: []*rbacpb.Subject{},
	}

	for _, s := range roleBinding.Subjects {
		prb.Subjects = append(prb.GetSubjects(), &rbacpb.Subject{
			Kind: s.Kind,
			Name: s.Name,
		})
	}

	return prb
}

func (rs *GrpcRbacServer) getRolebinding(ctx context.Context, req *generalpb.GetRequest) (*rbacv1.RoleBinding, error) {
	rolebinding, err := util.GenericHfGetter(ctx, req, rs.roleBindingClient, rs.roleBindingLister.RoleBindings(util.GetReleaseNamespace()), "rolebinding", rs.roleBindingSynced())
	if err != nil {
		return &rbacv1.RoleBinding{}, err
	}

	if _, ok := rolebinding.Labels[hflabels.RBACManagedLabel]; !ok {
		// this isn't a hobbyfarm rolebinding. we don't serve your kind here
		glog.Error("permission denied: rolebinding not managed by hobbyfarm")
		return &rbacv1.RoleBinding{}, hferrors.GrpcError(
			codes.PermissionDenied,
			"rolebinding not managed by hobbyfarm",
			req,
		)
	}

	return rolebinding, nil
}
