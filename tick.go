package tickremind

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/user"
)

//----------------------------------------
type UserSettings struct {
	HasPebble bool
	Twitter   string
}

type TickRule struct {
	RuleName   string
	IsBucketed bool
	Bucket     int64
}

type TickEntryValue struct {
	When  time.Time
	Value int
}

type HtmlTickObj struct {
	RuleName string
	RuleKey  string
	Entries  []TickEntryValue
}

//---------------------------------------- Setup Functions

func init() {
	http.HandleFunc("/", root)
	http.HandleFunc("/rules", getRules)
	http.HandleFunc("/create_user", createTickUser)
	http.HandleFunc("/create_rule/", createTickRule)
	http.HandleFunc("/tick/", makeTick)
}

//---------------------------------------- Key Users
const TICK_SETTINGS_KIND = "TickSettings"
const TICK_RULE = "TickRule"
const TICK_ENTRY_VALUE = "TickEntryValue"

func tickSettingsKey(ctx context.Context, u *user.User) *datastore.Key {
	return datastore.NewKey(ctx, TICK_SETTINGS_KIND, u.String(), 0, nil)
}

//---------------------------------------- Useful Funcs

const HTML_CREATE_USER_FORM = `
<form action="/create_user" method="post">
 <div><input type="submit" value="Create User"></div>
</form>`

//---------------------------------------- API Funcs

func root(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	// Get User
	u := user.Current(ctx)
	if u == nil {
		http.Error(w, "User Must Log In", http.StatusBadRequest)
		return
	}

	// Get User Settings
	var settingObj UserSettings
	usrKey := tickSettingsKey(ctx, u)
	if errDB := datastore.Get(ctx, usrKey, &settingObj); errDB != nil {
		fmt.Fprintf(w, "<div>Please create your user entry... %s</div> %s", errDB.Error(), HTML_CREATE_USER_FORM)
		return
	}

	htmlObj := struct {
		HasPebble bool
		Twitter   string
		Rules     []HtmlTickObj
	}{
		HasPebble: settingObj.HasPebble,
		Twitter:   settingObj.Twitter,
	}

	// Get Rules
	ruleQ := datastore.NewQuery(TICK_RULE).Ancestor(usrKey).Order("RuleName")
	iter := ruleQ.Run(ctx)
	for {
		var tr TickRule
		k, err := iter.Next(&tr)
		if err == datastore.Done {
			break // No further entities match the query.
		}

		hto := HtmlTickObj{
			RuleName: tr.RuleName,
			RuleKey:  k.Encode(),
		}

		q := datastore.NewQuery(TICK_ENTRY_VALUE).Ancestor(k).Order("When").Limit(356)
		_, errEntQ := q.GetAll(ctx, &hto.Entries)
		if errEntQ != nil {
			http.Error(w, errEntQ.Error(), http.StatusBadRequest)
			return
		}

		htmlObj.Rules = append(htmlObj.Rules, hto)
	}

	t, e := template.ParseFiles("root.html")
	if e != nil {
		fmt.Fprintf(w, "-- TEMPLATE ERROR --\n %s", e.Error())
	}

	t.Execute(w, htmlObj)
}

