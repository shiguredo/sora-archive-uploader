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

- [UPDATE] report ファイルアップロード後のウェブフックに `recording_metadata` を追加する
  - アップロードした report ファイルの `recording_metadata` の内容をウェブフックに含めて送信する
  - report ファイルに `recording_metadata` のキーが存在しない場合にはウェブフックにも `recording_metadata` を含めない
  - @tnamao
- [UPDATE] CI の staticcheck を 2024.1.1 にアップデート
  - @voluntas
- [UPDATE] go 1.23.2 にアップデート
  - @voluntas

## 2023.1.0

**祝いリリース**
