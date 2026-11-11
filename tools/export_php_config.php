<?php
/**
 * PHP 版の設定から Go 版の設定ファイル (YAML) を作る. 移行時に PHP 版のサーバで実行する.
 *
 *   php export_php_config.php <PHP 版を置いたディレクトリ> <MFW_ENV> > config.yml
 *
 * - timezone は PHP の実行時の TZ (date_default_timezone_get()) を書き出す.
 *   CLI と Apache で php.ini が違う場合は, Apache 側の date.timezone と一致しているか確認すること.
 * - cache_prefix は PHP 版と同じ "{application_identifier}_{MFW_ENV}_{ディレクトリ名}_".
 * - session.secret は毎回ランダムに作る. 複数台構成では全台で同じ値にすること.
 * 出力は JSON 形式 (YAML として読める).
 */

if ($argc < 3) {
	fwrite(STDERR, "usage: php {$argv[0]} <emlauncher dir> <MFW_ENV>\n");
	exit(2);
}
$root = realpath($argv[1]);
$env = $argv[2];
if ($root === false || !is_file("$root/config/emlauncher_config.php")) {
	fwrite(STDERR, "config/emlauncher_config.php not found in {$argv[1]}\n");
	exit(1);
}
define('APP_ROOT', $root);

include "$root/config/emlauncher_config.php";
include "$root/config/mfw_serverenv_config.php";
if (!isset($emlauncher_config[$env]) || !isset($serverenv_config[$env])) {
	fwrite(STDERR, "environment '$env' is not defined\n");
	exit(1);
}
$c = $emlauncher_config[$env];
$s = $serverenv_config[$env];

function v($a, $k, $default = null) {
	return isset($a[$k]) ? $a[$k] : $default;
}

// PDO の DSN (mysql:dbname=...;host=...;port=...)
$dsn = v($s['database'], 'default_master', '');
$dsnParams = array();
foreach (explode(';', preg_replace('/^mysql:/', '', $dsn)) as $kv) {
	if (strpos($kv, '=') !== false) {
		list($k, $val) = explode('=', $kv, 2);
		$dsnParams[trim($k)] = trim($val);
	}
}
if (isset($dsnParams['unix_socket'])) {
	fwrite(STDERR, "warning: unix_socket is not supported. set database.host / port manually.\n");
}

$id = str_replace(' ', '_', v($serverenv_config, 'application_identifier', ''));
$login = v($c, 'login', array());
$aws = v($c, 'aws', array());
$local = v($c, 'local_file', array());
$apk = v($c, 'apkfile', array());
$pkg = v($c, 'package', array());

$out = array(
	'listen' => '127.0.0.1:8080',
	'timezone' => date_default_timezone_get(),
	'cache_prefix' => "{$id}_{$env}_" . basename($root) . '_',
	'title_prefix' => (string)v($c, 'title_prefix', ''),
	'enable_https' => (bool)v($c, 'enable_https', false),
	'mail_sender' => (string)v($c, 'mail_sender', ''),
	'mail_bcc_to' => (string)v($c, 'mail_bcc_to', ''),
	'upload_lock_wait_timeout' => (int)v($c, 'upload_lock_wait_timeout', 0),
	'mail' => array(
		'sendmail' => ini_get('sendmail_path') ?: '/usr/sbin/sendmail -t -i',
	),
	'login' => array(
		'enable_password' => (bool)v($login, 'enable_password', false),
		'enable_google_auth' => (bool)v($login, 'enable_google_auth', false),
		'google_app_id' => (string)v($login, 'google_app_id', ''),
		'google_app_secret' => (string)v($login, 'google_app_secret', ''),
		'allowed_mailaddr_pattern' => (string)v($login, 'allowed_mailaddr_pattern', ''),
		'enable_azuread_auth' => (bool)v($login, 'enable_azuread_auth', false),
		'azuread_app_id' => (string)v($login, 'azuread_app_id', ''),
		'azuread_app_secret' => (string)v($login, 'azuread_app_secret', ''),
	),
	'package' => array(
		'file_size_warning_ios' => (int)v($pkg, 'file_size_warning_ios', 0),
		'file_size_warning_android' => (int)v($pkg, 'file_size_warning_android', 0),
	),
	'session' => array(
		'secret' => bin2hex(random_bytes(32)),
	),
	'database' => array(
		'host' => v($dsnParams, 'host', 'localhost'),
		'port' => (int)v($dsnParams, 'port', 3306),
		'name' => v($dsnParams, 'dbname', 'emlauncher'),
		'authfile' => (string)v($s['database'], 'authfile', ''),
	),
	'memcache' => array(
		'host' => (string)v(v($s, 'memcache', array()), 'host', 'localhost'),
		'port' => (int)v(v($s, 'memcache', array()), 'port', 11211),
	),
	'storage_class' => (string)v($c, 'storage_class', 'S3'),
	'aws' => array(
		'key' => (string)v($aws, 'key', ''),
		'secret' => (string)v($aws, 'secret', ''),
		'region' => (string)v($aws, 'region', ''),
		'bucket_name' => (string)v($aws, 'bucket_name', ''),
		'base_url' => (string)v($aws, 'base_url', ''),
		'external_url' => (string)v($aws, 'external_url', ''),
	),
	'local_file' => array(
		'path' => (string)v($local, 'path', ''),
		'url_prefix' => (string)v($local, 'url_prefix', ''),
		'serve' => true,
	),
	'apkfile' => array(
		'java' => (string)v($apk, 'java', 'java'),
		'bundletool' => (string)v($apk, 'bundletool', ''),
		'keystore' => (string)v($apk, 'keystore', ''),
		'kspass' => (string)v($apk, 'kspass', ''),
		'keyalias' => (string)v($apk, 'keyalias', ''),
		'keypass' => (string)v($apk, 'keypass', ''),
		'aapt2' => (string)v($apk, 'aapt2', ''),
	),
);

echo json_encode($out, JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE), "\n";
