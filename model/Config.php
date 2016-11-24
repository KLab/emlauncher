<?php

class Config
{
	const CONFIG_FILE = '/config/emlauncher_config.php';
	protected static $config = null;

	public static function get($key)
	{
		if(static::$config===null){
			include APP_ROOT . static::CONFIG_FILE;
			static::$config = $emlauncher_config[mfwServerEnv::getEnv()];
		}
		return isset(static::$config[$key])? static::$config[$key]: null;
	}

	public static function has($key)
	{
		if(static::$config===null){
			include APP_ROOT . static::CONFIG_FILE;
			static::$config = $emlauncher_config[mfwServerEnv::getEnv()];
		}
		return isset(static::$config[$key]);
	}

}
