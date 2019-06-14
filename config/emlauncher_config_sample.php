<?php
/**@file
 * EMLauncher設定.
 * emlauncher_config.phpにリネームする.
 */
require_once APP_ROOT.'/libs/aws/aws-autoloader.php';

$emlauncher_config = array(
	/** EC2環境用の設定 (httpd.confでSetEnv MFW_ENV 'ec2') */
	'ec2' => array(
		/**
		 * アップデート通知やパスワードリセットのメールの送信元アドレス.
		 */
		'mail_sender' => 'EMLauncher <no-reply@example.com>',

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
			 *	ユーザを追加する時は`user_pass`テーブルに`email`のみを登録し
			 *	パスワードリセットの手順を踏むことでパスワードを登録する.
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
			 */
			'key' => 'xxxxxxxx',
			'secret' => 'xxxxxxxx',

			/** S3のRegion. */
			'region' => Aws\Common\Enum\Region::TOKYO,

			/** S3のbucket名. 予め作成しておく. */
			'bucket_name' => 'emlauncher',

			/** S3互換ストレージを利用する場合のURL（LocalSackなど）
			 *	AWSのS3を利用するときは指定しない
			 *	base_url: EMLauncherからアクセスするときのAPIエンドポイント
			 *			  Dockerでlocalstackを利用する場合、
			 *			  webコンテナから見えるs3コンテナのエンドポイントを指定する
			 *	external_url: ブラウザからアクセスするときのURL
			 *				  base_urlと同じ場合は指定しない
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
			'base_url' => 'http://localstack:4572',
			'external_url' => 'http://localhost:4572',
			),
		),
	);

/**
 * ローカル環境用の設定. (MFW_ENV=local)
 * Googleアカウント認証を無効にし、bucket名も変更している.
 */
$emlauncher_config['local'] = $emlauncher_config['ec2'];
$emlauncher_config['local']['login']['enable_google_auth'] = false;
$emlauncher_config['local']['aws']['bucket_name'] = 'emlauncher-dev';
