package authnservice

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	hflabels "github.com/hobbyfarm/gargantua/v3/pkg/labels"
	settingUtil "github.com/hobbyfarm/gargantua/v3/pkg/setting"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	authnpb "github.com/hobbyfarm/gargantua/v3/protos/authn"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	settingpb "github.com/hobbyfarm/gargantua/v3/protos/setting"
	userpb "github.com/hobbyfarm/gargantua/v3/protos/user"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

type PreparedScheduledEvent struct {
	Id          string `json:"id"`
	Description string `json:"description"`
	Name        string `json:"name"`
	EndDate     string `json:"end_timestamp"`
}

func (a AuthServer) ChangePasswordFunc(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	user, err := a.internalAuthnServer.AuthN(r.Context(), &authnpb.AuthNRequest{
		Token: token,
	})
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to change password")
		return
	}

	r.ParseForm()

	oldPassword := r.PostFormValue("old_password")
	newPassword := r.PostFormValue("new_password")

	err = a.ChangePassword(user, oldPassword, newPassword, r.Context())

	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "error", fmt.Sprintf("error changing password for user %s", user.Id))
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "success", "password changed")

	glog.V(2).Infof("changed password for user %s", user.Email)
}

func (a AuthServer) UpdateSettingsFunc(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	user, err := a.internalAuthnServer.AuthN(r.Context(), &authnpb.AuthNRequest{
		Token: token,
	})
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to update settings")
		return
	}

	r.ParseForm()

	newSettings := make(map[string]string)
	for key := range r.Form {
		newSettings[key] = r.FormValue(key) //Ignore when multiple values were set for one argument. Just take the first one
	}

	err = a.UpdateSettings(user, newSettings, r.Context())

	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "error", fmt.Sprintf("error updating settings for user %s", user.Id))
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "success", "settings updated")

	glog.V(2).Infof("settings updated for user %s", user.Email)
}

func (a AuthServer) ListAccessCodeFunc(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	user, err := a.internalAuthnServer.AuthN(r.Context(), &authnpb.AuthNRequest{
		Token: token,
	})
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get accesscode")
		return
	}

	accessCodes := user.GetAccessCodes()
	// If "accessCodes" variable is nil -> convert it to an empty slice
	if accessCodes == nil {
		accessCodes = []string{}
	}

	encodedACList, err := json.Marshal(accessCodes)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedACList)

	glog.V(2).Infof("retrieved accesscode list for user %s", user.GetEmail())
}

func (a AuthServer) RetreiveSettingsFunc(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	user, err := a.internalAuthnServer.AuthN(r.Context(), &authnpb.AuthNRequest{
		Token: token,
	})
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get settings")
		return
	}

	settings := user.GetSettings()
	// If "settings" variable is nil -> convert it to an empty map
	if settings == nil {
		settings = make(map[string]string)
	}

	encodedSettings, err := json.Marshal(settings)

	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedSettings)

	glog.V(2).Infof("retrieved settings list for user %s", user.Email)
}

func (a AuthServer) AddAccessCodeFunc(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	user, err := a.internalAuthnServer.AuthN(r.Context(), &authnpb.AuthNRequest{
		Token: token,
	})
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get accesscode")
		return
	}

	r.ParseForm()

	accessCode := strings.ToLower(r.PostFormValue("access_code"))

	// Validate that the AccessCode
	// starts and ends with an alphanumeric character.
	// Only contains '.' and '-' special characters in the middle.
	// Must be at least 5 Characters long (1 Start + At least 3 in the middle + 1 End)
	validator, _ := regexp.Compile(`^[a-z0-9]+[a-z0-9\-\.]{3,}[a-z0-9]+$`)
	validAccessCode := validator.MatchString(accessCode)
	if !validAccessCode {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "AccessCode does not meet criteria.")
		return
	}

	set, err := a.settingClient.GetSettingValue(r.Context(), &generalpb.ResourceId{Id: string(settingUtil.StrictAccessCodeValidation)})
	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error adding accesscode")
		return
	}

	if s, ok := set.GetValue().(*settingpb.SettingValue_BoolValue); err != nil || !ok || set == nil {
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error adding accesscode")
		return
	} else if s.BoolValue {
		validation, err := a.acClient.ValidateExistence(r.Context(), &generalpb.ResourceId{Id: accessCode})
		if err != nil {
			util.ReturnHTTPMessage(w, r, 500, "internalerror", "error adding accesscode")
			return
		}
		if !validation.Valid {
			util.ReturnHTTPMessage(w, r, 400, "badrequest", "AccessCode is invalid.")
			return
		}
	}

	err = a.AddAccessCode(user, accessCode, r.Context())

	if err != nil {
		glog.Error(err)
		util.ReturnHTTPMessage(w, r, 500, "error", "error adding access code")
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "success", accessCode)

	glog.V(2).Infof("added accesscode %s to user %s", accessCode, user.Email)
}

