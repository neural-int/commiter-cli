# #143 人間確認用の可視入力資料（2026-09-30）

対象は前回holdout8例。以下はJSON decode後にpromptで見えるfile内容・relation context・候補partition。gold、rationale、前回モデル結果は含めていない。全文promptの保存ではなく、合成入力の証拠抜粋である。

共通の判断条件: independent change purposeでcomplete partitionを1つ選ぶ。path/test/importの一致だけでは同目的の証明としない。各例の回答は「C001／C002／一意に区別できない」とし、可視情報の根拠・反証・不明点を書く。過去gold等を見ている場合は非盲検と申告する。

## h143_source_test_independent

### F001: src/search/filter.go（raw_diff）

```diff
@@ -1,1 +1,2 @@
+func HideArchivedResults(archived bool) bool { return archived }
```
### F002: src/search/filter_test.go（raw_diff）

```diff
@@ -1,1 +1,2 @@
-const legacySnapshotTitle = "old filter behavior"
+const legacySnapshotTitle = "legacy query compatibility"
```

可視relation context（構造ヒント）:
```json
{
  "candidate_components": [
    {
      "id": "C001",
      "file_ids": [
        "F001",
        "F002"
      ]
    }
  ],
  "relations": [
    {
      "source_id": "F002",
      "target_id": "F001",
      "kind": "source_test",
      "class": "soft",
      "reason": "matching_test_path",
      "evidence": {
        "type": "test_source_paths",
        "value": "src/search/filter_test.go",
        "related": "src/search/filter.go"
      }
    }
  ],
  "auxiliary_hints": [
    {
      "kind": "path_proximity",
      "evidence": {
        "type": "directory",
        "value": "src/search"
      },
      "file_ids": [
        "F001",
        "F002"
      ]
    }
  ],
  "statistics": {
    "node_count": 2,
    "edge_count": 1,
    "edges_by_kind": {
      "source_test": 1
    },
    "observation_count": 0,
    "observations_by_kind": {},
    "observations_by_outcome": {},
    "component_count": 1,
    "dense_pair_baseline": 1,
    "candidate_pair_count": 1,
    "reduced_pair_count": 0,
    "auxiliary_hint_count": 1
  }
}
```

候補:
```json
[
  {
    "id": "C001",
    "groups": [
      [
        "F001",
        "F002"
      ]
    ]
  },
  {
    "id": "C002",
    "groups": [
      [
        "F001"
      ],
      [
        "F002"
      ]
    ]
  }
]
```

回答: 未記入。根拠・反証・不明点: 未記入。

## h143_stem_docs_independent

### F001: src/upload/limit.go（raw_diff）

```diff
@@ -1,1 +1,2 @@
-func MaxUploadBytes() int { return 1024 }
+func MaxUploadBytes() int { return 2048 }
```
### F002: docs/upload/limit.md（raw_diff）

```diff
@@ -1,1 +1,2 @@
-Support mailbox: support-old@example.invalid
+Support mailbox: helpdesk@example.invalid
```

候補:
```json
[
  {
    "id": "C001",
    "groups": [
      [
        "F001"
      ],
      [
        "F002"
      ]
    ]
  },
  {
    "id": "C002",
    "groups": [
      [
        "F001",
        "F002"
      ]
    ]
  }
]
```

回答: 未記入。根拠・反証・不明点: 未記入。

## h143_shared_directory_independent

### F001: src/ops/compress.go（raw_diff）

```diff
@@ -1,1 +1,2 @@
-func CompressionLevel() int { return 1 }
+func CompressionLevel() int { return 6 }
```
### F002: src/ops/locale.go（raw_diff）

```diff
@@ -1,1 +1,2 @@
-func DefaultLocale() string { return "en" }
+func DefaultLocale() string { return "ja" }
```

