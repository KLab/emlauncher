<?php
require_once __DIR__.'/Application.php';

/*! @file
 * あるユーザがインストールしているアプリの情報.
 * テーブルはapp_install_user.
 * @sa InstallLog
 */

class InstallApp extends mfwObject {
	const SET_CLASS = 'InstallAppSet';

	protected $app = null;

	public function getAppId(){
		return $this->value('app_id');
	}
	public function getMail(){
		return $this->value('mail');
	}
	public function getLastInstalled(){
		return $this->value('last_installed');
	}
	public function getNotifySetting(){
		return (bool)$this->value('notify');
	}
	public function getApp()
	{
		if($this->app===null){
			$this->app = ApplicationDb::retrieveByPK($this->getAppId());
		}
		return $this->app;
	}
	public function setApp(Application $app){
		$this->app = $app;
	}
}

class InstallAppSet extends mfwObjectSet {
	const PRIMARY_KEY = 'app_id';
	protected $user;
	protected $apps = null;

	protected function selectApps()
	{
		if($this->apps===null){
			$app_ids = $this->getColumnArray('app_id');
			$this->apps = ApplicationDb::retrieveByPKs($app_ids);
		}
		return $this->apps;
	}

	public function __construct(User $user,Array $rows=array())
	{
		parent::__construct($rows);
		$this->user = $user;
	}
	public static function hypostatize(Array $row=array())
	{
		return new InstallApp($row);
	}
	protected function unsetCache($id)
	{
		parent::unsetCache($id);
	}

	public function offsetGet($offset)
	{
		$ia = parent::offsetGet($offset);
		$apps = $this->selectApps();
		if(isset($apps[$ia->getAppId()])){
			$ia->setApp($apps[$ia->getAppId()]);
		}
		return $ia;
	}
}