func (a AuthServer) RemoveAccessCodeFunc(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	user, err := a.internalAuthnServer.AuthN(r.Context(), &authnpb.AuthNRequest{
		Token: token,
	})
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get accesscode")
		return
	}

	vars := mux.Vars(r)

	accessCode := strings.ToLower(vars["access_code"])

	err = a.RemoveAccessCodes(user, []string{accessCode}, r.Context())

	if err != nil {
		glog.Error(err)
		util.ReturnHTTPMessage(w, r, 500, "error", "error removing access code")
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "success", accessCode)

	glog.V(2).Infof("removed accesscode %s to user %s", accessCode, user.Email)
}

func (a AuthServer) RemoveMultipleAccessCodesFunc(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	user, err := a.internalAuthnServer.AuthN(r.Context(), &authnpb.AuthNRequest{
		Token: token,
	})
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get accesscode")
		return
	}

	var acUnmarshaled []string
	err = json.NewDecoder(r.Body).Decode(&acUnmarshaled)
	if err != nil {
		glog.Errorf("error while unmarshaling accesscodes %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "error attempting to update")
		return
	}

	err = a.RemoveAccessCodes(user, acUnmarshaled, r.Context())

	if err != nil {
		glog.Error(err)
		util.ReturnHTTPMessage(w, r, 500, "error", "error removing access code")
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "success", "access codes have been deleted")

	glog.V(2).Infof("removed accesscodes %v to user %s", acUnmarshaled, user.Email)
}

func (a AuthServer) AddAccessCode(user *userpb.User, accessCode string, ctx context.Context) error {
	if len(user.GetId()) == 0 || len(accessCode) == 0 {
		return fmt.Errorf("bad parameters passed, %s:%s", user.GetId(), accessCode)
	}

	accessCode = strings.ToLower(accessCode)

	// check if this is an otac
	otac, err := a.acClient.GetOtac(ctx, &generalpb.GetRequest{Id: accessCode})
	if err != nil {
		//otac does not exist. normal access code
	} else {
		//otac does exist, check if already redeemed
		if otac.GetRedeemedTimestamp() != "" && otac.GetUser() != user.GetId() {
			return fmt.Errorf("one time access code already in use")
		}
		if otac.GetRedeemedTimestamp() == "" {
			//otac not in use, redeem now
			otac.User = user.GetId()
			otac.RedeemedTimestamp = time.Now().Format(time.UnixDate)
			_, err = a.acClient.UpdateOtac(ctx, otac)
			if err != nil {
				return fmt.Errorf("error redeeming one time access code %v", err)
			}
		}
		// when we are here the user had the otac added previously
	}

	if len(user.GetAccessCodes()) == 0 {
		user.AccessCodes = []string{}
	} else {
		for _, ac := range user.GetAccessCodes() {
			if ac == accessCode {
				return fmt.Errorf("access code already added to user")
			}
		}
	}

	// Important: user.GetPassword() contains the hashed password. Hence, it can and should not be updated!
	// Otherwise the password would be updated to the current password hash value.
	// To not update the password, we therefore need to provide an empty string or a user object without password.
	user = &userpb.User{
		Id:          user.Id,
		AccessCodes: append(user.AccessCodes, accessCode),
	}

	_, err = a.userClient.UpdateUser(ctx, user)

	if err != nil {
		return err
	}

	return nil
}

