<?php
/**
 * ServerEnv configuration
 */
$serverenv_config = array(

	/**
	 * アプリケーション識別子.
	 * KVSのキーのprefix等に使われる.
	 */
	'application_identifier' => 'ohoflight2',

	/**
	 * EC2環境用の設定 (httpd.confでSetEnv MFW_ENV 'ec2')
	 */
	'ec2' => array(

		/**
		 * Database設定
		 */
		'database' => array(
			/** DBの ユーザ名:パスワード が書かれたファイル */
			'authfile' => '/path/to/dbauth-file',
			/** DBの接続先 */
			'default_master' => 'mysql:dbname=emlauncher;host=localhost',
			),

		/**
		 * HTTPプロキシ設定.
		 * 外部への接続にプロキシを使う場合に設定する.
		 */
		'http_proxy' => array(
			),

		/**
		 * Memcached設定.
		 */
		'memcache' => array(
			'host' => 'localhost',
			'port' => 11211,
			),
		),

	);

$serverenv_config['local'] = $serverenv_config['ec2'];

