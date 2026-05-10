package ctrpermission

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtoapplication"
	"github.com/morehao/ark-iam/iam/internal/dto/dtopermission"
	"github.com/morehao/ark-iam/iam/internal/dto/dtouser"
)

type stubRoleSvc struct {
	removeUserReq *dtouser.RemoveRoleUserReq
}

func (s *stubRoleSvc) Create(ctx *gin.Context, req *dtopermission.RoleCreateReq) (*dtopermission.RoleCreateResp, error) {
	panic("unexpected call")
}

func (s *stubRoleSvc) Delete(ctx *gin.Context, req *dtopermission.RoleDeleteReq) error {
	panic("unexpected call")
}

func (s *stubRoleSvc) Update(ctx *gin.Context, req *dtopermission.RoleUpdateReq) error {
	panic("unexpected call")
}

func (s *stubRoleSvc) Detail(ctx *gin.Context, req *dtopermission.RoleDetailReq) (*dtopermission.RoleDetailResp, error) {
	panic("unexpected call")
}

func (s *stubRoleSvc) PageList(ctx *gin.Context, req *dtopermission.RolePageListReq) (*dtopermission.RolePageListResp, error) {
	panic("unexpected call")
}

func (s *stubRoleSvc) ListUsers(ctx *gin.Context, req *dtouser.RoleUserListReq) (*dtouser.RoleUserListResp, error) {
	panic("unexpected call")
}

func (s *stubRoleSvc) AssignUsers(ctx *gin.Context, req *dtouser.AssignRoleUsersReq) error {
	panic("unexpected call")
}

func (s *stubRoleSvc) RemoveUser(ctx *gin.Context, req *dtouser.RemoveRoleUserReq) error {
	s.removeUserReq = req
	return nil
}

func (s *stubRoleSvc) ListApplications(ctx *gin.Context, req *dtouser.RoleApplicationListReq) (*dtouser.RoleApplicationListResp, error) {
	panic("unexpected call")
}

func (s *stubRoleSvc) AssignApplications(ctx *gin.Context, req *dtouser.AssignRoleApplicationsReq) error {
	panic("unexpected call")
}

type stubApplicationSvc struct {
	removeRoleReq   *dtoapplication.RemoveApplicationRoleReq
	deleteSecretReq *dtoapplication.DeleteApplicationSecretReq
}

func (s *stubApplicationSvc) Create(ctx *gin.Context, req *dtoapplication.ApplicationCreateReq) (*dtoapplication.ApplicationCreateResp, error) {
	panic("unexpected call")
}

func (s *stubApplicationSvc) Delete(ctx *gin.Context, req *dtoapplication.ApplicationDeleteReq) error {
	panic("unexpected call")
}

func (s *stubApplicationSvc) Update(ctx *gin.Context, req *dtoapplication.ApplicationUpdateReq) error {
	panic("unexpected call")
}

func (s *stubApplicationSvc) Detail(ctx *gin.Context, req *dtoapplication.ApplicationDetailReq) (*dtoapplication.ApplicationDetailResp, error) {
	panic("unexpected call")
}

func (s *stubApplicationSvc) PageList(ctx *gin.Context, req *dtoapplication.ApplicationPageListReq) (*dtoapplication.ApplicationPageListResp, error) {
	panic("unexpected call")
}

func (s *stubApplicationSvc) ListRoles(ctx *gin.Context, req *dtoapplication.ApplicationRoleListReq) (*dtoapplication.ApplicationRoleListResp, error) {
	panic("unexpected call")
}

func (s *stubApplicationSvc) AssignRoles(ctx *gin.Context, req *dtoapplication.AssignApplicationRolesReq) error {
	panic("unexpected call")
}

func (s *stubApplicationSvc) RemoveRole(ctx *gin.Context, req *dtoapplication.RemoveApplicationRoleReq) error {
	s.removeRoleReq = req
	return nil
}

func (s *stubApplicationSvc) ListSecrets(ctx *gin.Context, req *dtoapplication.ApplicationSecretListReq) (*dtoapplication.ApplicationSecretListResp, error) {
	panic("unexpected call")
}

func (s *stubApplicationSvc) CreateSecret(ctx *gin.Context, req *dtoapplication.CreateApplicationSecretReq) (*dtoapplication.CreateApplicationSecretResp, error) {
	panic("unexpected call")
}

func (s *stubApplicationSvc) DeleteSecret(ctx *gin.Context, req *dtoapplication.DeleteApplicationSecretReq) error {
	s.deleteSecretReq = req
	return nil
}

func TestRoleControllerRemoveUserBindsURI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubRoleSvc{}
	ctr := &roleCtr{roleSvc: svc}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodDelete, "/v1/iam/role/users/7/9", nil)
	ctx.Request = req
	ctx.Params = gin.Params{{Key: "roleId", Value: "7"}, {Key: "userId", Value: "9"}}

	ctr.RemoveUser(ctx)

	if svc.removeUserReq == nil {
		t.Fatal("expected RemoveUser service to receive request")
	}
	if svc.removeUserReq.RoleID != 7 || svc.removeUserReq.UserID != 9 {
		t.Fatalf("expected URI binding to populate roleID=7 and userID=9, got %+v", *svc.removeUserReq)
	}
}

func TestApplicationControllerRemoveRoleBindsURIAndQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubApplicationSvc{}
	ctr := &applicationCtr{applicationSvc: svc}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodDelete, "/v1/iam/application/roles/7?applicationId=11", nil)
	ctx.Request = req
	ctx.Params = gin.Params{{Key: "roleId", Value: "7"}}

	ctr.RemoveRole(ctx)

	if svc.removeRoleReq == nil {
		t.Fatal("expected RemoveRole service to receive request")
	}
	if svc.removeRoleReq.RoleID != 7 || svc.removeRoleReq.ApplicationID != 11 {
		t.Fatalf("expected URI/query binding to populate roleID=7 and applicationID=11, got %+v", *svc.removeRoleReq)
	}
}

func TestApplicationControllerRemoveRoleRequiresApplicationID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubApplicationSvc{}
	ctr := &applicationCtr{applicationSvc: svc}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodDelete, "/v1/iam/application/roles/7", nil)
	ctx.Request = req
	ctx.Params = gin.Params{{Key: "roleId", Value: "7"}}

	ctr.RemoveRole(ctx)

	if svc.removeRoleReq != nil {
		t.Fatal("expected RemoveRole service not to be called when applicationId is missing")
	}

	var payload struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected failure response to be JSON, got err: %v", err)
	}
	if recorder.Code == http.StatusOK && payload.Code == 0 {
		t.Fatalf("expected RemoveRole to fail when applicationId is missing, got status=%d code=%d body=%s", recorder.Code, payload.Code, recorder.Body.String())
	}
}

func TestApplicationControllerDeleteSecretBindsURI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubApplicationSvc{}
	ctr := &applicationCtr{applicationSvc: svc}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodDelete, "/v1/iam/application/secrets/15", nil)
	ctx.Request = req
	ctx.Params = gin.Params{{Key: "secretId", Value: "15"}}

	ctr.DeleteSecret(ctx)

	if svc.deleteSecretReq == nil {
		t.Fatal("expected DeleteSecret service to receive request")
	}
	if svc.deleteSecretReq.SecretID != 15 {
		t.Fatalf("expected URI binding to populate secretID=15, got %d", svc.deleteSecretReq.SecretID)
	}
}
