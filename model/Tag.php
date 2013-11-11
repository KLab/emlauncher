<?php

/**
 * Row object for 'tag' table.
 */
class Tag extends mfwObject {
	const DB_CLASS = 'TagDb';
	const SET_CLASS = 'TagSet';

	public function getId(){
		return $this->value('id');
	}
	public function getAppId(){
		return $this->value('app_id');
	}
	public function getName(){
		return $this->value('name');
	}
}

/**
 * Set of Tag objects.
 */
class TagSet extends mfwObjectSet {
	public static function hypostatize(Array $row=array())
	{
		return new Tag($row);
	}
	protected function unsetCache($id)
	{
		parent::unsetCache($id);
	}
}

/**
 * database accessor for 'tag' table.
 */
class TagDb extends mfwObjectDb {
	const TABLE_NAME = 'tag';
	const SET_CLASS = 'TagSet';

	public static function selectByAppId($app_id)
	{
		$query = 'WHERE app_id = ?';
		return static::selectSet($query,array($app_id));
	}
	public static function selectByPackage($pakcage_id)
	{
		$sql = 'SELECT t.* FROM package_tag j LEFT JOIN tag t ON j.tag_id = t.id WHERE package_id = ?';
		$rows = mfwDBI::getAll($sql,array($pakcage_id));
		return new TagSet($rows);
	}

}

