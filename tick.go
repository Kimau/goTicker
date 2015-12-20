package tickremind

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
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

//---------------------------------------- Setup Functions

func init() {
	http.HandleFunc("/", root)
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

//---------------------------------------- API Funcs

func root(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello, world!")
}

func createTickUser(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	u := user.Current(ctx)
	if u == nil {
		http.Error(w, "User Must Log In", http.StatusBadRequest)
		return
	}

	var settingObj UserSettings
	if errDB := datastore.Get(ctx, tickSettingsKey(ctx, u), settingObj); errDB != datastore.ErrNoSuchEntity {
		http.Error(w, fmt.Sprintf("User already exists: %s", errDB.Error()), 500)
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
	case "hourly":
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
	ctx := appengine.NewContext(r)
	u := user.Current(ctx)
	if u == nil {
		http.Error(w, "User Must Log In", http.StatusBadRequest)
		return
	}

	kStr := r.URL.Query().Get("key")
	if len(kStr) < 3 {
		http.Error(w, "Key not set", 400)
		return
	}

	// Tick Rule Key from Request
	// Get Entry Value
	t := TickEntryValue{
		When:  time.Now(),
		Value: 1,
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
