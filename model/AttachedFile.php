<?php
require_once APP_ROOT.'/model/Package.php';
require_once APP_ROOT.'/model/Storage.php';

/**
 * Row object for 'attached_file' table.
 */
class AttachedFile extends mfwObject {
	const DB_CLASS = 'AttachedFileDb';
	const SET_CLASS = 'AttachedFileSet';

	const TYPE_UNKNOWN = '';
	const TYPE_APK = 'apk';
	const TYPE_DSYM = 'dSYM';

	static function getTypeFromExt($ext){
		switch($ext){
		case 'apk':
			return self::TYPE_APK;
		default:
			return self::TYPE_UNKNOWN;
		}
	}

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
	public function getMime(){
		switch($this->getFileType()){
		case self::TYPE_APK:
			return Package::MIME_ANDROID_APK;
		default:
			return null;
		}
	}

	protected function getFileKey(){
		$key = "{$this->getAppId()}/{$this->getPackageId()}_{$this->getId()}_{$this->getBaseFileName()}";
		return Package::FILE_DIR.$key;
	}

	public function uploadFile($file_path,$default_mime){
		$key = $this->getFileKey();
		$mime = $this->getMime() ?: $default_mime;
		Storage::saveFile($key,$file_path,$mime);
	}

	public function renameTempFile($temp_name){
		$tempkey = Package::TEMP_DIR.$temp_name;
		$newkey = $this->getFileKey();
		Storage::rename($tempkey,$newkey);
	}

	public function deleteFile(){
		$key = $this->getFileKey();
		Storage::delete($key);
	}

	public function getFileUrl($expire=null){
		return Storage::url($this->getFileKey(),$expire,$this->getOriginalFileName());
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

	public function delete($con){
		if($this->count()==0){
			return;
		}
		AttachedFileDb::deleteSet($this->getColumnArray('id'),$con);
	}
	public function deleteFiles(){
		foreach($this as $file){
			$file->deleteFile();
		}
	}

	public function pickupByType($type){
		foreach($this as $file){
			if($file->getFileType()===$type){
				return $file;
			}
		}
		return null;
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
			'file_name' => Random::string(16).$ext,
			'original_file_name' => $file_name,
			'file_size' => $file_size,
			'file_type' => $file_type,
			'created' => date('Y-m-d H:i:s'),
			);
		$attached = new AttachedFile($row);
		$attached->insert($con);
		return $attached;
	}

	public static function selectByPackageId($package_id)
	{
		$query = 'WHERE package_id = :package_id ORDER BY id ASC';
		$bind = array(':package_id' => $package_id);
		return static::selectSet($query,$bind);
	}

	public static function deleteSet($ids,$con)
	{
		$table = static::TABLE_NAME;
		$bind = array();
		$query = "DELETE FROM `$table` WHERE `id` IN (".static::makeInPlaceHolder($ids,$bind).')';
		return mfwDBIBase::query($query,$bind,$con);
	}
}
