package rbac

import (
	"context"

	"github.com/golang/glog"
	authrProto "github.com/hobbyfarm/gargantua/v3/protos/authr"
	rbacProto "github.com/hobbyfarm/gargantua/v3/protos/rbac"
	userProto "github.com/hobbyfarm/gargantua/v3/protos/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (rs *GrpcRbacServer) Grants(c context.Context, gr *rbacProto.GrantRequest) (*authrProto.AuthRResponse, error) {
	as, err := rs.userIndex.GetAccessSet(gr.GetUserName())
	if err != nil {
		err := status.Newf(
			codes.Internal,
			"failed to retrieve access set for user %s",
			gr.GetUserName(),
		)

		err, wde := err.WithDetails(gr)
		if wde != nil {
			return &authrProto.AuthRResponse{Success: false}, wde
		}
		glog.Errorf("failed to retrieve access set for user %s", gr.GetUserName())
		return &authrProto.AuthRResponse{Success: false}, err.Err()
	}

	return &authrProto.AuthRResponse{Success: Grants(gr.GetPermission(), as)}, nil
}

func (rs *GrpcRbacServer) GetAccessSet(c context.Context, uid *userProto.UserId) (*rbacProto.AccessSet, error) {
	return rs.userIndex.GetAccessSet(uid.GetId())
}

func (rs *GrpcRbacServer) GetHobbyfarmRoleBindings(c context.Context, uid *userProto.UserId) (*rbacProto.RoleBindings, error) {
	rbs, err := rs.userIndex.getPreparedRoleBindings(uid.GetId())
	if err != nil {
		glog.Errorf("failed to retrieve rolebindings for user %s with error: %s", uid.GetId(), err.Error())
		err := status.Newf(
			codes.Internal,
			"failed to retrieve rolebindings for user %s",
			uid.GetId(),
		)

		err, wde := err.WithDetails(uid)
		if wde != nil {
			return nil, wde
		}

		return nil, err.Err()
	}
	return rbs, nil
}
