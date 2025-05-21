# 変更履歴

- CHANGE
  - 下位互換のない変更
- UPDATE
  - 下位互換がある変更
- ADD
  - 下位互換がある追加
- FIX
  - バグ修正

## develop

- [CHANGE] アップロードする録画データのファイルを `.webm` だけではなく `.mp4` のスクレイピングにも対応する
  - ログのメッセージや出力する際のパラメータ名、プログラム中の変数名や関数名も webm を使用している箇所は media に変更する
  - @tnamao
- [UPDATE] 設定に `exclude_webhook_recording_metadata` を追加し、report ファイルアップロード後のウェブフックに `recording_metadata` を含めるかどうか設定できるようにする
  - デフォルトは `false` で `recording_metadata` を送信するウェブフックに含める
  - `true` を設定するとレポートファイルに `recording_metadata` または `metadata` が含まれていてもウェブフックには含めない
- [UPDATE] report ファイルアップロード後のウェブフックに `recording_metadata` を追加する
  - アップロードした report ファイルの `recording_metadata` または `metadata` の内容をウェブフックの `recording_metadata` に含めて送信する
    - セッション録画の場合は `recording_metadata` の値を使用する
    - レガシー録画の場合は `metadata` の値を使用する
    - ウェブフックに含める際のキーはセッション録画でもレガシー録画でも共通で `recording_metadata` に設定する
  - report ファイルに `recording_metadata` または `metadata` のキーが存在しない場合にはウェブフックにも `recording_metadata` を含めない
  - @tnamao
- [ADD] linux arm64 向けリリースバイナリを追加
  - @voluntas
- [FIX] ウェブフック送信時の Response.Body のクローズ漏れを修正する
  - @tnamao
- [FIX] 5GB を超えるファイルのアップロード時に帯域制限がかかるように修正する
  - 帯域制限設定を行ってもマルチパートアップロードを有効にし、マルチパートアップロードの並列アップロード数を 1 つずつにすることで帯域制限を行う
  - この修正以前は、帯域制限設定を行うとマルチパートアップロードが無効となり 5GB を超えるファイルのアップロードができなかった
  - @tnamao

### misc

- [CHANGE] GitHub Actions の ubuntu-latest を ubuntu-24.04 に変更
  - @voluntas
- [UPDATE] CI の staticcheck を 2025.1.1 にアップデート
  - @tnamao
- [UPDATE] go 1.24.3 にアップデート
  - @tnamao
- [UPDATE] 依存ライブラリを更新
  - `minio/minio-go` 7.0.79 => 7.0.92
  - `rs/zerolog` 1.33.0 => 1.34.0
  - @tnamao

## 2023.1.0

**祝いリリース**