func (a AuthServer) RemoveAccessCodes(user *userpb.User, accessCodes []string, ctx context.Context) error {
	if len(user.GetId()) == 0 || len(accessCodes) == 0 || len(accessCodes[0]) == 0 {
		return fmt.Errorf("bad parameters passed, %s:%s", user.GetId(), accessCodes)
	}

	if len(user.AccessCodes) == 0 {
		// there were no access codes at this point so what are we doing
		return fmt.Errorf("user %s has no accesscodes", user.GetId())
	}

	for index, inputAccessCode := range accessCodes {
		accessCodes[index] = strings.ToLower(inputAccessCode)
	}

	newAccessCodes := util.GetUniqueStringsFromSlice(user.AccessCodes, accessCodes)

	// No codes have been changed, return before performing updates
	if len(user.AccessCodes) == len(newAccessCodes) {
		return nil
	}

	// Important: user.GetPassword() contains the hashed password. Hence, it can and should not be updated!
	// Otherwise the password would be updated to the current password hash value.
	// To not update the password, we therefore need to provide an empty string or a user object without password.
	updateAccessCode := &userpb.UpdateAccessCodesRequest{
		Id:          user.Id,
		AccessCodes: newAccessCodes,
	}

	_, err := a.userClient.UpdateAccessCodes(ctx, updateAccessCode)

	if err != nil {
		return err
	}

	return nil
}

func (a AuthServer) ChangePassword(user *userpb.User, oldPassword string, newPassword string, ctx context.Context) error {
	if len(user.GetId()) == 0 || len(oldPassword) == 0 || len(newPassword) == 0 {
		return fmt.Errorf("bad parameters passed, %s", user.GetId())
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.GetPassword()), []byte(oldPassword))

	if err != nil {
		glog.Errorf("old password incorrect for user ID %s: %v", user.GetId(), err)
		return fmt.Errorf("bad password change")
	}

	user.Password = newPassword

	_, err = a.userClient.UpdateUser(ctx, user)

	if err != nil {
		return err
	}

	return nil
}

func (a AuthServer) UpdateSettings(user *userpb.User, newSettings map[string]string, ctx context.Context) error {
	if len(user.GetId()) == 0 {
		return fmt.Errorf("bad parameters passed, %s", user.GetId())
	}

	settings := user.Settings

	for i, setting := range newSettings {
		settings[i] = setting
	}

	user = &userpb.User{
		Id:       user.GetId(),
		Settings: settings,
	}

	_, err := a.userClient.UpdateUser(ctx, user)

	if err != nil {
		return err
	}

	return nil
}

