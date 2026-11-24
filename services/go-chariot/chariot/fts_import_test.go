package chariot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func ftsExampleRuntime(t *testing.T, path string) *Runtime {
	t.Helper()
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	rt := classRuntime()
	dotExec(t, rt, string(source))
	return rt
}

func TestFTSBusinessCoreImport(t *testing.T) {
	rt := ftsExampleRuntime(t, filepath.Join("..", "..", "..", "examples", "fts", "business-core.ch"))
	for _, name := range []string{"Organization", "Person", "Address", "Phone"} {
		value, ok := rt.currentScope.Get(name)
		if !ok {
			t.Fatalf("missing class %s", name)
		}
		if _, ok := value.(*ClassValue); !ok {
			t.Fatalf("%s is %T, not a class", name, value)
		}
	}
	for _, check := range []string{
		`equal(length(ftsModel.classes), 4)`,
		`equal(length(ftsModel.relationships), 6)`,
		`equal(length(ftsModel.frames), 5)`,
		`equal(length(ftsModel.hierarchy), 4)`,
		`equal(ftsModel.excludedInstanceFrames, 7)`,
		`equal(ftsModel.excludedInstanceEdges, 13)`,
	} {
		if got := dotExec(t, rt, check); got != Bool(true) {
			t.Fatalf("%s: %v", check, got)
		}
	}
	dotExec(t, rt, `
setq(org, createOrganization(mapValue("organization_id", "org-1", "custom", 42), mapValue()))
setq(org2, createOrganization(mapValue("organization_id", "org-2"), mapValue()))
setq(person, createPerson(mapValue("person_id", "p-1", "dob", "2000-01-01"), mapValue()))
setq(address, createAddress(mapValue("address_1", "1 Main", "owner_id", "org-1", "owner_type", "organization"), mapValue()))
setq(phone, createPhone(mapValue("phone_id", "ph-1", "owner_id", "org-1", "owner_type", "organization"), mapValue()))
`)
	for _, check := range []string{
		`equal(org.title, "New Organization")`,
		`equal(org.ftsState.status, "active")`,
		`equal(org.ftsExtraAttributes.custom, 42)`,
		`equal(person.first_name, "New")`,
		`equal(person.last_name, "Person")`,
		`equal(person.middle_name, null)`,
		`equal(address.address_id, "address-new")`,
		`equal(address.is_primary, false)`,
		`equal(address.country_code, "US")`,
		`equal(phone.can_voice, true)`,
		`equal(phone.can_sms, false)`,
		`equal(phone.can_fax, false)`,
		`validateOrganization(mapValue("organization_id", "org-1"), mapValue())`,
	} {
		if got := dotExec(t, rt, check); got != Bool(true) {
			t.Fatalf("%s: %v", check, got)
		}
	}
	dotExec(t, rt, `setProp(org.ftsState, "status", "changed")`)
	if got := dotExec(t, rt, `org2.ftsState.status`); got != Str("active") {
		t.Fatalf("shared state defaults: %v", got)
	}
	for _, check := range []struct{ source, message string }{
		{`createOrganization(mapValue(), mapValue())`, "Organization.attributes.organization_id: required"},
		{`createPerson(mapValue("person_id","p"), mapValue())`, "Person.attributes.dob: required"},
		{`createOrganization(mapValue("organization_id",null),mapValue())`, "organization_id: expected string"},
		{`createOrganization(mapValue("organization_id","o","tax_id",null),mapValue())`, "tax_id: expected string"},
		{`createOrganization(mapValue("organization_id","o","tax_id",false),mapValue())`, "tax_id: expected string"},
		{`createOrganization(mapValue("organization_id","o"),mapValue("status",1))`, "Organization.state.status: expected string"},
		{`validateOrganization(mapValue(),mapValue())`, "organization_id: required"},
		{`createOrganization(null,mapValue())`, "expected M"},
		{`createOrganization(mapValue("organization_id","o"))`, "expects 2 arguments"},
	} {
		classError(t, rt, check.source, check.message)
	}
	relationships := dotExec(t, rt, `ftsModel.relationships`).(*ArrayValue)
	found := false
	for _, value := range relationships.Elements {
		edge := value.(*MapValue)
		if edge.Values["key"] != Str("edge.organization_has_person") {
			continue
		}
		found = true
		if edge.Values["sourceClass"] != Str("Organization") || edge.Values["targetClass"] != Str("Person") {
			t.Fatalf("wrong relationship endpoints: %v", edge)
		}
		relational := edge.Values["x-relational"].(*MapValue).Values["mysql"].(*MapValue)
		payload := relational.Values["associationTable"].(*MapValue).Values["payloadColumns"].(*MapValue)
		if len(payload.Values) != 9 || payload.Values["role"] == nil {
			t.Fatalf("association payload lost: %v", payload)
		}
	}
	if !found {
		t.Fatal("organization/person relationship missing")
	}
}

