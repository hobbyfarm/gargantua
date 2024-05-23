package rbac

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	hflabels "github.com/hobbyfarm/gargantua/v3/pkg/labels"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	rbacpb "github.com/hobbyfarm/gargantua/v3/protos/rbac"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/emptypb"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

func (rs *GrpcRbacServer) CreateRole(c context.Context, cr *rbacpb.Role) (*emptypb.Empty, error) {
	role, err := marshalRole(cr)
	if err != nil {
		glog.Errorf("invalid role: %v", err)
		return &emptypb.Empty{}, hferrors.GrpcError(
			codes.InvalidArgument,
			"invalid role",
			cr,
		)
	}

	_, err = rs.roleClient.Create(c, role, metav1.CreateOptions{})
	if err != nil {
		glog.Errorf("error creating role in kubernetes: %v", err)
		return &emptypb.Empty{}, hferrors.GrpcError(
			codes.Internal,
			"error creating role",
			cr,
		)
	}

	return &emptypb.Empty{}, nil
}

func (rs *GrpcRbacServer) GetRole(c context.Context, gr *generalpb.GetRequest) (*rbacpb.Role, error) {
	role, err := rs.getRole(c, gr)
	if err != nil {
		return nil, err
	}
	return unmarshalRole(role), nil
}

func (rs *GrpcRbacServer) UpdateRole(c context.Context, ur *rbacpb.Role) (*emptypb.Empty, error) {
	role, err := marshalRole(ur)
	if err != nil {
		glog.Errorf("invalid role: %v", err)
		return &emptypb.Empty{}, hferrors.GrpcError(
			codes.InvalidArgument,
			"invalid role",
			ur,
		)
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_, err := rs.roleClient.Update(c, role, metav1.UpdateOptions{})
		if err != nil {
			glog.Errorf("error while updating role in kubernetes: %v", err)
			return hferrors.GrpcError(
				codes.Internal,
				"error updating role",
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

func (rs *GrpcRbacServer) DeleteRole(ctx context.Context, req *generalpb.ResourceId) (*emptypb.Empty, error) {
	// we want to get the role first as this allows us to run it through the various checks before we attempt deletion
	// most important of which is checking that we have labeled it correctly
	// but it doesn't hurt to check if it exists before
	_, err := rs.getRole(ctx, &generalpb.GetRequest{Id: req.GetId()})
	if err != nil {
		return &emptypb.Empty{}, err
	}

	return util.DeleteHfResource(ctx, req, rs.roleClient, "role")
}

func (rs *GrpcRbacServer) ListRole(ctx context.Context, listOptions *generalpb.ListOptions) (*rbacpb.Roles, error) {
	doLoadFromCache := listOptions.GetLoadFromCache()
	var roles []rbacv1.Role
	var err error
	if !doLoadFromCache {
		var roleList *rbacv1.RoleList
		roleList, err = util.ListByHfClient(ctx, listOptions, rs.roleClient, "roles")
		if err == nil {
			roles = roleList.Items
		}
	} else {
		roles, err = util.ListByCache(listOptions, rs.roleLister, "roles", rs.roleSynced())
	}
	if err != nil {
		glog.Error(err)
		return &rbacpb.Roles{}, err
	}

	var preparedRoles = make([]*rbacpb.Role, 0)
	for _, r := range roles {
		pr := unmarshalRole(&r)
		preparedRoles = append(preparedRoles, pr)
	}

	return &rbacpb.Roles{Roles: preparedRoles}, nil
}

func marshalRole(preparedRole *rbacpb.Role) (*rbacv1.Role, error) {
	role := rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      preparedRole.GetName(),
			Namespace: util.GetReleaseNamespace(),
			Labels: map[string]string{
				hflabels.RBACManagedLabel: "true",
			},
		},
		Rules: []rbacv1.PolicyRule{},
	}

	for _, r := range preparedRole.GetRules() {
		for _, group := range r.GetApiGroups() {
			if group != "hobbyfarm.io" && group != "rbac.authorization.k8s.io" {
				return nil, fmt.Errorf("invalid api group specified: %v", group)
			}
		}

		role.Rules = append(role.Rules, rbacv1.PolicyRule{
			Verbs:     r.GetVerbs(),
			APIGroups: r.GetApiGroups(),
			Resources: r.GetResources(),
		})
	}

	return &role, nil
}

func unmarshalRole(role *rbacv1.Role) (preparedRole *rbacpb.Role) {
	preparedRole = &rbacpb.Role{}
	preparedRole.Name = role.Name

	for _, r := range role.Rules {
		preparedRole.Rules = append(preparedRole.Rules, &rbacpb.Rule{
			Resources: r.Resources,
			Verbs:     r.Verbs,
			ApiGroups: r.APIGroups,
		})
	}

	return preparedRole
}

func (rs *GrpcRbacServer) getRole(ctx context.Context, req *generalpb.GetRequest) (*rbacv1.Role, error) {
	role, err := util.GenericHfGetter(ctx, req, rs.roleClient, rs.roleLister.Roles(util.GetReleaseNamespace()), "role", rs.roleSynced())
	if err != nil {
		return &rbacv1.Role{}, err
	}

	if _, ok := role.Labels[hflabels.RBACManagedLabel]; !ok {
		// this isn't a hobbyfarm role. we don't serve your kind here
		glog.Error("permission denied: role not managed by hobbyfarm")
		return &rbacv1.Role{}, hferrors.GrpcError(
			codes.PermissionDenied,
			"role not managed by hobbyfarm",
			req,
		)
	}

	return role, nil
}
