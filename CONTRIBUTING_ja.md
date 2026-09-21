# commiter-cli へのコントリビューション

[English](CONTRIBUTING.md) | 日本語

`commiter-cli` へのコントリビューションを検討していただきありがとうございます。

`commiter-cli` は、再現可能な Git state の取り扱い、ローカル限定の LLM 分析、保守的な安全境界を重視して設計しています。利便性のためにこれらの性質を弱めるのではなく、維持する変更を前提とします。

## はじめに

挙動を変更する前に、ソフトウェア要求仕様書を確認してください。

- [ソフトウェア要求仕様書 (日本語)](SOFTWARE_REQUIREMENTS_SPECIFICATION.md)
- [Software Requirements Specification (English)](SOFTWARE_REQUIREMENTS_SPECIFICATION_en.md)

Issue や Pull Request を作成する前に、既存の Issue と SRS を確認してください。大きな仕様変更や設計判断が必要な場合は、先に Issue で相談してください。

v1 の主対象は macOS 14 以降の Apple Silicon です。CLI は主に Go で実装し、実行時には system Git と Ollama を使用します。

SRS で定義された挙動へ影響すると分かっている場合は、可能な範囲で関連する仕様やドキュメントを更新し、不確かな点は Pull Request に明記してください。Pull Request を提出するために、コントリビューターが正確な FR / SR / NFR / AC ID を特定したり、英語版 / 日本語版 SRS の整合性を保証したりすることは必須ではありません。最終判断は merge 前に Maintainer が行います。

## Issue

### Issue labels

主種別ラベルは原則1つ付与します。

- `enhancement`: 新機能・実装・改善
- `bug`: 不具合修正
- `documentation`: 文書の追加・変更

必要な場合だけ、主種別に加えて次の補助ラベルを付与します。

- `good first issue`: 初参加者でも取り組みやすい Issue
- `help wanted`: 外部コントリビューターの協力を募集する Issue

主種別を判断できない場合は、無理にラベルを付与しません。ラベルの新設や既存ラベルの大幅な変更は、Issue で運用方針を確認してから行います。

### Issue templates

コントリビューター向けの Issue Form は、迷わず選べる少数の入口に限定します。

- Bug report: 再現可能な不具合や回帰の報告
- Feature / Enhancement: 具体的な問題とユースケースに基づく新機能・改善の提案
- Documentation: 不足、誤り、古い内容、不明瞭さ、翻訳差分などの報告
- Design / RFC: 大きな挙動、アーキテクチャ、互換性、仕様変更を実装前に議論するための提案
- Maintainer task: 公開テンプレートに当てはまらない Implementation、Decision、Verification、release、CI、その他の内部作業

SRS、CLI や設定の互換性、Git state の挙動、security boundary、model backend architecture、その他の横断的な契約へ大きく影響する可能性がある場合は、Design / RFC を使用してください。

GitHub の Issue 作成画面では blank issue を選べません（`blank_issues_enabled: false`）。外部コントリビューターは Bug report、Feature / Enhancement、Documentation、Design / RFC を使ってください。メンテナーの内部作業には Maintainer task フォームを使ってください。どのテンプレートにも合わない場合は、Web UI の blank issue ボタンは使わず `gh issue create` で作成してください。

SRS から生成する Implementation Issue は、対応する要件、依存、成功・失敗条件、実装方針、検証条件を本文に記載します。maintainer が作成する Decision と Verification Issue は、それぞれの目的と成果に応じた内容を記載します。一般のコントリビューターは、選択したテンプレートの項目を可能な範囲で具体的に記載してください。

セキュリティ脆弱性は公開 Issue Form から報告せず、[SECURITY.md](SECURITY.md) に従って GitHub Private Vulnerability Reporting を使用してください。

## 開発環境

必要なもの:

- Go 1.23 以降
- Git
- Ollama（Ollama に依存する挙動を開発・テストする場合）

リポジトリを clone し、依存関係を取得します。

```sh
git clone https://github.com/neural-int/commiter-cli.git
cd commiter-cli
go mod download
```

