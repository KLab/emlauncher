<?php

/**
 * Row object for 'comment' table.
 */
class Comment extends mfwObject {
	const DB_CLASS = 'CommentDb';
	const SET_CLASS = 'CommentSet';

	public function getMessage(){
		return $this->value('message');
	}
	public function getPackageId(){
		return $this->value('package_id');
	}
	public function getNumber(){
		return $this->value('number');
	}

	public function getCreated($format=null){
		$created = $this->value('created');
		if($created && $format){
			$created = date($format,strtotime($created));
		}
		return $created;
	}
}

/**
 * Set of Comment objects.
 */
class CommentSet extends mfwObjectSet {
	public static function hypostatize(Array $row=array())
	{
		return new Comment($row);
	}
	protected function unsetCache($id)
	{
		parent::unsetCache($id);
	}
}

/**
 * database accessor for 'comment' table.
 */
class CommentDb extends mfwObjectDb {
	const TABLE_NAME = 'comment';
	const SET_CLASS = 'CommentSet';

	public static function selectCountByAppId($app_id)
	{
		$table = static::TABLE_NAME;
		$sql = "SELECT count(*) FROM $table WHERE app_id = ?";
		return (int)mfwDBIBase::getOne($sql,array($app_id));
	}


	public static function selectCountByAppIds(array $app_ids)
	{
		if(empty($app_ids)){
			return array();
		}

		$bind = array();
		$pf = static::makeInPlaceholder($app_ids,$bind);
		$table = static::TABLE_NAME;
		$sql = "SELECT app_id,count(*) FROM $table WHERE app_id IN ($pf)";
		$rows = mfwDBIBase::getAll($sql,$bind);

		$counts = array();
		foreach($rows as $r){
			$counts[$r['app_id']] = $r['count(*)'];
		}
		return $counts;
	}

	public static function selectByAppId($app_id,$limit=null,$offset=0)
	{
		$query = 'WHERE app_id = ? ORDER BY id DESC';
		if($limit!==null && ((int)$limit)>0){
			$query .= ' LIMIT '.(int)$limit;
		}
		if(((int)$offset)>0){
			$query .= ' OFFSET '.(int)$offset;
		}
		return static::selectSet($query,array($app_id));
	}
}

