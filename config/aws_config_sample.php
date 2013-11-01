<?php

$aws_config = array(
	'local' => array(
		'key' => 'xxxxxxxxxxxxxxxx',
		'secret' => 'xxxxxxxxxxxxxxxx',
		'region' => Aws\Common\Enum\Region::TOKYO,

		'bucket_name' => 'emlauncher-dev',
		),
	);

$aws_config['aws'] = $aws_config['local'];
$aws_config['aws']['bucket_name'] = 'emlauncher';

