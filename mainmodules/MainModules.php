<?php
require_once APP_ROOT.'/mainmodules/MainActions.php';

class MainModules extends mfwModules {

	protected static function rootdir()
	{
		return __DIR__;
	}

	public static function execute()
	{
		// PATH_INFO should be '/module/action'. The deefault is 'top/index'.
		$pathinfo = mfwRequest::getPathInfoArray();
		$module = (isset($pathinfo[0]) && !empty($pathinfo[0]))? $pathinfo[0] : 'top';
		$action = (isset($pathinfo[1]) && !empty($pathinfo[1]))? $pathinfo[1] : 'index';
		return parent::executeAction($module,$action);
	}

}
