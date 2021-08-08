<?php
/**@file
 * EMLauncher設定.
 * emlauncher_config.phpにリネームする.
 */

$emlauncher_config = array(
	/** EC2環境用の設定 (httpd.confでSetEnv MFW_ENV 'ec2') */
	'ec2' => array(
		/**
		 * アップデート通知やパスワードリセットのメールの送信元アドレス.
		 */
		'mail_sender' => 'EMLauncher <no-reply@example.com>',

		/**
		 * メールをBCCで送信するときのTOに入れるアドレス.
		 * 設定しない場合は`mail_sender`が使われる.
		 */
		'mail_bcc_to' => 'EMLauncher <bcc@example.com>',

		/**
		 * タイトル等につけるprefix
		 */
		'title_prefix' => '',

		/**
		 * HTTPSで動作させる.
		 * ログイン時にHTTPSで無かった場合、HTTPSでリダイレクトする.
		 */
		'enable_https' => false,

		/** ログインの設定. */
		'login' => array(
			/**
			 * email+passwordによるログインを許可.
			 * `user_pass`テーブルに登録されているアカウントでログイン可能にする.
			 * @note
			 *  ユーザを追加する時は`user_pass`テーブルに`email`のみを登録し
			 *  パスワードリセットの手順を踏むことでパスワードを登録する.
			 */
			'enable_password' => true,

			/**
			 * Googleアカウントでのログインを許可.
			 * アカウントのメールアドレスが'allowed_mailaddr_pattern'にマッチするか,
			 * user_passテーブルに存在したらログインを認める.
			 *
			 * 利用する場合, 事前にgoogoleにアプリを登録してOAuthのID, Secretを発行しておく.
			 */
			'enable_google_auth' => true,
			'google_app_id' => 'xxxxxxxx.apps.googleusercontent.com',
			'google_app_secret' => 'xxxxxxxx',
			'allowed_mailaddr_pattern' => '/@klab\.com$/',

			/**
			 * AzureADアカウントでのログインを許可.
			 * アカウントのメールアドレスが'allowed_mailaddr_pattern'にマッチするか,
			 * user_passテーブルに存在したらログインを認める.
			 *
			 * 利用する場合, 事前にAzureADにアプリを登録してOAuthのID, Secretを発行しておく.
			 */
			'enable_azuread_auth' => true,
			'azuread_app_id' => 'xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx',
			'azuread_app_secret' => 'xxxxxxxx',
			),

		/** AppStore, GooglePlayでの制限ファイルサイズ(MB) */
		'package' => array(
			'file_size_warning_ios' => 150,
			'file_size_warning_android' => 100,
			),

		/**
		 * Storage指定
		 * - S3
		 * - LocalFile
		 */
		'storage_class' => 'S3',

		/** AWSの設定 (storage_class='S3'の場合）*/
		'aws' => array(
			/**
			 * APIアクセスのためのKeyとSecret.
			 * null にすることでIAMロールを利用できる.
			 */
			'key' => 'xxxxxxxx',
			'secret' => 'xxxxxxxx',

			/** S3のRegion. */
			'region' => 'ap-northeast-1',

			/** S3のbucket名. 予め作成しておく. */
			'bucket_name' => 'emlauncher',

			/** S3互換ストレージを利用する場合のURL（LocalSackなど）
			 *  AWSのS3を利用するときは指定しない
			 *  base_url: EMLauncherからアクセスするときのAPIエンドポイント
			 *  external_url: ブラウザからアクセスするときのURL (base_urlと同じ場合は省略可)
			 */
			// 'base_url => 'http://localstack:4572',
			// 'external_url => 'http://localhost:4572',
			),

		/** LocalFileの設定 (storage_class='LocalFile'の場合) */
		'local_file' => array(
			/** 保存先ディレクトリ. 予め作成してApacheに書き込み権限を与えておく. */
			'path' => '/path/to/directory',

			/** ブラウザからアクセスするときのURLに使われるprefix. */
			'url_prefix' => '/path/for/url',
			),

		/** APKファイルの設定 */
		'apkfile' => array(
			/** Javaコマンド 指定しない場合'java' */
			'java' => 'java -Xmx2048m',

			/** BundleToolのパス */
			'bundletool' => '/path/to/bundletool.jar',

			/** 再署名用のKeyStoreのパス */
			'keystore' => '/path/to/keystore.jks',

			/** キーストアのパスワード. 書式はbundletoolのks-passオプションと同じ. */
			'kspass' => 'pass:xxxxxxxx',

			/** 使用するキーペア */
			'keyalias' => 'emlauncherkey',

			/** キーのパスワード 書式はbundletoolのkey-passオプションと同じ. */
			'keypass' => 'pass:xxxxxxxx',
			/** BundleTool内包のaapt2以外を使うなら指定 */
			'aapt2' => '/path/to/aapt2',
			),
		),

	/**
	 * Docker開発環境用 (MFS_ENV=docker)
	 */
	'docker' => array(
		'mail_sender' => 'EMLauncher <no-reply@example.com>',
		'title_prefix' => '[Docker] ',
		'enable_https' => false,
		'login' => array(
			'enable_password' => true,
			'enable_google_auth' => false,
			'enable_azuread_auth' => false,
			),
		'package' => array(
			'file_size_warning_ios' => 150,
			'file_size_warning_android' => 100,
			),
		'storage_class' => 'LocalFile', // LocalStackを利用する場合は'S3'を指定する
		'local_file' => array(
			'path' => '/var/www/emlauncher',
			'url_prefix' => '/files',
			),
		'aws' => array(
			'bucket_name' => 'emlauncher-dev',
			'key' => 'mykey',
			'secret' => 'mysecret',
			'region' => 'ap-northeast-1',
			// LocalStackの設定は docker-compose.s3-localstack.yml を参照
			'base_url' => 'http://localstack:4572', // webコンテナから見えるs3コンテナのエンドポイント
			'external_url' => 'http://localhost:4572', // ブラウザからアクセスするときのURL
			),
		'apkfile' => array(
			'bundletool' => '/bundletool.jar',
			'keystore' => '/emlauncher.keystore',
			'keyalias' => 'emlauncher',
			'kspass' => 'pass:emlauncher',
			'keypass' => 'pass:emlauncher',
			'aapt2' => '/aapt2',
			),
		),
	);

/**
 * ローカル環境用の設定. (MFW_ENV=local)
 * Googleアカウント認証を無効にし、bucket名も変更している.
 */
$emlauncher_config['local'] = $emlauncher_config['ec2'];
$emlauncher_config['local']['login']['enable_google_auth'] = false;
$emlauncher_config['local']['login']['enable_azuread_auth'] = false;
$emlauncher_config['local']['aws']['bucket_name'] = 'emlauncher-dev';