標準の確認コマンドを実行します。

```sh
gofmt -w .
go test ./...
go vet ./...
go build ./cmd/commiter
```

生成したバイナリや、変更と無関係なローカルファイルは commit しないでください。

## 変更の作成

最新の `main` から目的ごとの branch を作成し、1つの Pull Request は1つの一貫した目的に限定してください。原則として、1つの Pull Request は1つの Issue に対応させます。

既存の package 境界に沿った小さな変更を優先してください。要求された挙動に不要なリファクタリング、フォーマットだけの大規模差分、不要な依存追加は避けてください。

Go コードでは以下を守ってください。

- `gofmt` を実行する。
- 明示的な error と決定的な挙動を優先する。
- subprocess 実行時の引数境界を保持し、argv ベースの実行を shell string 実行へ置き換えない。
- 挙動変更や回帰に対する test を追加または更新する。
- repository path、Git output、hook output、verification output、LLM output などの untrusted text を terminal-safe に扱う。

## 安全性に関わる変更

以下の挙動には意図的な security invariant が含まれます。これらに触れる変更では、特に慎重な確認と対応する test が必要です。

明示的な仕様変更なしに、以下の保証を弱めないでください。

- LLM 分析に使用する repository content は loopback / local processing の外へ送信しない。
- 明確な機密ファイルは自動除外し、汎用 override で対象化できない。
- 機密候補は内容を読む前に確認する。
- 対象外 staged content と staged selection を保護する。
- verification trust は repository-scoped のままにする。
- `--no-verify` を使用せず Git hook を尊重する。
- commiter は force push、reset、stash、amend、自動 rollback を実行しない。
- 不正または安全でない LLM output によって Git mutation が発生しない。

fixture や例に実際の credentials、private key、token、`.env` の内容、その他の secret を追加しないでください。

## テスト

新しい挙動には、意味のある最小の package 境界で test を追加してください。bug fix には回帰 test を追加してください。

Pull Request を作成する前に以下を実行します。

```sh
go test ./...
go vet ./...
go build ./cmd/commiter
```

CGo または Tree-sitter integration に影響する変更では、対応する Apple Silicon build path も確認してください。

Git mutation、verification、hook、機密ファイル処理、LLM plan validation に影響する変更では、失敗経路の test も追加し、Git state が SRS で要求された状態に保たれることを確認してください。

## ドキュメント

repository 内のドキュメントへのリンクには相対リンクを使用してください。

要件、command、configuration、安全挙動、contributor workflow を変更すると分かっている場合は、可能な範囲で関連ドキュメントを更新し、不確かな点は Pull Request に明記してください。documentation の十分性と英語版 / 日本語版 SRS の意味上の整合性については、merge 前に Maintainer が最終判断します。元仕様が変更されない限り code block、configuration key、requirement ID、command name、file name、数値 threshold は翻訳間で同一にしてください。

## Pull Request

Pull Request はリポジトリ全体で1つの契約を使用します。Pull Request template と `PR Policy / pr-policy` check により、Ready for review になった時点でコントリビューター向けの機械判定可能な項目を検証します。

### Title

Pull Request title は以下を必須とします。

- 英語かつ ASCII 文字で記載する。
- `<type>(<optional-scope>): <summary>` の Conventional Commits 形式にする。
- type は `feat`、`fix`、`docs`、`test`、`refactor`、`perf`、`build`、`ci`、`chore` のいずれかを使用する。
- breaking change の場合は、該当するとき `!` を colon の前に付ける。

例:

```text
feat(cli): add JSON output
fix(git): preserve staged selection
ci: validate pull request metadata
```

### 必須 body section

Ready for review にする前に、template の必須 section を残したまま記入してください。

本文の自由記述は英語または日本語で記載できます。ただし、policy check が機械的に検証するため、section の見出し、Issue の close keyword、verification の check label、Requirements impact の checkbox label は template の表記を変更しないでください。

