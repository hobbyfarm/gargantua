package userservice

import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"strings"

	"github.com/golang/glog"
	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfv2 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v2"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	"github.com/hobbyfarm/gargantua/v3/pkg/errors"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"github.com/hobbyfarm/gargantua/v3/protos/general"
	userProto "github.com/hobbyfarm/gargantua/v3/protos/user"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	empty "google.golang.org/protobuf/types/known/emptypb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
)

const (
	emailIndex = "authc.hobbyfarm.io/user-email-index"
)

type GrpcUserServer struct {
	userProto.UnimplementedUserSvcServer
	hfClientSet hfClientset.Interface
	userIndexer cache.Indexer
	ctx         context.Context
}

func NewGrpcUserServer(hfClientSet hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory, ctx context.Context) (*GrpcUserServer, error) {
	inf := hfInformerFactory.Hobbyfarm().V2().Users().Informer()
	indexers := map[string]cache.IndexFunc{emailIndex: emailIndexer}
	err := inf.AddIndexers(indexers)
	if err != nil {
		glog.Fatalf("Error adding indexer %v", err)
		return nil, err
	}
	return &GrpcUserServer{
		hfClientSet: hfClientSet,
		userIndexer: inf.GetIndexer(),
		ctx:         ctx,
	}, nil
}

func emailIndexer(obj interface{}) ([]string, error) {
	user, ok := obj.(*hfv2.User)
	if !ok {
		return []string{}, nil
	}
	return []string{user.Spec.Email}, nil
}

func (u *GrpcUserServer) CreateUser(c context.Context, cur *userProto.CreateUserRequest) (*general.ResourceId, error) {
	if len(cur.GetEmail()) == 0 || len(cur.GetPassword()) == 0 {
		return &general.ResourceId{}, errors.GrpcError(
			codes.InvalidArgument,
			"error creating user, email or password field blank",
			cur,
		)
	}

	_, err := u.GetUserByEmail(context.Background(), &userProto.GetUserByEmailRequest{Email: cur.GetEmail()})

	if err == nil {
		// the user was found, we should return info
		return &general.ResourceId{}, errors.GrpcError(
			codes.AlreadyExists,
			"user %s already exists",
			cur,
			cur.GetEmail(),
		)
	}

	newUser := hfv2.User{}

	hasher := sha256.New()
	hasher.Write([]byte(cur.GetEmail()))
	sha := base32.StdEncoding.WithPadding(-1).EncodeToString(hasher.Sum(nil))[:10]
	id := "u-" + strings.ToLower(sha)
	newUser.Name = id
	newUser.Spec.Email = cur.GetEmail()

	settings := make(map[string]string)
	settings["terminal_theme"] = "default"

	newUser.Spec.Settings = settings

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(cur.GetPassword()), bcrypt.DefaultCost)
	if err != nil {
		return &general.ResourceId{}, errors.GrpcError(
			codes.Internal,
			"error while hashing password for email %s",
			cur,
			cur.GetEmail(),
		)
	}

	newUser.Spec.Password = string(passwordHash)

	_, err = u.hfClientSet.HobbyfarmV2().Users(util.GetReleaseNamespace()).Create(u.ctx, &newUser, metav1.CreateOptions{})

	if err != nil {
		return &general.ResourceId{}, errors.GrpcError(
			codes.Internal,
			"error creating user",
			cur,
		)
	}

	return &general.ResourceId{Id: id}, nil
}

func (u *GrpcUserServer) getUser(id string) (*userProto.User, error) {
	if len(id) == 0 {
		return &userProto.User{}, fmt.Errorf("user id passed in was empty")
	}
	obj, err := u.hfClientSet.HobbyfarmV2().Users(util.GetReleaseNamespace()).Get(u.ctx, id, metav1.GetOptions{})
	if err != nil {
		return &userProto.User{}, fmt.Errorf("error while retrieving User by id: %s with error: %v", id, err)
	}

	return &userProto.User{
		Id:          obj.Name,
		Email:       obj.Spec.Email,
		Password:    obj.Spec.Password,
		AccessCodes: obj.Spec.AccessCodes,
		Settings:    obj.Spec.Settings,
	}, nil
}

