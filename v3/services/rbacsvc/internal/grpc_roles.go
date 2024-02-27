package rbac

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"github.com/hobbyfarm/gargantua/v3/protos/general"
	rbacProto "github.com/hobbyfarm/gargantua/v3/protos/rbac"
	"google.golang.org/grpc/codes"
	empty "google.golang.org/protobuf/types/known/emptypb"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

func (rs *GrpcRbacServer) CreateRole(c context.Context, cr *rbacProto.Role) (*empty.Empty, error) {
	role, err := marshalRole(cr)
	if err != nil {
		glog.Errorf("invalid role: %v", err)
		return &empty.Empty{}, hferrors.GrpcError(
			codes.InvalidArgument,
			"invalid role",
			cr,
		)
	}

	_, err = rs.roleClient.Create(c, role, metav1.CreateOptions{})
	if err != nil {
		glog.Errorf("error creating role in kubernetes: %v", err)
		return &empty.Empty{}, hferrors.GrpcError(
			codes.Internal,
			"error creating role",
			cr,
		)
	}

	return &empty.Empty{}, nil
}

func (rs *GrpcRbacServer) GetRole(c context.Context, gr *general.GetRequest) (*rbacProto.Role, error) {
	role, err := rs.getRole(c, gr)
	if err != nil {
		return nil, err
	}
	return unmarshalRole(role), nil
}

func (rs *GrpcRbacServer) UpdateRole(c context.Context, ur *rbacProto.Role) (*empty.Empty, error) {
	role, err := marshalRole(ur)
	if err != nil {
		glog.Errorf("invalid role: %v", err)
		return &empty.Empty{}, hferrors.GrpcError(
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
		return &empty.Empty{}, retryErr
	}

	return &empty.Empty{}, nil
}

func (rs *GrpcRbacServer) DeleteRole(ctx context.Context, req *general.ResourceId) (*empty.Empty, error) {
	// we want to get the role first as this allows us to run it through the various checks before we attempt deletion
	// most important of which is checking that we have labeled it correctly
	// but it doesn't hurt to check if it exists before
	_, err := rs.getRole(ctx, &general.GetRequest{Id: req.GetId()})
	if err != nil {
		return &empty.Empty{}, err
	}

	return util.DeleteHfResource(ctx, req, rs.roleClient, "role")
}

func (rs *GrpcRbacServer) ListRole(ctx context.Context, listOptions *general.ListOptions) (*rbacProto.Roles, error) {
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
		return &rbacProto.Roles{}, err
	}

	var preparedRoles = make([]*rbacProto.Role, 0)
	for _, r := range roles {
		pr := unmarshalRole(&r)
		preparedRoles = append(preparedRoles, pr)
	}

	return &rbacProto.Roles{Roles: preparedRoles}, nil
}

func marshalRole(preparedRole *rbacProto.Role) (*rbacv1.Role, error) {
	role := rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      preparedRole.GetName(),
			Namespace: util.GetReleaseNamespace(),
			Labels: map[string]string{
				util.RBACManagedLabel: "true",
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

func unmarshalRole(role *rbacv1.Role) (preparedRole *rbacProto.Role) {
	preparedRole = &rbacProto.Role{}
	preparedRole.Name = role.Name

	for _, r := range role.Rules {
		preparedRole.Rules = append(preparedRole.Rules, &rbacProto.Rule{
			Resources: r.Resources,
			Verbs:     r.Verbs,
			ApiGroups: r.APIGroups,
		})
	}

	return preparedRole
}

func (rs *GrpcRbacServer) getRole(ctx context.Context, req *general.GetRequest) (*rbacv1.Role, error) {
	role, err := util.GenericHfGetter(ctx, req, rs.roleClient, rs.roleLister.Roles(util.GetReleaseNamespace()), "role", rs.roleSynced())
	if err != nil {
		return &rbacv1.Role{}, err
	}

	if _, ok := role.Labels[util.RBACManagedLabel]; !ok {
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
