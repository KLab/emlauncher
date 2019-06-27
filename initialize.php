<?php

define('APP_ROOT',realpath(dirname(__FILE__)));

require_once APP_ROOT.'/mfw/mfwServerEnv.php';
require_once APP_ROOT.'/mfw/mfwRequest.php';
require_once APP_ROOT.'/mfw/mfwModules.php';
require_once APP_ROOT.'/mfw/mfwActions.php';
require_once APP_ROOT.'/mfw/mfwTemplate.php';
require_once APP_ROOT.'/mfw/mfwSession.php';
require_once APP_ROOT.'/mfw/mfwMemcache.php';
require_once APP_ROOT.'/mfw/mfwApc.php';
require_once APP_ROOT.'/mfw/mfwDBConnection.php';
require_once APP_ROOT.'/mfw/mfwDBIBase.php';
require_once APP_ROOT.'/mfw/mfwObject.php';
require_once APP_ROOT.'/mfw/mfwObjectSet.php';
require_once APP_ROOT.'/mfw/mfwObjectDb.php';
require_once APP_ROOT.'/mfw/mfwHttp.php';
//require_once APP_ROOT.'/mfw/mfwOAuth.php';

require_once APP_ROOT."/vendor/autoload.php";

function apache_log($key,$value)
{
	static $log = array();
	if(function_exists('apache_setenv')){
		$log['env'] = mfwServerEnv::getEnv();
		$log[$key] = $value;
		apache_setenv('LOGMSG',json_encode($log));
	}
}