func (u *GrpcUserServer) GetUserById(ctx context.Context, gur *general.ResourceId) (*userProto.User, error) {
	if len(gur.GetId()) == 0 {
		return &userProto.User{}, errors.GrpcError(
			codes.InvalidArgument,
			"no id passed in",
			gur,
		)
	}

	user, err := u.getUser(gur.GetId())

	if err != nil {
		glog.Errorf("error while retrieving user %v", err)
		return &userProto.User{}, errors.GrpcError(
			codes.NotFound,
			"no user %s found",
			gur,
			gur.GetId(),
		)
	}
	glog.V(2).Infof("retrieved user %s", user.GetId())
	return user, nil
}

func (u *GrpcUserServer) ListUser(ctx context.Context, listOptions *general.ListOptions) (*userProto.ListUsersResponse, error) {
	users, err := u.hfClientSet.HobbyfarmV2().Users(util.GetReleaseNamespace()).List(u.ctx, metav1.ListOptions{
		LabelSelector: listOptions.GetLabelSelector(),
	})

	if err != nil {
		glog.Errorf("error while retrieving users %v", err)
		return &userProto.ListUsersResponse{}, errors.GrpcError(
			codes.Internal,
			"no users found",
			listOptions,
		)
	}

	preparedUsers := []*userProto.User{} // must be declared this way so as to JSON marshal into [] instead of null
	for _, s := range users.Items {
		preparedUsers = append(preparedUsers, &userProto.User{
			Id:          s.Name,
			Email:       s.Spec.Email,
			Password:    s.Spec.Password,
			AccessCodes: s.Spec.AccessCodes,
			Settings:    s.Spec.Settings,
		})
	}

	glog.V(2).Infof("listed users")

	return &userProto.ListUsersResponse{Users: preparedUsers}, nil
}

func (u *GrpcUserServer) UpdateUser(ctx context.Context, userRequest *userProto.User) (*userProto.User, error) {
	id := userRequest.GetId()
	if id == "" {
		return &userProto.User{}, errors.GrpcError(
			codes.InvalidArgument,
			"no ID passed in",
			userRequest,
		)
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		user, err := u.hfClientSet.HobbyfarmV2().Users(util.GetReleaseNamespace()).Get(u.ctx, id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			return errors.GrpcError(
				codes.Internal,
				"error while retrieving user %s",
				userRequest,
				userRequest.GetId(),
			)
		}

		if userRequest.GetEmail() != "" {
			user.Spec.Email = userRequest.GetEmail()
		}

		if userRequest.GetPassword() != "" {
			passwordHash, err := bcrypt.GenerateFromPassword([]byte(userRequest.GetPassword()), bcrypt.DefaultCost)
			if err != nil {
				return fmt.Errorf("bad")
			}
			user.Spec.Password = string(passwordHash)
		}

		if userRequest.GetAccessCodes() != nil {
			user.Spec.AccessCodes = userRequest.GetAccessCodes()
		}
		if userRequest.GetSettings() != nil {
			user.Spec.Settings = userRequest.GetSettings()
		}

		_, updateErr := u.hfClientSet.HobbyfarmV2().Users(util.GetReleaseNamespace()).Update(u.ctx, user, metav1.UpdateOptions{})
		return updateErr
	})

	if retryErr != nil {
		return &userProto.User{}, errors.GrpcError(
			codes.Internal,
			"error attempting to update",
			userRequest,
		)
	}

	return userRequest, nil
}

func (u *GrpcUserServer) UpdateAccessCodes(ctx context.Context, updateAccessCodesRequest *userProto.UpdateAccessCodesRequest) (*userProto.User, error) {
	id := updateAccessCodesRequest.GetId()
	if id == "" {
		return &userProto.User{}, errors.GrpcError(
			codes.InvalidArgument,
			"no ID passed in",
			updateAccessCodesRequest,
		)
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		user, err := u.hfClientSet.HobbyfarmV2().Users(util.GetReleaseNamespace()).Get(u.ctx, id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			return errors.GrpcError(
				codes.Internal,
				"error while retrieving user %s",
				updateAccessCodesRequest,
				updateAccessCodesRequest.GetId(),
			)
		}

		if updateAccessCodesRequest.GetAccessCodes() != nil {
			user.Spec.AccessCodes = updateAccessCodesRequest.GetAccessCodes()
		} else {
			user.Spec.AccessCodes = make([]string, 0)
		}

		_, updateErr := u.hfClientSet.HobbyfarmV2().Users(util.GetReleaseNamespace()).Update(u.ctx, user, metav1.UpdateOptions{})
		return updateErr
	})

	if retryErr != nil {
		return &userProto.User{}, errors.GrpcError(
			codes.Internal,
			"error attempting to update",
			updateAccessCodesRequest,
		)
	}

	return &userProto.User{}, nil
}

