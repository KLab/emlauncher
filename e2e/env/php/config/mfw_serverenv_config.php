<?php
/**
 * E2E テスト用の ServerEnv 設定.
 */
$e2e_base = array(
	'database' => array(
		'authfile' => '/dbauth',
		'default_master' => 'mysql:dbname=emlauncher;host=db;charset=utf8mb4',
		),
	'memcache' => array(
		'host' => 'memcached',
		'port' => 11211,
		),
	'http_proxy' => array(
		),
	);

$serverenv_config = array(
	'application_identifier' => 'emlauncher',
	'e2e_local' => $e2e_base,
	'e2e_s3' => $e2e_base,
	);