- `Summary`: 問題と採用したアプローチを説明する。
- `Related issue`: 原則として `Closes #123`、`Fixes #123`、`Resolves #123` のいずれかを使用する。Issue が適切でない限定的な例外では `N/A: <reason>` と理由を明示する。
- `Changes`: PR に含まれる具体的な変更を列挙する。
- `Verification`: 実行した標準 check を選択する。未選択の標準 check はすべて `Skipped / not applicable` に理由を記載する。
- `Requirements impact`: 分かる範囲で「要件への影響はないと思う」「要件へ影響する可能性がある」「判断できないため Maintainer review が必要」のいずれか1つを選択する。正確な FR / SR / NFR / AC ID の特定や、英語版 / 日本語版 SRS の整合性保証はコントリビューターの必須要件ではない。
- `Safety impact`: Git-state handling、local-only LLM processing、sensitive-file handling、verification、hook、output safety への影響を記載する。影響がない場合は `None.` と記載する。

### Release note metadata

すべての Pull Request で、template の `Release note`、`Release category`、`Breaking change` section を保持してください。カテゴリと breaking-change は、それぞれ1つだけ選択します。Release note には利用者から見た変更内容を英語で記載し、release workflow が使用する正本にします。日本語の Release note は追加しないでください。workflow が英語から生成します。

Release note に `None` を指定できるのは、カテゴリが `Internal` または `None` の場合だけです。利用者向けカテゴリ（`Added`、`Changed`、`Fixed`、`Security`、`Distribution`）では、具体的な英語の説明が必要です。Release workflow は merge 済み Pull Request の metadata を再検証し、metadata が不足または曖昧な場合は公開前に停止します。

日本語文は Google Cloud Translation Basic の `nmt` モデルで生成します。Cloud Translation API と課金を有効化した Google Cloud project を用意し、その API に制限した API key を repository secret `GOOGLE_TRANSLATE_API_KEY` に設定してください。Release workflow は英語の Release note 本文だけを翻訳 API に送信し、ソースコードや Pull Request の diff は送信しません。コード span、コマンド、version、URL、path、configuration key、Pull Request reference は保護し、翻訳後に検証します。認証情報の欠落や翻訳結果の不正があれば公開前に停止します。

Policy check は構造と明示的な確認事項だけを検証します。技術的な説明が十分かどうかは CI では判定せず、review で判断します。

merge 前に Maintainer は、SRS impact、該当する場合の Requirement ID、英語版 / 日本語版 SRS の意味上の整合性、release / breaking-change impact を最終確認します。Pull Request を merge することは、Maintainer がその変更についてこれらの確認を妥当と判断したことを意味します。

Draft Pull Request は未完成でも構いません。PR policy は Ready for review になった時点から強制します。

明示的に allowlist された automation（現在は `dependabot[bot]` と `github-actions[bot]`）は、人間向け body check の対象外です。ただし title policy は適用します。Dependabot は Conventional Commit に適合する prefix を生成するよう設定します。

### Scope、label、merge method

1つの Pull Request は1つの一貫した目的に限定してください。原則として、1つの Pull Request は1つの primary Issue に対応させます。

Pull Request label は必須にしません。changed lines や file 数による hard limit も設けません。scope の良し悪しは行数ではなく、責務と目的の一貫性で判断します。

Pull Request の merge method は squash merge を使用します。検証済みの Pull Request title が、`main` に追加される commit title の基礎になります。

review 指摘への対応も同じく焦点を維持し、正確性や安全性のために必要でない限り、指摘範囲を超えて変更を拡大しないでください。

## Commit Message

可能な範囲で Conventional Commits を使用してください。

```text
feat(cli): add command
fix(git): preserve staged state
docs: add English SRS
test(safety): cover sensitive-file exclusion
```

各 commit は内部的に一貫した目的へまとめ、無関係な変更を混在させないでください。

## License と Conduct

コントリビューションは、このリポジトリに適用される [MIT License](LICENSE) の下で提供されるものとします。

すべての参加者は [Code of Conduct (行動規範)](CODE_OF_CONDUCT_ja.md) に従ってください。セキュリティ脆弱性は公開 Issue や Discussion ではなく、[Security Policy](SECURITY.md) に従って報告してください。
