# H182-023 フェーズ別計測

load/prepare/grammar/prefill/decodeとhostの完全な数値はmeasurements.json。包含区間を加算しない。

| case | phase | load秒 | prepare秒 | grammar秒 | generation wall秒 | helper wall秒 | user CPU秒 | system CPU秒 |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| minicpm5-int4-json_schema | bounded-grouping | 0.552 | 0.130 | 0.359 | 50.162 | 52.641 | 12.420 | 3.236 |
| minicpm5-int4-json_schema | bounded-category | 0.453 | 0.045 | 0.367 | 67.119 | 68.835 | 19.416 | 4.428 |
| minicpm5-int4-json_schema | bounded-text | 0.461 | 0.044 | 0.372 | 115.037 | 116.808 | 32.615 | 7.486 |
| minicpm5-int4-none | bounded-grouping | 0.448 | 0.043 | 0.000 | 65.819 | 67.480 | 17.779 | 4.640 |
| minicpm5-int4-none | bounded-category | 0.577 | 0.049 | 0.000 | 87.001 | 89.139 | 30.706 | 7.916 |
| minicpm5-int8-none | bounded-grouping | 1.251 | 0.088 | 0.000 | 80.968 | 83.707 | 21.826 | 5.849 |
| minicpm5-int8-none | bounded-category | 0.933 | 0.056 | 0.000 | 66.951 | 69.895 | 32.500 | 8.826 |
| minicpm5-int8-none | bounded-text | 0.621 | 0.049 | 0.000 | 124.757 | 127.230 | 62.149 | 16.372 |
| minicpm5-int8-json_schema | bounded-grouping | 0.716 | 0.069 | 0.390 | 90.547 | 93.102 | 43.050 | 11.227 |
| minicpm5-int8-json_schema | bounded-category | 0.730 | 0.055 | 0.374 | 68.747 | 71.912 | 33.447 | 8.909 |
| minicpm5-int8-json_schema | bounded-text | 0.640 | 0.047 | 0.374 | 125.185 | 128.102 | 63.077 | 16.392 |
