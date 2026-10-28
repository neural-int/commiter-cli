# 構造化出力ハーネス

研究用設定に次のいずれかを指定する。

```json
{"output_constraint":"json_schema"}
```

```json
{"output_constraint":"none"}
```

両方で同一のsystem指示を使用する。noneは生成時の文法マスクだけを外す。生成後のホスト検証は保持する。詳細・実施範囲はgoal.md。

## 準備・非推論監査

以下の絶対pathは利用環境に合わせる。outは新規ディレクトリを指定し、過去の測定結果を上書きしない。

```sh
python3 tools/benchmark182/structured_output.py prepare --out /absolute/private/study
python3 tools/benchmark182/structured_assets.py --study /absolute/private/study
python3 tools/benchmark182/structured_output.py audit \
  --out /absolute/private/study --helper /absolute/helper \
  --requests /absolute/private/captured-requests
python3 tools/benchmark182/structured_model_audit.py \
  --study /absolute/private/study --helper /absolute/helper \
  --requests /absolute/private/captured-requests
```

captured-requestsは既存E2Eで収集したbounded-grouping.json、bounded-category.json、bounded-text.jsonを含む。auditはモデルをロードしない。structured_model_auditは重みをロードしてtokenizer・予定cacheを検査するが生成はしない。モデル監査の出力ディレクトリは再利用しない。

## 次回の実推論への接続

`tools/benchmark182/structured_output.py`は実行可能なhelperプロキシでもある。mlx.ClientのExecutableにその絶対pathを指定し、次の環境変数でケースを選ぶ。

```sh
export COMMITER_STUDY_HELPER=/absolute/release/commiter-mlx-helper
export COMMITER_STUDY_CONFIG=/absolute/private/study/qwen4-on-json_schema.json
export COMMITER_STUDY_MODEL_PATH=/absolute/private/study/models/Qwen3.5-4B-4bit
```

このプロキシを実行すると推論が始まる。prepare/auditでは実行しない。ホストが実際のGrouping応答を検証してからCategoryへ渡し、Category結果をTextへ渡す通常の順序を使う。保存済みCategory/Text要求の再生を、新たな実Grouping込みE2Eと呼ばない。ホスト側のモデル選択制約・cycle timeout・履歴取得予算などの研究用設定は、次の測定契約で別途固定する。現在の通常CLIのproduction設定を変更するものではない。

## ファイル数上限なしの検証ビルド

```sh
python3 tools/benchmark182/build.py /absolute/private/unlimited-build --unlimited-files
```

今回の5ファイル・3目的fixtureには、このビルドのevaluation.testを使用する。
Go側Plannerと履歴E2Eテストのファイル数による拒否をoverlayで取り除く。
入力容量、IPC、validator、120秒cycle timeoutなどは、このオプションでは変更しない。
既存の引数なしビルドは過去の16ファイル条件を保持する。
