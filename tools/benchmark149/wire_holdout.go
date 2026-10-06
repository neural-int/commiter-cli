package main

// Fixed before first inference: two distinct wire contracts change in both
// their producer and consumer. Neither selects or calls the other's source.
func wireHoldout() fixture {
	a := sourceCase("events/writer/encode.go", "writer", "import \"encoding/json\"", `func Emit(id string) string { b,_:=json.Marshal(map[string]string{"trace_id":id}); return string(b) }`, `func Emit(id string) string { b,_:=json.Marshal(map[string]string{"correlation_id":id}); return string(b) }`)
	at := testCase("events/writer/encode_test.go", "writer", "EventField", `if Emit("x")!="{\"trace_id\":\"x\"}" {t.Fatal("event field")}`, `if Emit("x")!="{\"correlation_id\":\"x\"}" {t.Fatal("event field")}`)
	b := sourceCase("events/reader/decode.go", "reader", "import \"encoding/json\"", `func Read(s string) string { var v map[string]string; json.Unmarshal([]byte(s),&v); return v["trace_id"] }`, `func Read(s string) string { var v map[string]string; json.Unmarshal([]byte(s),&v); return v["correlation_id"] }`)
	bt := testCase("events/reader/decode_test.go", "reader", "EventRead", `if Read("{\"trace_id\":\"x\"}")!="x" {t.Fatal("event read")}`, `if Read("{\"correlation_id\":\"x\"}")!="x" {t.Fatal("event read")}`)
	c := sourceCase("metrics/writer/encode.go", "writer", "import \"encoding/json\"", `func EmitMicros(n int64) string { b,_:=json.Marshal(map[string]int64{"elapsed_us":n}); return string(b) }`, `func EmitMicros(n int64) string { b,_:=json.Marshal(map[string]int64{"elapsed_ms":n/1000}); return string(b) }`)
	ct := testCase("metrics/writer/encode_test.go", "writer", "DurationUnit", `if EmitMicros(2000)!="{\"elapsed_us\":2000}" {t.Fatal("duration unit")}`, `if EmitMicros(2000)!="{\"elapsed_ms\":2}" {t.Fatal("duration unit")}`)
	d := sourceCase("metrics/reader/decode.go", "reader", "import \"encoding/json\"", `func Micros(s string) int64 { var v map[string]int64; json.Unmarshal([]byte(s),&v); return v["elapsed_us"] }`, `func Micros(s string) int64 { var v map[string]int64; json.Unmarshal([]byte(s),&v); return v["elapsed_ms"]*1000 }`)
	dt := testCase("metrics/reader/decode_test.go", "reader", "DurationRead", `if Micros("{\"elapsed_us\":2000}")!=2000 {t.Fatal("duration read")}`, `if Micros("{\"elapsed_ms\":2}")!=2000 {t.Fatal("duration read")}`)
	return fixtureFromCases("holdout-event-field-and-duration-unit-8", []changeCase{a, at, b, bt, c, ct, d, dt}, [][]int{{0, 1, 2, 3}, {4, 5, 6, 7}})
}
