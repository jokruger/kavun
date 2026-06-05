package stdlib

import "testing"

func TestJSON(t *testing.T) {
	module(t, "json").call(rta, "encode", 5).expect(rta, []byte("5"))
	module(t, "json").call(rta, "encode", "foobar").expect(rta, []byte(`"foobar"`))
	module(t, "json").call(rta, "encode", MAP{"foo": 5}).expect(rta, []byte("{\"foo\":5}"))
	module(t, "json").call(rta, "encode", IMAP{"foo": 5}).expect(rta, []byte("{\"foo\":5}"))
	module(t, "json").call(rta, "encode", ARR{1, 2, 3}).expect(rta, []byte("[1,2,3]"))
	module(t, "json").call(rta, "encode", IARR{1, 2, 3}).expect(rta, []byte("[1,2,3]"))
	module(t, "json").call(rta, "encode", MAP{"foo": "bar"}).expect(rta, []byte("{\"foo\":\"bar\"}"))
	module(t, "json").call(rta, "encode", MAP{"foo": 1.8}).expect(rta, []byte("{\"foo\":1.8}"))
	module(t, "json").call(rta, "encode", MAP{"foo": true}).expect(rta, []byte("{\"foo\":true}"))
	module(t, "json").call(rta, "encode", MAP{"foo": '8'}).expect(rta, []byte("{\"foo\":56}"))
	module(t, "json").call(rta, "encode", MAP{"foo": []byte("foo")}).expect(rta, []byte("{\"foo\":\"Zm9v\"}")) // json encoding returns []byte as base64 encoded string
	module(t, "json").call(rta, "encode", MAP{"foo": ARR{"bar", 1, 1.8, '8', true}}).expect(rta, []byte("{\"foo\":[\"bar\",1,1.8,56,true]}"))
	module(t, "json").call(rta, "encode", MAP{"foo": IARR{"bar", 1, 1.8, '8', true}}).expect(rta, []byte("{\"foo\":[\"bar\",1,1.8,56,true]}"))
	module(t, "json").call(rta, "encode", MAP{"foo": ARR{ARR{"bar", 1}, ARR{"bar", 1}}}).expect(rta, []byte("{\"foo\":[[\"bar\",1],[\"bar\",1]]}"))
	module(t, "json").call(rta, "encode", MAP{"foo": MAP{"string": "bar"}}).expect(rta, []byte("{\"foo\":{\"string\":\"bar\"}}"))
	module(t, "json").call(rta, "encode", MAP{"foo": IMAP{"string": "bar"}}).expect(rta, []byte("{\"foo\":{\"string\":\"bar\"}}"))
	module(t, "json").call(rta, "encode", MAP{"foo": MAP{"map1": MAP{"string": "bar"}}}).expect(rta, []byte("{\"foo\":{\"map1\":{\"string\":\"bar\"}}}"))
	module(t, "json").call(rta, "encode", ARR{ARR{"bar", 1}, ARR{"bar", 1}}).expect(rta, []byte("[[\"bar\",1],[\"bar\",1]]"))

	module(t, "json").call(rta, "decode", `5`).expect(rta, 5)
	module(t, "json").call(rta, "decode", `"foo"`).expect(rta, "foo")
	module(t, "json").call(rta, "decode", `[1,2,3,"bar"]`).expect(rta, ARR{1, 2, 3, "bar"})
	module(t, "json").call(rta, "decode", `{"foo":5}`).expect(rta, MAP{"foo": 5})
	module(t, "json").call(rta, "decode", `{"foo":2.5}`).expect(rta, MAP{"foo": 2.5})
	module(t, "json").call(rta, "decode", `{"foo":true}`).expect(rta, MAP{"foo": true})
	module(t, "json").call(rta, "decode", `{"foo":"bar"}`).expect(rta, MAP{"foo": "bar"})
	module(t, "json").call(rta, "decode", `{"foo":[1,2,3,"bar"]}`).expect(rta, MAP{"foo": ARR{1, 2, 3, "bar"}})

	module(t, "json").call(rta, "indent", []byte("{\"foo\":[\"bar\",1,1.8,56,true]}"), "", "  ").expect(rta, []byte(`{
  "foo": [
    "bar",
    1,
    1.8,
    56,
    true
  ]
}`))

	module(t, "json").call(rta, "html_escape", []byte(`{"M":"<html>foo &`+"\xe2\x80\xa8 \xe2\x80\xa9"+`</html>"}`)).
		expect(rta, []byte(`{"M":"\u003chtml\u003efoo \u0026\u2028 \u2029\u003c/html\u003e"}`))
}
