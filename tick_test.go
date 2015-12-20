package tickremind

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/net/context"
	"google.golang.org/appengine/aetest"
	"google.golang.org/appengine/user"
)

func makeValidPostReq(t *testing.T, inst aetest.Instance, con context.Context, url string, h http.HandlerFunc) string {
	req, err := inst.NewRequest("POST", url, nil)
	if err != nil {
		t.Fatalf("Failed to create req: %v", err)
	}

	u := user.User{Email: "test@example.com"}

	aetest.Login(&u, req)

	rt := httptest.NewRecorder()
	h(rt, req)

	bStr := rt.Body.String()

	t.Log(rt.Code, bStr)
	if rt.Code > 299 {
		t.Fail()
	}

	return bStr
}

func TestCreateUsers(t *testing.T) {
	inst, err := aetest.NewInstance(nil)
	if err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}
	defer inst.Close()

	con, closeFunc, err := aetest.NewContext()

	var rsp string

	rsp = makeValidPostReq(t, inst, con, "/", root)
	rsp = makeValidPostReq(t, inst, con, "/create_user", createTickUser)
	rsp = makeValidPostReq(t, inst, con, "/create_rule/?name=weight&bucket=day", createTickRule)

	jObj := struct{ Key string }{Key: "empty"}
	errJ := json.Unmarshal([]byte(rsp), &jObj)
	if errJ != nil {
		t.Log(errJ)
	}
	rsp = makeValidPostReq(t, inst, con, fmt.Sprintf("/tick/123?key=%s", jObj.Key), makeTick)
	rsp = makeValidPostReq(t, inst, con, fmt.Sprintf("/tick/256?key=%s", jObj.Key), makeTick)

	closeFunc()

}
