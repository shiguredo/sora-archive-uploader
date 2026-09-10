# オブジェクトストレージへのアップロード先パス (プレフィックス) を設定できるようにする

- Created: 2026-09-10
- Completed: {YYYY-MM-DD}
- Branch: feature/add-s3-object-key-prefix
- Polished: {YYYY-MM-DD}
- Reporter: @voluntas

## 目的

オブジェクトストレージにアップロードするファイルのオブジェクトキーに、設定で指定した任意のパス (プレフィックス) を付けられるようにする。バケット直下に `recording_id` が並ぶ現在の構成では、環境・テナント・用途ごとにオブジェクトをディレクトリ状に分割して保存できない。

## 現状

オブジェクトキーは 4 箇所で `<recording_id>/<ファイル名>` の形式に固定されている。

- `uploader.go` の `handleArchive` 内の `metadataObjectKey` と `mediaObjectKey`
- `uploader.go` の `handleReport` 内の `reportObjectKey`
- `uploader.go` の `handleArchiveEnd` 内の `objectKey`

いずれも `fmt.Sprintf("%s/%s", <recording_id>, <ファイル名>)` で生成しており、先頭に付与できるのは `recording_id` だけになっている。

```go
metadataObjectKey := fmt.Sprintf("%s/%s", am.RecordingID, metadataFilename)
mediaObjectKey := fmt.Sprintf("%s/%s", am.RecordingID, mediaFilename)
reportObjectKey := fmt.Sprintf("%s/%s", rr.RecordingID, filename)
objectKey := fmt.Sprintf("%s/%s", aem.RecordingID, filename)
```

`config.go` の `Config` にも `config_example.ini` にも、アップロード先のパスを指定する設定項目は存在しない。

## 設計方針

- `config.go` の `Config` に `object_storage_path` を追加し、`ObjectStoragePath string` として読み込む。
- オブジェクトキー生成を 1 つの関数 (または `Uploader` のメソッド) に集約し、先頭に `object_storage_path` を付与する。現在 4 箇所に散っている `fmt.Sprintf` をこの関数経由に統一する。
- S3 のオブジェクトキーはスラッシュ区切りであるため、パスの結合には `path.Join` を使う (`filepath.Join` は OS 依存の区切り文字になるため使わない)。
- `object_storage_path` が空文字のときは従来どおり `<recording_id>/<ファイル名>` とし、後方互換を保つ。
- 指定値の先頭・末尾の `/` の扱い (許容するか正規化するか) を決め、`config_example.ini` と README に明記する。
- バケット名 (`object_storage_bucket_name`) はバケットそのものの指定であり、オブジェクトキーのプレフィックスとは別物として扱う。

## 完了条件

- `object_storage_path` を設定すると、アップロードされるすべてのオブジェクトキーの先頭に指定したパスが付与される
- `object_storage_path` を未設定 (空文字) にすると、従来と同じオブジェクトキーになる
- `config_example.ini` に `object_storage_path` の設定例と説明が追加されている
- 設定の仕様が README に記載されている
- `go build ./...` と `go test ./...` が通る

## 解決方法

1. `config.go` の `Config` に `object_storage_path` に対応する `ObjectStoragePath` フィールドを追加する
2. `object_storage_path` と `recording_id` とファイル名からオブジェクトキーを組み立てる共通関数を追加する
3. `handleArchive` / `handleReport` / `handleArchiveEnd` の 4 箇所を共通関数経由に置き換える
4. `config_example.ini` と README に `object_storage_path` を追記する
5. `go build ./...` と `go test ./...` で確認する
