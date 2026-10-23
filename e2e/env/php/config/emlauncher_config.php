<?php
/**
 * E2E テスト用の EMLauncher 設定.
 * e2e_local: LocalFile ストレージ, e2e_s3: S3 (LocalStack).
 */
$e2e_base = array(
	'mail_sender' => 'EMLauncher <no-reply@example.com>',
	'mail_bcc_to' => 'EMLauncher <bcc@example.com>',
	'title_prefix' => '[E2E] ',
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
	'storage_class' => 'LocalFile',
	'local_file' => array(
		'path' => '/var/www/emlauncher',
		'url_prefix' => '/files',
		),
	'aws' => array(
		'bucket_name' => 'emlauncher-e2e',
		'key' => 'test',
		'secret' => 'test',
		'region' => 'ap-northeast-1',
		'base_url' => 'http://s3:4566',
		'external_url' => getenv('E2E_S3_EXTERNAL_URL') ?: 'http://localhost:14566',
		),
	'apkfile' => array(
		'java' => 'java -Xmx1g',
		'bundletool' => '/bundletool.jar',
		'keystore' => '/emlauncher.keystore',
		'keyalias' => 'emlauncher',
		'kspass' => 'pass:emlauncher',
		'keypass' => 'pass:emlauncher',
		),
	);

$emlauncher_config = array(
	'e2e_local' => $e2e_base,
	'e2e_s3' => array_merge($e2e_base, array('storage_class' => 'S3')),
	);