可視relation context（構造ヒント）:
```json
{
  "candidate_components": [
    {
      "id": "C001",
      "file_ids": [
        "F001"
      ]
    },
    {
      "id": "C002",
      "file_ids": [
        "F002"
      ]
    }
  ],
  "relations": null,
  "auxiliary_hints": [
    {
      "kind": "path_proximity",
      "evidence": {
        "type": "directory",
        "value": "src/ops"
      },
      "file_ids": [
        "F001",
        "F002"
      ]
    }
  ],
  "statistics": {
    "node_count": 2,
    "edge_count": 0,
    "edges_by_kind": {},
    "observation_count": 0,
    "observations_by_kind": {},
    "observations_by_outcome": {},
    "component_count": 2,
    "dense_pair_baseline": 1,
    "candidate_pair_count": 0,
    "reduced_pair_count": 1,
    "pair_reduction_reason": "no_evidence_path_between_candidate_components",
    "auxiliary_hint_count": 1
  }
}
```

候補:
```json
[
  {
    "id": "C001",
    "groups": [
      [
        "F001",
        "F002"
      ]
    ]
  },
  {
    "id": "C002",
    "groups": [
      [
        "F001"
      ],
      [
        "F002"
      ]
    ]
  }
]
```

回答: 未記入。根拠・反証・不明点: 未記入。

## h143_two_features_interleaved

### F001: src/checkout/discount.go（raw_diff）

```diff
@@ -1,1 +1,2 @@
+func ApplyMemberDiscount(total int) int { return total * 90 / 100 }
```
### F002: src/monitor/health.go（raw_diff）

```diff
@@ -1,1 +1,2 @@
+func HealthStatus() string { return "ready" }
```
### F003: src/checkout/discount_test.go（raw_diff）

```diff
@@ -1,1 +1,2 @@
+func TestMemberDiscount(t *testing.T) { if ApplyMemberDiscount(100) != 90 { t.Fatal("member discount") } }
```
### F004: docs/monitor/readiness.md（raw_diff）

```diff
@@ -1,1 +1,2 @@
+The readiness endpoint now returns ready when the service can accept requests.
```

可視relation context（構造ヒント）:
```json
{
  "candidate_components": [
    {
      "id": "C001",
      "file_ids": [
        "F001",
        "F003"
      ]
    },
    {
      "id": "C002",
      "file_ids": [
        "F002"
      ]
    },
    {
      "id": "C003",
      "file_ids": [
        "F004"
      ]
    }
  ],
  "relations": [
    {
      "source_id": "F003",
      "target_id": "F001",
      "kind": "source_test",
      "class": "soft",
      "reason": "matching_test_path",
      "evidence": {
        "type": "test_source_paths",
        "value": "src/checkout/discount_test.go",
        "related": "src/checkout/discount.go"
      }
    }
  ],
  "auxiliary_hints": [
    {
      "kind": "path_proximity",
      "evidence": {
        "type": "directory",
        "value": "src/checkout"
      },
      "file_ids": [
        "F001",
        "F003"
      ]
    }
  ],
  "statistics": {
    "node_count": 4,
    "edge_count": 1,
    "edges_by_kind": {
      "source_test": 1
    },
    "observation_count": 0,
    "observations_by_kind": {},
    "observations_by_outcome": {},
    "component_count": 3,
    "dense_pair_baseline": 6,
    "candidate_pair_count": 1,
    "reduced_pair_count": 5,
    "pair_reduction_reason": "no_evidence_path_between_candidate_components",
    "auxiliary_hint_count": 1
  }
}
```

候補:
```json
[
  {
    "id": "C001",
    "groups": [
      [
        "F001",
        "F003"
      ],
      [
        "F002",
        "F004"
      ]
    ]
  },
  {
    "id": "C002",
    "groups": [
      [
        "F001",
        "F002",
        "F003",
        "F004"
      ]
    ]
  }
]
```

回答: 未記入。根拠・反証・不明点: 未記入。

## h143_atomic_validation

### F001: src/accounts/handle.go（raw_diff）

```diff
@@ -1,1 +1,2 @@
+func ValidHandle(s string) bool { return len(s) >= 3 && len(s) <= 20 }
```
### F002: src/accounts/handle_test.go（raw_diff）

```diff
@@ -1,1 +1,2 @@
+func TestHandleLength(t *testing.T) { if ValidHandle("ab") || !ValidHandle("abc") { t.Fatal("handle length") } }
```
### F003: docs/accounts/handle.md（raw_diff）

