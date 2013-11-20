<?php

/*! @file
 * あるユーザがインストールしているアプリの情報.
 * テーブルはapp_install_user.
 * @sa InstallLog
 */

class InstallApp extends mfwObject {
	const SET_CLASS = 'InstallAppSet';

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
}

class InstallAppSet extends mfwObjectSet {
	const PRIMARY_KEY = 'app_id';
	protected $user;

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
}