func TestFTSGeneratedValidation(t *testing.T) {
	rt := ftsExampleRuntime(t, filepath.Join("..", "..", "..", "packages", "chariot-codegen", "test", "fixtures", "fts-validation.ch"))
	for _, attrs := range []string{
		`mapValue("id","s")`,
		`mapValue("id","s","count",null)`,
		`mapValue("id","s","count",0)`,
		`mapValue("id","s","count",10)`,
		`mapValue("id","s","label","雪")`,
		`mapValue("id","s","label","ab")`,
		`mapValue("id","s","mode","a","tags",array("x"))`,
		`mapValue("id","s","settings",mapValue("enabled",false))`,
		`mapValue("id","s","empty",mapValue(),"nothing",null)`,
	} {
		if got := dotExec(t, rt, `validateSample(`+attrs+`,mapValue())`); got != Bool(true) {
			t.Fatalf("%s: %v", attrs, got)
		}
		dotExec(t, rt, `createSample(`+attrs+`,mapValue())`)
	}
	for _, check := range []struct{ attrs, message string }{
		{`mapValue()`, "id: required"},
		{`mapValue("id",1)`, "id: expected string"},
		{`mapValue("id","s","count",1.5)`, "count: expected integer"},
		{`mapValue("id","s","count",-1)`, "count: minimum"},
		{`mapValue("id","s","count",11)`, "count: maximum"},
		{`mapValue("id","s","label","")`, "label: minLength"},
		{`mapValue("id","s","label","abc")`, "label: maxLength"},
		{`mapValue("id","s","label","雪雪雪")`, "label: maxLength"},
		{`mapValue("id","s","mode","c")`, "mode: not in enum"},
		{`mapValue("id","s","tags",array())`, "tags: minItems"},
		{`mapValue("id","s","tags",array("x","y","z"))`, "tags: maxItems"},
		{`mapValue("id","s","tags",array(1))`, "tags[]: expected string"},
		{`mapValue("id","s","settings",mapValue())`, "settings.enabled: required"},
		{`mapValue("id","s","settings",mapValue("enabled",null))`, "enabled: expected boolean"},
		{`mapValue("id","s","settings",mapValue("enabled",true,"extra",1))`, "settings: unknown field"},
		{`mapValue("id","s","unknown",1)`, "attributes: unknown field"},
		{`mapValue("id","s","empty",mapValue("unknown",1))`, "empty: unknown field"},
		{`mapValue("id","s","nothing",false)`, "nothing: expected null"},
	} {
		classError(t, rt, `createSample(`+check.attrs+`,mapValue())`, check.message)
		classError(t, rt, `validateSample(`+check.attrs+`,mapValue())`, check.message)
	}
	classError(t, rt, `createSample(mapValue("id","s"),mapValue("status","bad"))`, "state.status: not in enum")
}

func TestFTSErrorBuiltin(t *testing.T) {
	rt := classRuntime()
	for _, check := range []struct{ source, message string }{
		{`error("validation failed")`, "validation failed"},
		{`error()`, "requires 1 argument"},
		{`error("one","two")`, "requires 1 argument"},
		{`error(42)`, "must be a string"},
		{`setq(message,"variable message") error(message)`, "variable message"},
	} {
		classError(t, rt, check.source, check.message)
	}

	_, err := dotParse(t, `setq(after,false) error("stop") setq(after,true)`).Exec(rt)
	if err == nil || !strings.Contains(err.Error(), "stop") {
		t.Fatal(err)
	}
	if got := dotExec(t, rt, `after`); got != Bool(false) {
		t.Fatal("execution continued after error")
	}
}

func TestFTSApplyCanonicalKeys(t *testing.T) {
	rt := classRuntime()
	dotExec(t, rt, `setq(seen,mapValue())
apply(withSignature(func(key,value){setProp(seen,key,value)},signature(array("S","V"),"V")),mapValue("answer",42))
setq(indexes,array())
apply(withSignature(func(index,value){addTo(indexes,index)},signature(array("N","V"),"V")),array("a","b"))`)
	if got := dotExec(t, rt, `seen.answer`); got != Number(42) {
		t.Fatal(got)
	}
	if got := dotExec(t, rt, `getAt(indexes,1)`); got != Number(1) {
		t.Fatal(got)
	}
}
