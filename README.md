# Sora Archive Uploader

<!-- [![GitHub tag (latest SemVer)](https://img.shields.io/github/tag/shiguredo/sora-archive-uploader.svg)](https://github.com/shiguredo/sora-archive-uploader) -->

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

<!-- [![Actions Status](https://github.com/shiguredo/sora-archive-uploader/actions/workflows/ci.yml/badge.svg?branch=develop)](https://github.com/shiguredo/sora_exporter/actions/workflows/ci.yml) -->

## About Shiguredo's open source software

We will not respond to PRs or issues that have not been discussed on Discord. Also, Discord is only available in Japanese.

Please read https://github.com/shiguredo/oss/blob/master/README.en.md before use.

## 時雨堂のオープンソースソフトウェアについて

利用前に https://github.com/shiguredo/oss をお読みください。

## Sora Archive Uploader について

Sora が出力する録画関連のファイルを S3 または S3 互換オブジェクトストレージにアップロードするツールです。
systemd タイマーユニットを利用しての定期実行を想定しています。

[Sora Cloud](https://sora-cloud.shiguredo.jp/) で実際に利用している仕組みからツールとして切り出して公開しています。

## 目的

Sora は録画を行った場合、録画ファイルを WebM 、録画メタデータ JSON ファイルで出力します。
Sora Cloud では出力されたファイルをオブジェクトストレージにアップロードする仕組みが必要となり開発しました。

## 特徴

- systemd の設定だけで利用できます
- 並列でオブジェクトストレージにアップロードできます
- アップロード完了時に指定された URL にウェブフックリクエストを通知します
- ウェブフックにはベーシック認証や mTLS が利用可能です
- アップロードに失敗した場合は設定ファイルで指定した隔離ディレクトリに移動します
- アップロードの帯域制限を設定できます

### 対応オブジェクトストレージ

- AWS S3
- MinIO
- GCP GCS
- Vultr Object Storage
- Linode Object Storage
- DigitalOcean Spaces
- Cloudflare R2

## まずは使ってみる

config.ini に必要な情報を設定してください。

```bash
$ cp config_example.ini config.ini
```

make でビルドして実行します。

```bash
$ make
$ ./bin/sora-archive-uploader -C config.ini
```

## Discord

最新の状況などは Discord で共有しています。質問や相談も Discord でのみ受け付けています。

https://discord.gg/shiguredo

## 有償での優先実装が可能な機能一覧

**詳細は Discord またはメールにてお問い合わせください**

- オープンソースでの公開が前提
- 可能であれば企業名の公開
  - 公開が難しい場合は `企業名非公開` と書かせていただきます

### 機能

- [Amazon S3 SSE-S3](https://docs.aws.amazon.com/ja_jp/AmazonS3/latest/userguide/UsingServerSideEncryption.html) への対応
- [Azure Blob Storage](https://azure.microsoft.com/ja-jp/products/storage/blobs/) への対応

## ライセンス

```
Copyright 2022-2023, Takeshi Namao (Original Author)
Copyright 2022-2023, Shiguredo Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```
