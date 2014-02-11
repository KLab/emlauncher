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

	// AppStore, GooglePlayでの制限ファイルサイズ(MB)
	const IOS_FILE_SIZE_LIMIT_MB = 100;
	const ANDROID_FILE_SIZE_LIMIT_MB = 50;

	protected $app = null;
	protected $tags = null;
	protected $install_users = null;

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
	public function getBaseFileName(){
		return $this->value('file_name');
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
	public function getOriginalFileName(){
		return $this->value('original_file_name');
	}
	public function getFileSize(){
		return $this->value('file_size');
	}
	public function getCreated($format=null){
		$created = $this->value('created');
		if($created && $format){
			$created = date($format,strtotime($created));
		}
		return $created;
	}

	public function getFileSizeLimitMB()
	{
		switch($this->getPlatform()){
		case self::PF_IOS:
			return self::IOS_FILE_SIZE_LIMIT_MB;
		case self::PF_ANDROID:
			return self::ANDROID_FILE_SIZE_LIMIT_MB;
		default:
			return 0;
		}
	}

	public function isFileSizeWarned()
	{
		$limit = $this->getFileSizeLimitMB() * 1024 * 1024;
		if($limit==0){
			return false;
		}
		return ($this->getFileSize() > $limit);
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
	public function deleteFile()
	{
		$key = $this->getFileKey();
		S3::delete($key);
	}
	public function getFileUrl($expire=null)
	{
		return S3::url($this->getFileKey(),$expire);
	}

	public function getInstallUrl()
	{
		return mfwRequest::makeUrl("/package/install?id={$this->getId()}");
	}

	public function getInstallUsers()
	{
		if($this->install_users===null){
			$this->install_users = InstallLog::getPackageInstallUsers($this);
		}
		return $this->install_users;
	}

	public function getInstallCount()
	{
		$users = $this->getInstallUsers();
		return count($users);
	}

	public function updateInfo($title,$description,TagSet $tags,$con=null)
	{
		$this->row['title'] = $title;
		$this->row['description'] = $description;
		$this->update($con);
		$this->applyTags($tags,$con);
	}

	public function delete($con=null)
	{
		TagDb::removeFromPackage($this,$con);
		return parent::delete($con);
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

	public static function insertNewPackage($app_id,$platform,$ext,$title,$description,$ios_identifier,$org_file_name,$file_size,TagSet $tags,$con)
	{
		$row = array(
			'app_id' => $app_id,
			'platform' => $platform,
			'file_name' => Random::string(16).".$ext",
			'title' => $title,
			'description' => $description,
			'ios_identifier' => $ios_identifier,
			'original_file_name' => $org_file_name,
			'file_size' => $file_size,
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

	public static function selectByAppIdPfTagsWithLimit($app_id, $pf_filter, array $tags, $offset, $count)
	{
		if (!$pf_filter && empty($tags)) {
			$query = 'WHERE app_id = :app_id';
			$query .= sprintf(' ORDER BY id DESC LIMIT %d, %d', $offset, $count);
			return static::selectSet($query, array(':app_id' => $app_id));
		}
		$sql = 'SELECT p.* FROM package AS p LEFT JOIN package_tag AS t ON p.id = t.package_id WHERE p.app_id = ?';
		$bind = array($app_id);
		if ($pf_filter) {
			$sql .= ' AND p.platform = ?';
			array_push($bind, $pf_filter);
		}
		if (!empty($tags)) {
			$sql .= ' AND t.tag_id in ('. implode(',', array_fill(0, count($tags), '?')) .')';
			$bind = array_merge($bind, $tags);
			$sql .= ' GROUP BY p.id HAVING COUNT(p.id) >= '.count($tags);
		}
		$sql .= sprintf(' ORDER BY p.id DESC LIMIT %d, %d', $offset, $count);
		return new PackageSet(mfwDBIBase::getAll($sql, $bind));
	}
}

