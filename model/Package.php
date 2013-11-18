<?php
require_once APP_ROOT.'/model/Application.php';
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

	protected $app = null;
	protected $tags = null;

	public function getId(){
		return $this->value('id');
	}
	public function getAppId(){
		return $this->value('app_id');
	}
	public function getApplication()
	{
		if($this->app===null){
			$this->app = ApplicationDb::retrieveByPK($this->getAppId());
		}
		return $this->app;
	}
	public function getPlatform(){
		return $this->value('platform');
	}
	public function getTitle(){
		return $this->value('title');
	}
	public function getDescription(){
		return $this->value('description');
	}
	public function getIOSIdentifier(){
		return $this->value('ios_identifier');
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
	public function applyTags(TagSet $tags,$con=null)
	{
		$this->tags = TagDb::updatePackageTags($this->getId(),$tags,$con);
	}

	protected function getFileKey()
	{
		$key = "{$this->getAppId()}/{$this->getId()}_{$this->value('file_name')}";
		return static::FILE_DIR.$key;
	}
	public function uploadFile($file_content,$mime)
	{
		$key = $this->getFileKey();
		S3::upload($key,$file_content,$mime,'private');
	}
	public static function uploadTempFile($file_content,$ext,$mime)
	{
		$tmp_name = Random::string(16).".$ext";
		S3::upload(static::TEMP_DIR.$tmp_name,$file_content,$mime,'private');
		return $tmp_name;
	}
	public function renameTempFile($temp_name)
	{
		$tempkey = static::TEMP_DIR.$temp_name;
		$newkey = $this->getFileKey();
		S3::rename($tempkey,$newkey,'private');
	}
	public function getFileUrl($expire=null)
	{
		return S3::url($this->getFileKey(),$expire);
	}

	public function getInstallUrl()
	{
		return mfwRequest::makeUrl("/package/install?id={$this->getId()}");
	}

	public function getInstallCount()
	{
		return InstallLog::getPackageInstallCount($this);
	}

	public function updateInfo($title,$description,TagSet $tags,$con=null)
	{
		$this->row['title'] = $title;
		$this->row['description'] = $description;
		$this->update($con);
		$this->applyTags($tags,$con);
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

	public static function getPackageInfo($name,$file,$mime)
	{
		$platform = Package::PF_UNKNOWN;
		$ext = pathinfo($name,PATHINFO_EXTENSION);
		$is_zip = substr($file,0,4)==="PK\x03\x04";
		if($is_zip && $ext==='apk'){
			$platform = Package::PF_ANDROID;
			$mime = Package::MIME_ANDROID;
		}
		if($is_zip && $ext==='ipa'){
			$platform = Package::PF_IOS;
			$mime = Package::MIME_IOS;
		}
		return array($platform,$ext,$mime);
	}

	public static function insertNewPackage($app_id,$platform,$ext,$title,$description,$ios_identifier,TagSet $tags,$con)
	{
		$row = array(
			'app_id' => $app_id,
			'platform' => $platform,
			'file_name' => Random::string(16).".$ext",
			'title' => $title,
			'description' => $description,
			'ios_identifier' => $ios_identifier,
			'created' => date('Y-m-d H:i:s'),
			);
		$pkg = new Package($row);
		$pkg->insert($con);
		$pkg->applyTags($tags,$con);
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