func (a AuthServer) RegisterWithAccessCodeFunc(w http.ResponseWriter, r *http.Request) {
	set, err := a.settingClient.GetSettingValue(r.Context(), &generalpb.ResourceId{Id: string(settingUtil.SettingRegistrationDisabled)})
	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error performing registration")
		return
	}

	if s, ok := set.GetValue().(*settingpb.SettingValue_BoolValue); err != nil || !ok || set == nil {
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error performing registration")
		return
	} else if s.BoolValue {
		util.ReturnHTTPMessage(w, r, http.StatusConflict, "disabled", "registration disabled")
		return
	}
	r.ParseForm()

	email := r.PostFormValue("email")
	accessCode := strings.ToLower(r.PostFormValue("access_code"))
	password := r.PostFormValue("password")
	// should we reconcile based on the access code posted in? nah

	if len(email) == 0 || len(accessCode) == 0 || len(password) == 0 {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "invalid input. required fields: email, access_code, password")
		return
	}

	// Validate that the AccessCode
	// starts and ends with an alphanumeric character.
	// Only contains '.' and '-' special characters in the middle.
	// Must be at least 5 Characters long (1 Start + At least 3 in the middle + 1 End)
	validator, _ := regexp.Compile(`^[a-z0-9]+[a-z0-9\-\.]{3,}[a-z0-9]+$`)
	validAccessCode := validator.MatchString(accessCode)
	if !validAccessCode {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "AccessCode does not meet criteria.")
		return
	}

	set, err = a.settingClient.GetSettingValue(r.Context(), &generalpb.ResourceId{Id: string(settingUtil.StrictAccessCodeValidation)})
	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error performing registration")
		return
	}

	if s, ok := set.GetValue().(*settingpb.SettingValue_BoolValue); err != nil || !ok || set == nil {
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error performing registration")
		return
	} else if s.BoolValue {
		validation, err := a.acClient.ValidateExistence(r.Context(), &generalpb.ResourceId{Id: accessCode})
		if err != nil {
			util.ReturnHTTPMessage(w, r, 500, "internalerror", "error performing registration")
			return
		}
		if !validation.Valid {
			util.ReturnHTTPMessage(w, r, 400, "badrequest", "AccessCode is invalid.")
			return
		}
	}

	userId, err := a.userClient.CreateUser(r.Context(), &userpb.CreateUserRequest{
		Email:    email,
		Password: password,
	})

	if err != nil {
		s := status.Convert(err)
		details, _ := hferrors.ExtractDetail[*userpb.CreateUserRequest](s)
		if s.Code() == codes.InvalidArgument {
			glog.Errorf("error creating user, invalid argument for user with email: %s", details.GetEmail())
			util.ReturnHTTPMessage(w, r, 400, "error", s.Message())
			return
		} else if s.Code() == codes.AlreadyExists {
			glog.Errorf("user with email %s already exists", details.GetEmail())
			util.ReturnHTTPMessage(w, r, 409, "error", s.Message())
			return
		}
		glog.Errorf("error creating user: %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "error", "error creating user")
		return
	}

	// from this point, the user is created
	// we are now trying to add the access code he provided

	user, err := a.userClient.GetUserById(r.Context(), &generalpb.GetRequest{
		Id: userId.GetId(),
	})

	if err != nil {
		s := status.Convert(err)
		details, _ := hferrors.ExtractDetail[*generalpb.GetRequest](s)
		if s.Code() == codes.InvalidArgument {
			glog.Error("error retrieving created user, no id passed in")
		} else {
			glog.Errorf("error while retrieving created user %s: %s", details.GetId(), hferrors.GetErrorMessage(err))
		}
		util.ReturnHTTPMessage(w, r, 500, "error", "error creating user with accesscode")
	}

	err = a.AddAccessCode(user, accessCode, r.Context())

	if err != nil {
		glog.Errorf("error adding accessCode to newly created user %s %v", email, err)
	}

	glog.V(2).Infof("created user %s", email)
	util.ReturnHTTPMessage(w, r, 201, "info", "created user")
}

func (a AuthServer) LoginFunc(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	email := r.PostFormValue("email")
	password := r.PostFormValue("password")

	user, err := a.userClient.GetUserByEmail(r.Context(), &userpb.GetUserByEmailRequest{Email: email})

	if err != nil {
		glog.Errorf("there was an error retrieving the user %s: %v", email, err)
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "login failed")
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.GetPassword()), []byte(password))

	if err != nil {
		glog.Errorf("password incorrect for user %s: %v", email, err)
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "login failed")
		return
	}

	token, err := a.GenerateJWT(user)

	if err != nil {
		glog.Error(err)
	}

	a.userClient.SetLastLoginTimestamp(r.Context(), &generalpb.ResourceId{Id: user.GetId()})

	util.ReturnHTTPMessage(w, r, 200, "authorized", token)
}

