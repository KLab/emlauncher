<?php

/**
 * Row object for 'application_owner' table.
 */
class ApplicationOwner extends mfwObject {
	const DB_CLASS = 'ApplicationOwnerDb';
	const SET_CLASS = 'ApplicationOwnerSet';

	public function getOwnerMail(){
		return $this->value('owner_mail');
	}
	public function getAppId(){
		return $this->value('app_id');
	}
}

/**
 * Set of ApplicationOwner objects.
 */
class ApplicationOwnerSet extends mfwObjectSet {
	public static function hypostatize(Array $row=array())
	{
		return new ApplicationOwner($row);
	}
	protected function unsetCache($id)
	{
		parent::unsetCache($id);
	}

	public function getMailArray()
	{
		return $this->getColumnArray('owner_mail');
	}
}

/**
 * database accessor for 'application_owner' table.
 */
class ApplicationOwnerDb extends mfwObjectDb {
	const TABLE_NAME = 'application_owner';
	const SET_CLASS = 'ApplicationOwnerSet';

	public static function selectByAppId($app_id)
	{
		$query = "WHERE app_id = ?";
		return static::selectSet($query,array($app_id));
	}

	public static function selectByOwnerMail($mail)
	{
		$query = "WHERE owner_mail = ?";
		return static::selectSet($query,array($mail));
	}

	public static function deleteOwner($app_id,array $owners,$con=null)
	{
		$bind = array(':app_id' => $app_id);
		$ph = self::makeInPlaceHolder($owners,$bind,'owner_mail');

		$sql = "DELETE FROM application_owner WHERE app_id = :app_id AND owner_mail IN ($ph)";
		return mfwDBIBase::query($sql,$bind,$con);
	}

	public static function addOwner($app_id,$owners,$con=null)
	{
		$bind = array(':app_id' => $app_id);
		$values = array();
		foreach($owners as $k=>$v){
			$key = ":mail_$k";
			$values[] = "(:app_id,$key)";
			$bind[$key] = $v;
		}
		$sql = 'INSERT INTO application_owner (app_id,owner_mail) VALUES '
			. implode(',',$values);
		return mfwDBIBase::query($sql,$bind,$con);
	}

}

