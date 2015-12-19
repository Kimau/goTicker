package tickremind

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/net/context"
	"google.golang.org/appengine/aetest"
	"google.golang.org/appengine/user"
)

func makeValidPostReq(t *testing.T, inst aetest.Instance, con context.Context, url string, h http.HandlerFunc) {
	req, err := inst.NewRequest("POST", url, nil)
	if err != nil {
		t.Fatalf("Failed to create req: %v", err)
	}

	u := user.User{Email: "test@example.com"}

	aetest.Login(&u, req)

	rt := httptest.NewRecorder()
	h(rt, req)

	if rt.Code > 299 {
		t.Error(rt.Body)
		t.Fail()
	} else {
		t.Log(rt.Code, rt.Body)
	}

}

func TestCreateUsers(t *testing.T) {
	inst, err := aetest.NewInstance(nil)
	if err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}
	defer inst.Close()

	con, closeFunc, err := aetest.NewContext()

	makeValidPostReq(t, inst, con, "/", root)
	makeValidPostReq(t, inst, con, "/create_user", createTickUser)
	makeValidPostReq(t, inst, con, "/create_rule/weight?bucket=day", createTickRule)
	makeValidPostReq(t, inst, con, "/tick/weight/123", makeTick)
	makeValidPostReq(t, inst, con, "/tick/weight/256", makeTick)

	closeFunc()

}
