<?php

/**
 * Row object for 'app_comment' table.
 */
class AppComment extends mfwObject {
	const DB_CLASS = 'AppCommentDb';
	const SET_CLASS = 'AppCommentSet';
}

/**
 * Set of AppComment objects.
 */
class AppCommentSet extends mfwObjectSet {
	public static function hypostatize(Array $row=array())
	{
		return new AppComment($row);
	}
	protected function unsetCache($id)
	{
		parent::unsetCache($id);
	}
}

/**
 * database accessor for 'app_comment' table.
 */
class AppCommentDb extends mfwObjectDb {
	const TABLE_NAME = 'app_comment';
	const SET_CLASS = 'AppCommentSet';

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

}