func (a AuthServer) GenerateJWT(user *userpb.User) (string, error) {
	// Get Expiration Date Setting
	setting, err := a.settingClient.GetSettingValue(context.Background(), &generalpb.ResourceId{Id: string(settingUtil.UserTokenExpiration)})
	if err != nil {
		return "", err
	}

	tokenExpiration := time.Duration(24)
	if s, ok := setting.GetValue().(*settingpb.SettingValue_Int64Value); err != nil || !ok || setting == nil {
		return "", fmt.Errorf("error retreiving retention Time setting")
	} else {
		tokenExpiration = time.Duration(s.Int64Value)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": user.GetEmail(),
		"nbf":   time.Now().Unix(),                                  // not valid before now
		"exp":   time.Now().Add(time.Hour * tokenExpiration).Unix(), // expire after [tokenExpiration] hours
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString([]byte(user.GetPassword()))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (a *AuthServer) GetAccessSet(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	user, err := a.internalAuthnServer.AuthN(r.Context(), &authnpb.AuthNRequest{
		Token: token,
	})
	if err != nil {
		util.ReturnHTTPMessage(w, r, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	// need to get the user's access set and publish to front end
	as, err := a.rbacClient.GetAccessSet(r.Context(), &generalpb.ResourceId{Id: user.GetId()})
	if err != nil {
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "internalerror", "internal error fetching access set")
		glog.Error(err)
		return
	}

	encodedAS, err := util.GetProtoMarshaller().Marshal(as)
	if err != nil {
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "internalerror", "internal error encoding access set")
		glog.Error(err)
		return
	}

	util.ReturnHTTPContent(w, r, http.StatusOK, "access_set", encodedAS)
}

func (a AuthServer) DeleteUser(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	user, err := a.internalAuthnServer.AuthN(r.Context(), &authnpb.AuthNRequest{
		Token: token,
	})
	if err != nil {
		util.ReturnHTTPMessage(w, r, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	_, err = a.userClient.DeleteUser(r.Context(), &generalpb.ResourceId{Id: user.GetId()})

	if err != nil {
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "error", "Error during account deletion")
	}
}

func (a AuthServer) ListScheduledEventsFunc(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	user, err := a.internalAuthnServer.AuthN(r.Context(), &authnpb.AuthNRequest{
		Token: token,
	})
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list suitable scheduledevents")
		return
	}

	// This holds a map of AC -> SE
	accessCodeScheduledEvent := make(map[string]PreparedScheduledEvent)

	// First we add ScheduledEvents based on OneTimeAccessCodes
	otacReq, _ := labels.NewRequirement(hflabels.OneTimeAccessCodeLabel, selection.In, user.GetAccessCodes())
	selector := labels.NewSelector()
	selector = selector.Add(*otacReq)

	otacList, err := a.acClient.ListOtac(r.Context(), &generalpb.ListOptions{LabelSelector: selector.String()})

	if err == nil {
		for _, otac := range otacList.GetOtacs() {
			se, err := a.scheduledEventClient.GetScheduledEvent(r.Context(), &generalpb.GetRequest{Id: otac.Labels[hflabels.ScheduledEventLabel]})
			if err != nil {
				continue
			}
			endTime := se.GetEndTime()

			// If OTAC specifies a max Duration we need to calculate the EndTime correctly
			if otac.GetMaxDuration() != "" {
				otacEndTime, err := time.Parse(time.UnixDate, otac.GetRedeemedTimestamp())
				if err != nil {
					continue
				}
				otacDurationWithDays, _ := util.GetDurationWithDays(otac.GetMaxDuration())
				otacDuration, err := time.ParseDuration(otacDurationWithDays)
				if err != nil {
					continue
				}
				otacEndTime = otacEndTime.Add(otacDuration)
				endTime = otacEndTime.Format(time.UnixDate)
			}

			accessCodeScheduledEvent[otac.GetId()] = PreparedScheduledEvent{se.GetId(), se.GetDescription(), se.GetName(), endTime}
		}
	}

	acReq, err := labels.NewRequirement(hflabels.AccessCodeLabel, selection.In, user.GetAccessCodes())
	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "error", "internal error while retrieving scheduled events")
		return
	}
	selector = labels.NewSelector()
	selector = selector.Add(*acReq)

	// Afterwards we retreive the normal AccessCodes
	acList, err := a.acClient.ListAc(r.Context(), &generalpb.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "error", "internal error while retrieving scheduled events")
		return
	}
	accessCodes := acList.GetAccessCodes()
	//Getting single SEs should be faster than listing all of them and iterating them in O(n^2), in most cases users only have a hand full of accessCodes.
	for _, ac := range accessCodes {
		se, err := a.scheduledEventClient.GetScheduledEvent(r.Context(), &generalpb.GetRequest{Id: ac.GetLabels()[hflabels.ScheduledEventLabel]})
		if err != nil {
			glog.Error(err)
			continue
		}
		accessCodeScheduledEvent[ac.GetId()] = PreparedScheduledEvent{se.GetId(), se.GetDescription(), se.GetName(), se.GetEndTime()}
	}

	encodedMap, err := json.Marshal(accessCodeScheduledEvent)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedMap)
}
