package yapi

import "strings"

// nameMock is one naming convention and the mock it suggests.
type nameMock struct {
	matches func(name string) bool
	mock    string
}

func containsAny(words ...string) func(string) bool {
	return func(name string) bool {
		for _, w := range words {
			if strings.Contains(name, w) {
				return true
			}
		}
		return false
	}
}

// nameMocks are EasyYapi's naming conventions for parameter examples, tried in
// order on the lower-cased name; the first that matches wins. They are
// substring tests, as EasyYapi's are, so `usageStatus` holds "age" and
// `rows` matches nothing.
var nameMocks = []nameMock{
	{containsAny("email"), "@email"},
	{containsAny("phone", "mobile", "tel"), "@phone"},
	{containsAny("url", "link", "website"), "@url"},
	{containsAny("uuid", "guid"), "@uuid"},
	{func(n string) bool {
		return strings.Contains(n, "name") && containsAny("user", "first", "last")(n)
	}, "@cname"},
	{containsAny("name"), "@string"},
	{containsAny("address"), "@county(true)"},
	{containsAny("city"), "@city"},
	{containsAny("country"), "@county"},
	{containsAny("province", "state"), "@province"},
	{containsAny("zip", "postal"), "@zip"},
	{containsAny("ip"), "@ip"},
	{func(n string) bool { return strings.Contains(n, "id") && !strings.Contains(n, "identity") }, "@id"},
	{func(n string) bool { return strings.Contains(n, "date") && strings.Contains(n, "birth") }, "@date('yyyy-MM-dd')"},
	{containsAny("date", "time"), "@datetime"},
	{containsAny("age"), "@integer(0, 120)"},
	{containsAny("price", "amount", "money"), "@float(0, 10000, 2, 2)"},
	{containsAny("count", "num", "number", "qty", "quantity"), "@integer(0, 100)"},
	{containsAny("password", "pwd"), "******"},
	{containsAny("token", "key"), "@string(32)"},
	{containsAny("image", "img", "avatar", "photo"), "@image"},
	{containsAny("desc", "description", "content", "remark", "comment"), "@cparagraph"},
	{containsAny("title"), "@ctitle"},
	{containsAny("code"), "@string(6)"},
}

// parameterMock is the example EasyYapi gives a query or path parameter that
// declares none: a mock suggested by its name, or @string. A parameter is text
// on the wire, so its declared type plays no part.
func parameterMock(name string) string {
	lower := strings.ToLower(name)
	for _, m := range nameMocks {
		if m.matches(lower) {
			return m.mock
		}
	}
	return "@string"
}