```diff
@@ -1,1 +1,2 @@
+Account handles must contain between 3 and 20 characters.
```

可視relation context（構造ヒント）:
```json
{
  "candidate_components": [
    {
      "id": "C001",
      "file_ids": [
        "F001",
        "F002"
      ]
    },
    {
      "id": "C002",
      "file_ids": [
        "F003"
      ]
    }
  ],
  "relations": [
    {
      "source_id": "F002",
      "target_id": "F001",
      "kind": "source_test",
      "class": "soft",
      "reason": "matching_test_path",
      "evidence": {
        "type": "test_source_paths",
        "value": "src/accounts/handle_test.go",
        "related": "src/accounts/handle.go"
      }
    }
  ],
  "auxiliary_hints": [
    {
      "kind": "path_proximity",
      "evidence": {
        "type": "directory",
        "value": "src/accounts"
      },
      "file_ids": [
        "F001",
        "F002"
      ]
    }
  ],
  "statistics": {
    "node_count": 3,
    "edge_count": 1,
    "edges_by_kind": {
      "source_test": 1
    },
    "observation_count": 0,
    "observations_by_kind": {},
    "observations_by_outcome": {},
    "component_count": 2,
    "dense_pair_baseline": 3,
    "candidate_pair_count": 1,
    "reduced_pair_count": 2,
    "pair_reduction_reason": "no_evidence_path_between_candidate_components",
    "auxiliary_hint_count": 1
  }
}
```

候補:
```json
[
  {
    "id": "C001",
    "groups": [
      [
        "F001"
      ],
      [
        "F002"
      ],
      [
        "F003"
      ]
    ]
  },
  {
    "id": "C002",
    "groups": [
      [
        "F001",
        "F002",
        "F003"
      ]
    ]
  }
]
```

回答: 未記入。根拠・反証・不明点: 未記入。

## h143_crossdir_documentation

### F001: src/storage/prune.go（raw_diff）

```diff
@@ -1,1 +1,2 @@
+func PruneExpiredItems(ageDays int) bool { return ageDays > 14 }
```
### F002: docs/operators/cleanup.md（raw_diff）

```diff
@@ -1,1 +1,2 @@
+Items older than fourteen days are removed automatically by the storage cleanup job.
```

候補:
```json
[
  {
    "id": "C001",
    "groups": [
      [
        "F001",
        "F002"
      ]
    ]
  },
  {
    "id": "C002",
    "groups": [
      [
        "F001"
      ],
      [
        "F002"
      ]
    ]
  }
]
```

回答: 未記入。根拠・反証・不明点: 未記入。

## h143_paraphrased_feature

### F001: src/identity/recover.go（raw_diff）

```diff
@@ -1,1 +1,2 @@
+func RecoveryLinkExpires(minutes int) bool { return minutes >= 30 }
```
### F002: docs/security/credential_reset.md（raw_diff）

```diff
@@ -1,1 +1,2 @@
+A credential reset URL becomes unusable half an hour after it is issued.
```

候補:
```json
[
  {
    "id": "C001",
    "groups": [
      [
        "F001"
      ],
      [
        "F002"
      ]
    ]
  },
  {
    "id": "C002",
    "groups": [
      [
        "F001",
        "F002"
      ]
    ]
  }
]
```

回答: 未記入。根拠・反証・不明点: 未記入。

## h143_crosscomponent_feature

### F001: src/api/pagination.go（raw_diff）

```diff
@@ -1,1 +1,2 @@
+const DefaultPageSize = 25
```
### F002: src/web/results.go（raw_diff）

```diff
@@ -1,1 +1,2 @@
+func ResultPageSize() int { return api.DefaultPageSize }
```
### F003: docs/api/paging.md（raw_diff）

```diff
@@ -1,1 +1,2 @@
+API and web result pages default to 25 entries per page.
```

候補:
```json
[
  {
    "id": "C001",
    "groups": [
      [
        "F001",
        "F002",
        "F003"
      ]
    ]
  },
  {
    "id": "C002",
    "groups": [
      [
        "F001"
      ],
      [
        "F002"
      ],
      [
        "F003"
      ]
    ]
  }
]
```

回答: 未記入。根拠・反証・不明点: 未記入。
