<?php
/**@file
 * EMLauncher設定.
 * emlauncher_config.phpにリネームする.
 */
require_once APP_ROOT.'/libs/aws/aws-autoloader.php';

$emlauncher_config = array(
	/** ローカル環境用の設定 */
	'local' => array(

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
			 *  ユーザを追加する時は`user_pass`テーブルに`email`のみを登録し
			 *  パスワードリセットの手順を踏むことでパスワードを登録する.
			 */
			'enable_password' => true,

			/**
			 * Googleアカウントでのログインを許可.
			 * アカウントのメールアドレスが'allowed_mailaddr_pattern'にマッチするか,
			 * user_passテーブルに存在したらログインを認める.
			 *
			 * 利用する場合, 事前にgoogole appsを作成してOAuthのID, Secretを発行しておく.
			 */
			'enable_google_auth' => true,
			'google_app_id' => 'xxxxxxxx.apps.googleusercontent.com',
			'google_app_secret' => 'xxxxxxxx',
			'allowed_mailaddr_pattern' => '/@klab\.com$/',
			),

		/** AWSの設定 */
		'aws' => array(
			/**
			 * APIアクセスのためのKeyとSecret.
			 */
			'key' => 'xxxxxxxx',
			'secret' => 'xxxxxxxx',

			/** S3のRegion. */
			'region' => Aws\Common\Enum\Region::TOKYO,

			/** S3のbucket名. 予め作成しておく. */
			'bucket_name' => 'emlauncher-dev',
			),
		),
	);

/**
 * EC2環境 (ApacheでSetEnv MFW_ENV 'ec2') の場合の設定.
 * 基本的にlocalと一緒だが、google認証を無効にして、bucket名も変えている.
 */
$emlauncher_config['ec2'] = $emlauncher_config['local'];
$emlauncher_config['ec2']['login']['enable_google_auth'] = false;
$emlauncher_config['ec2']['aws']['aws_bucket_name'] = 'emlauncher';