func (u *GrpcUserServer) GetUserByEmail(c context.Context, gur *userProto.GetUserByEmailRequest) (*userProto.User, error) {
	if len(gur.GetEmail()) == 0 {
		return &userProto.User{}, errors.GrpcError(
			codes.InvalidArgument,
			"email passed in was empty",
			gur,
		)
	}

	obj, err := u.userIndexer.ByIndex(emailIndex, gur.GetEmail())
	if err != nil {
		return &userProto.User{}, errors.GrpcError(
			codes.Internal,
			"error while retrieving user by e-mail: %s with error: %v",
			gur,
			gur.GetEmail(),
			err,
		)
	}

	if len(obj) < 1 {
		return &userProto.User{}, errors.GrpcError(
			codes.NotFound,
			"user not found by email: %s",
			gur,
			gur.GetEmail(),
		)
	}

	user, ok := obj[0].(*hfv2.User)

	if !ok {
		return &userProto.User{}, errors.GrpcError(
			codes.Internal,
			"error while converting user found by email to object: %s",
			gur,
			gur.GetEmail(),
		)
	}

	return &userProto.User{
		Id:          user.Name,
		Email:       user.Spec.Email,
		Password:    user.Spec.Password,
		AccessCodes: user.Spec.AccessCodes,
		Settings:    user.Spec.Settings,
	}, nil
}

func (u *GrpcUserServer) DeleteUser(c context.Context, userId *general.ResourceId) (*empty.Empty, error) {
	id := userId.GetId()

	if len(id) == 0 {
		return &empty.Empty{}, errors.GrpcError(
			codes.InvalidArgument,
			"no id passed in",
			userId,
		)
	}

	user, err := u.hfClientSet.HobbyfarmV2().Users(util.GetReleaseNamespace()).Get(u.ctx, id, metav1.GetOptions{})
	if err != nil {
		glog.Errorf("error fetching user %s from server during delete request: %s", id, err)
		return &empty.Empty{}, errors.GrpcError(
			codes.Internal,
			"error fetching user %s from server",
			userId,
			userId.GetId(),
		)
	}

	// get a list of sessions for the user
	sessionList, err := u.hfClientSet.HobbyfarmV1().Sessions(util.GetReleaseNamespace()).List(u.ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", util.UserLabel, id),
	})

	if err != nil {
		glog.Errorf("error retrieving session list for user %s during delete: %s", id, err)
		return &empty.Empty{}, errors.GrpcError(
			codes.Internal,
			"error retrieving session list for user %s",
			userId,
			userId.GetId(),
		)
	}

	if len(sessionList.Items) > 0 {
		// there are sessions present but they may be expired. let's check
		for _, v := range sessionList.Items {
			if !v.Status.Finished {
				return &empty.Empty{}, errors.GrpcError(
					codes.Internal,
					"cannot delete user, existing sessions found",
					userId,
				)
			}
		}

		// getting here means there are sessions present but they are not active
		// let's delete them for cleanliness' sake
		if ok, err := u.deleteSessions(sessionList.Items); !ok {
			glog.Errorf("error deleting old sessions for user %s: %s", id, err)
			return &empty.Empty{}, errors.GrpcError(
				codes.Internal,
				"cannot delete user, error removing old sessions",
				userId,
			)
		}
	}

	// at this point we have either delete all old sessions, or there were no sessions  to begin with
	// so we should be safe to delete the user

	deleteErr := u.hfClientSet.HobbyfarmV2().Users(util.GetReleaseNamespace()).Delete(u.ctx, user.Name, metav1.DeleteOptions{})
	if deleteErr != nil {
		glog.Errorf("error deleting user %s: %s", id, deleteErr)
		return &empty.Empty{}, errors.GrpcError(
			codes.Internal,
			"error deleting user %s",
			userId,
			userId.GetId(),
		)
	}
	return &empty.Empty{}, nil
}

func (u *GrpcUserServer) deleteSessions(sessions []hfv1.Session) (bool, error) {
	for _, v := range sessions {
		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			err := u.hfClientSet.HobbyfarmV1().Sessions(util.GetReleaseNamespace()).Delete(u.ctx, v.Name, metav1.DeleteOptions{})
			return err
		})

		if retryErr != nil {
			return false, retryErr
		}
	}

	return true, nil
}
