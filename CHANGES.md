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

- [FIX] 5GB を超えるファイルのアップロード時に帯域制限がかかるように修正する
  - 帯域制限設定を行ってもマルチパートアップロードを有効にし、マルチパートアップロードの並列アップロード数を 1 つずつにすることで帯域制限を行う
  - この修正以前は、帯域制限設定を行うとマルチパートアップロードが無効となり 5GB を超えるファイルのアップロードができなかった
  - @tnamao
- [UPDATE] report ファイルアップロード後のウェブフックに `recording_metadata` を追加する
  - アップロードした report ファイルの `recording_metadata` または `metadata` の内容をウェブフックの `recording_metadata` に含めて送信する
    - セッション録画の場合は `recording_metadata` の値を使用する
    - レガシー録画の場合は `metadata` の値を使用する
    - ウェブフックに含める際のキーはセッション録画でもレガシー録画でも共通で `recording_metadata` に設定する
  - report ファイルに `recording_metadata` または `metadata` のキーが存在しない場合にはウェブフックにも `recording_metadata` を含めない
  - @tnamao
- [UPDATE] CI の staticcheck を 2024.1.1 にアップデート
  - @voluntas
- [UPDATE] go 1.23.2 にアップデート
  - @voluntas

## 2023.1.0

**祝いリリース**
