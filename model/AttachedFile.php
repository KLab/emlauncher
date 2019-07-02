<?php

/**
 * Row object for 'attached_file' table.
 */
class AttachedFile extends mfwObject {
	const DB_CLASS = 'AttachedFileDb';
	const SET_CLASS = 'AttachedFileSet';

	const TYPE_UNKNOWN = '';
	const TYPE_AAB = 'aab';

	public function getId(){
		return $this->value('id');
	}
	public function getAppId(){
		return $this->value('app_id');
	}
	public function getPackageId(){
		return $this->value('package_id');
	}
	public function getBaseFileName(){
		return $this->value('file_name');
	}
	public function getOriginalFileName(){
		return $this->value('original_file_name');
	}
	public function getFileSize(){
		return $this->value('file_size');
	}
	public function getFileType(){
		return $this->value('file_type');
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
 * Set of AttachedFile objects.
 */
class AttachedFileSet extends mfwObjectSet {
	public static function hypostatize(Array $row=array())
	{
		return new AttachedFile($row);
	}
	protected function unsetCache($id)
	{
		parent::unsetCache($id);
	}
}

/**
 * database accessor for 'attached_file' table.
 */
class AttachedFileDb extends mfwObjectDb {
	const TABLE_NAME = 'attached_file';
	const SET_CLASS = 'AttachedFileSet';

	public static function insertNewAttachedFile($package,$file_name,$file_size,$file_type,$con)
	{
		$ext = strrchr($file_name, '.');
		$row  = array(
			'app_id' => $package->getAppId(),
			'package_id' => $package->getId(),
			'file_name' => Random::string(16).".$ext",
			'original_file_name' => $file_name,
			'file_size' => $file_size,
			'file_type' => $file_type,
			'created' => date('Y-m-d H:i:s'),
			);
		$attached = new AttachedFile($row);
		$attached->insert($con);
		return $attached;
	}

	public static function selectByPackageId($pakckage_id)
	{
		$query = 'WHERE package_id = :package_id ORDER BY id ASC';
		$bind = array(':package_id' => $package_id);
		return static::selectSet($query,$bind);
	}
}
