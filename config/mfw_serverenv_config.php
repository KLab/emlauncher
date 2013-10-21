<?php
/**
 * ServerEnv configuration
 */
$serverenv_config = array(

	'local' => array(
		'database' => array(
			'authfile' => '/home/dbauth/httpd',
			'default_master' => 'mysql:dbname=emlauncher;host=localhost',
			),
		'http_proxy' => array(
			//'host' => '127.0.0.1',
			//'port' => 8080,
			),
		'memcache' => array(
			'host' => 'localhost',
			'port' => 11211,
			),
		),

	);
