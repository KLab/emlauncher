<?php
$serverenv_config = array(
  'application_identifier' => 'emlauncher',
  'docker' => array(
    'database' => array(
      'authfile' => '/dbauth',
      'default_master' => 'mysql:dbname=emlauncher;host=db',
    ),
    'memcache' => array(
      'host' => 'memcached',
      'port' => 11211,
    ),
    'http_proxy' => array(
    ),
  ),
);
$serverenv_config['local'] = $serverenv_config['docker'];
