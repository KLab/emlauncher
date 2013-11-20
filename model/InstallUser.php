<?php

/*! @file
 * あるアプリをインストールしているユーザの情報.
 * テーブルはapp_install_user.
 * @sa InstallLog
 */

class InstallUser extends mfwObject {
	const SET_CLASS = 'InstallUserSet';

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

class InstallUserSet extends mfwObjectSet {
	const PRIMARY_KEY = 'mail';
	public static function hypostatize(Array $row=array())
	{
		return new InstallUser($row);
	}
	protected function unsetCache($id)
	{
		parent::unsetCache($id);
	}
}

