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
	public static function selectByAppIdForUpdate($app_id,PDO $con=null)
	{
		$query = 'WHERE app_id = ? FOR UPDATE';
		return static::selectSet($query,array($app_id),$con);
	}
	public static function selectByPackage($pakcage_id)
	{
		$sql = 'SELECT t.* FROM package_tag j LEFT JOIN tag t ON j.tag_id = t.id WHERE package_id = ?';
		$rows = mfwDBI::getAll($sql,array($pakcage_id));
		return new TagSet($rows);
	}

	public static function insertNewTag($app_id,$name,PDO $con=null)
	{
		$row = array(
			'app_id' => $app_id,
			'name' => $name,
			);
		$tag = new Tag($row);
		$tag->insert($con);
		return $tag;
	}

	public static function updatePackageTags($package_id,$tags,$con=null)
	{
		$sql = 'DELETE FROM package_tag WHERE package_id = :package_id';
		$bind = array(
			':package_id' => $package_id,
			);
		mfwDBIBase::query($sql,$bind,$con);

		if($tags->count()){
			$bulk = array();
			foreach($tags as $tag){
				$bulk[] = "(:package_id,{$tag->getId()})";
			}
			$sql = 'INSERT INTO package_tag (package_id,tag_id) VALUES '.implode(',',$bulk);
			mfwDBIBase::query($sql,$bind,$con);
		}
		return $tags;
	}

}