func getRules(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	// Get User
	u := user.Current(ctx)
	if u == nil {
		http.Error(w, "User Must Log In", http.StatusBadRequest)
		return
	}

	// Get User Settings
	var settingObj UserSettings
	usrKey := tickSettingsKey(ctx, u)
	if errDB := datastore.Get(ctx, usrKey, &settingObj); errDB != nil {
		fmt.Fprintf(w, "<div>Please create your user entry... %s</div> %s", errDB.Error(), HTML_CREATE_USER_FORM)
		return
	}

	htmlObj := struct {
		HasPebble bool
		Twitter   string
		Rules     []HtmlTickObj
	}{
		HasPebble: settingObj.HasPebble,
		Twitter:   settingObj.Twitter,
	}

	// Get Rules
	ruleQ := datastore.NewQuery(TICK_RULE).Ancestor(usrKey).Order("RuleName")
	iter := ruleQ.Run(ctx)
	for {
		var tr TickRule
		k, err := iter.Next(&tr)
		if err == datastore.Done {
			break // No further entities match the query.
		}

		hto := HtmlTickObj{
			RuleName: tr.RuleName,
			RuleKey:  k.Encode(),
		}

		q := datastore.NewQuery(TICK_ENTRY_VALUE).Ancestor(k).Order("When").Limit(356)
		_, errEntQ := q.GetAll(ctx, &hto.Entries)
		if errEntQ != nil {
			http.Error(w, errEntQ.Error(), http.StatusBadRequest)
			return
		}

		htmlObj.Rules = append(htmlObj.Rules, hto)
	}

	jObj, errJson := json.Marshal(htmlObj)
	if errJson != nil {
		http.Error(w, errJson.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jObj)
}

func createTickUser(w http.ResponseWriter, r *http.Request) {
	if strings.ToLower(r.Method) != "post" {
		http.Error(w, "Must be POST", http.StatusBadRequest)
		return
	}

	ctx := appengine.NewContext(r)
	u := user.Current(ctx)
	if u == nil {
		http.Error(w, "User Must Log In", http.StatusBadRequest)
		return
	}

	var settingObj UserSettings
	k := tickSettingsKey(ctx, u)
	if errDB := datastore.Get(ctx, k, settingObj); errDB != datastore.ErrNoSuchEntity {
		http.Error(w, fmt.Sprintf("User already exists: %s", errDB.Error()), 500)
		return
	}

	settingObj.HasPebble = false
	settingObj.Twitter = u.Email[:strings.Index(u.Email, "@")]

	if _, err := datastore.Put(ctx, k, &settingObj); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	jObj, errJson := json.Marshal(settingObj)
	if errJson != nil {
		http.Error(w, errJson.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jObj)
}

func createTickRule(w http.ResponseWriter, r *http.Request) {
	if strings.ToLower(r.Method) != "post" {
		http.Error(w, fmt.Sprintf("Must be POST not %s", r.Method), http.StatusBadRequest)
		return
	}

	ctx := appengine.NewContext(r)
	u := user.Current(ctx)
	if u == nil {
		http.Error(w, "User Must Log In", http.StatusBadRequest)
		return
	}

	// Setup Tick Rule
	tRule := TickRule{
		Bucket:     int64(time.Minute * 15),
		IsBucketed: true,
		RuleName:   r.FormValue("name"),
	}

	bucketStr := r.FormValue("bucket")
	switch bucketStr {
	case "hour":
		tRule.IsBucketed = true
		tRule.Bucket = int64(time.Hour * 1)
		break
	case "day":
		tRule.IsBucketed = true
		tRule.Bucket = int64(time.Hour * 24)
		break
	default:

		i, err := strconv.Atoi(bucketStr)
		if err == nil {
			tRule.IsBucketed = true
			tRule.Bucket = int64(time.Minute) * int64(i)
		}

		break
	}

	newKey := datastore.NewIncompleteKey(ctx, TICK_RULE, tickSettingsKey(ctx, u))

	fullKey, errDB := datastore.Put(ctx, newKey, &tRule)
	if errDB != nil {
		http.Error(w, errDB.Error(), 500)
		return
	}

	newData := struct {
		Type string
		Key  string
		Rule TickRule
	}{
		Type: TICK_RULE,
		Key:  fullKey.Encode(),
		Rule: tRule,
	}

	jObj, errJson := json.Marshal(newData)

	if errJson != nil {
		http.Error(w, errJson.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jObj)
}

func makeTick(w http.ResponseWriter, r *http.Request) {
	if strings.ToLower(r.Method) != "post" {
		http.Error(w, "Must be POST", http.StatusBadRequest)
		return
	}

	ctx := appengine.NewContext(r)
	u := user.Current(ctx)
	if u == nil {
		http.Error(w, "User Must Log In", http.StatusBadRequest)
		return
	}

	kStr := r.FormValue("key")
	if len(kStr) < 3 {
		http.Error(w, "Key not set", 400)
		return
	}

	// Value

	iValue, err := strconv.Atoi(r.FormValue("value"))
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// Tick Rule Key from Request
	// Get Entry Value
	t := TickEntryValue{
		When:  time.Now(),
		Value: iValue,
	}

	trKey, errKey := datastore.DecodeKey(kStr)
	if (errKey != nil) || (trKey == nil) {
		http.Error(w, errKey.Error(), 500)
		return
	}

	newKey := datastore.NewIncompleteKey(ctx, TICK_ENTRY_VALUE, trKey)

	fullKey, errDB := datastore.Put(ctx, newKey, &t)
	if errDB != nil {
		http.Error(w, errDB.Error(), 500)
		return
	}

	newData := struct {
		Type  string
		Key   string
		Entry TickEntryValue
	}{
		Type:  TICK_RULE,
		Key:   fullKey.Encode(),
		Entry: t,
	}

	jObj, errJson := json.Marshal(newData)

	if errJson != nil {
		http.Error(w, errJson.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jObj)
}
