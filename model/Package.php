<?php
require_once APP_ROOT.'/model/Tag.php';
require_once APP_ROOT.'/model/Random.php';
require_once APP_ROOT.'/model/S3.php';

/**
 * Row object for 'package' table.
 */
class Package extends mfwObject {
	const DB_CLASS = 'PackageDb';
	const SET_CLASS = 'PackageSet';

	const PF_ANDROID = 'Android';
	const PF_IOS = 'iOS';
	const PF_UNKNOWN = 'unknown';
	const MIME_ANDROID = 'application/vnd.android.package-archive';
	const MIME_IOS = 'application/octet-stream';

	const FILE_DIR = 'package/';
	const TEMP_DIR = 'temp-data/';

	protected $tags = null;

	public function getId(){
		return $this->value('id');
	}
	public function getAppId(){
		return $this->value('app_id');
	}
	public function getplatform(){
		return $this->value('platform');
	}
	public function getTitle(){
		return $this->value('title');
	}
	public function getDescription(){
		return $this->value('description');
	}
	public function getCreated($format=null){
		$created = $this->value('created');
		if($created && $format){
			$created = date($format,strtotime($created));
		}
		return $created;
	}

	public function getTags()
	{
		if($this->tags===null){
			$this->tags = TagDb::selectByPackageId($this->getId());
		}
		return $this->tags;
	}
	public function applyTags($tags,$con=null)
	{
		$this->tags = TagDb::updatePackageTags($this->getId(),$tags,$con);
	}

	protected function getFileKey()
	{
		$key = "{$this->getAppId()}/{$this->getId()}_{$this->value('file_name')}";
		return static::FILE_DIR.$key;
	}
	public function renameTempFile()
	{
		$tempkey = static::TEMP_DIR.$this->value('file_name');
		$newkey = $this->getFileKey();
		S3::rename($tempkey,$newkey,'private');
	}
	public function getFileUrl($expire=null)
	{
		return S3::url($this->getFileKey(),$expire);
	}

}

/**
 * Set of Package objects.
 */
class PackageSet extends mfwObjectSet {
	public static function hypostatize(Array $row=array())
	{
		return new Package($row);
	}
	protected function unsetCache($id)
	{
		parent::unsetCache($id);
	}
}

/**
 * database accessor for 'package' table.
 */
class PackageDb extends mfwObjectDb {
	const TABLE_NAME = 'package';
	const SET_CLASS = 'PackageSet';

	public static function checkPlatform($name,$file)
	{
		$ext = pathinfo($name,PATHINFO_EXTENSION);
		$is_zip = substr($file,0,4)==="PK\x03\x04";
		if($is_zip && $ext==='apk'){
			return Package::PF_ANDROID;
		}
		if($is_zip && $ext==='ipa'){
			return Package::PF_IOS;
		}
		return Package::PF_UNKNOWN;
	}

	public static function uploadTemporary($name,$file,$mime)
	{
		$platform = static::checkPlatform($name,$file);

		switch($platform){
		case Package::PF_ANDROID:
			$mime = Package::MIME_ANDROID;
			$ext = 'apk';
			break;
		case Package::PF_IOS:
			$mime = Package::MIME_IOS;
			$ext = 'ipa';
			break;
		default:
			$ext = pathinfo($name,PATHINFO_EXTENSION);
		}

		$temp_name = Random::string(16).".$ext";
		S3::upload(Package::TEMP_DIR.$temp_name,$file,$mime,'private');

		return array($temp_name,$platform);
	}

	public static function insertNewPackage($app_id,$platform,$file_name,$title,$description)
	{
		$row = array(
			'app_id' => $app_id,
			'platform' => $platform,
			'file_name' => $file_name,
			'title' => $title,
			'description' => $description,
			'created' => date('Y-m-d H:i:s'),
			);
		$pkg = new Package($row);
		$pkg->insert();
		return $pkg;
	}

	public static function selectByAppId($app_id,$pf_filter=null)
	{
		$query = 'WHERE app_id = :app_id';
		$bind = array(':app_id' => $app_id);
		if($pf_filter){
			$query .= ' AND platform = :platform';
			$bind[':platform'] = $pf_filter;
		}
		$query .= ' ORDER BY id DESC';
		return static::selectSet($query,$bind);
	}
}

