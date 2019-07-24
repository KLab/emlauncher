<?php
require_once APP_ROOT.'/model/Application.php';
require_once APP_ROOT.'/model/Tag.php';
require_once APP_ROOT.'/model/Random.php';
require_once APP_ROOT.'/model/Storage.php';
require_once APP_ROOT.'/model/AttachedFile.php';
require_once APP_ROOT.'/model/Config.php';

/**
 * Row object for 'package' table.
 */
class Package extends mfwObject {
	const DB_CLASS = 'PackageDb';
	const SET_CLASS = 'PackageSet';

	const PF_ANDROID = 'Android';
	const PF_IOS = 'iOS';
	const PF_UNKNOWN = 'unknown';
	const MIME_ANDROID_APK = 'application/vnd.android.package-archive';
	const MIME_ANDROID_AAB = 'application/octet-stream';
	const MIME_IOS = 'application/octet-stream';

	const FILE_DIR = 'package/';
	const TEMP_DIR = 'temp-data/';

	const SHORT_DESCRIPTION_LENGTH = 100;

	protected $app = null;
	protected $tags = null;
	protected $guest_passes = null;
	protected $install_users = null;
	protected $attached_files = null;
	protected $config = null;

	protected function getConfig(){
		if($this->config===null){
			$this->config = Config::get('package');
		}
		return $this->config;
	}

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
	public function getShortDescription(){
		$desc = $this->getDescription();
		if(mb_strlen($desc,'UTF-8')>self::SHORT_DESCRIPTION_LENGTH){
			$desc = mb_substr($desc,0,self::SHORT_DESCRIPTION_LENGTH,'UTF-8') . '...';
		}
		return $desc;
	}

	public function getIdentifier(){
		return $this->value('identifier');
	}
	public function getOriginalFileName(){
		return $this->value('original_file_name');
	}
	public function getFileSize(){
		return $this->value('file_size');
	}
	public function isProtected(){
		return (bool)$this->value('protect');
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
		$conf = $this->getConfig();
		switch($this->getPlatform()){
		case self::PF_IOS:
			return (int)$conf['file_size_warning_ios'];
		case self::PF_ANDROID:
			return (int)$conf['file_size_warning_android'];
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
	public function initTags(TagSet $tags,$con=null)
	{
		TagDb::insertPackageTags($this,$tags,$con);
		$this->tags = $tags;
	}

	protected function getFileKey()
	{
		$key = "{$this->getAppId()}/{$this->getId()}_{$this->value('file_name')}";
		return static::FILE_DIR.$key;
	}
	public function uploadFile($file_path,$mime)
	{
		$key = $this->getFileKey();
		Storage::saveFile($key,$file_path,$mime);
	}
	public static function uploadTempFile($file_path,$ext,$mime)
	{
		$tmp_name = Random::string(16).".$ext";
		Storage::saveFile(static::TEMP_DIR.$tmp_name,$file_path,$mime);
		return $tmp_name;
	}
	public function renameTempFile($temp_name)
	{
		$tempkey = static::TEMP_DIR.$temp_name;
		$newkey = $this->getFileKey();
		Storage::rename($tempkey,$newkey);
	}
	public function deleteFile()
	{
		$key = $this->getFileKey();
		Storage::delete($key);
	}
	public function getFileUrl($expire=null)
	{
		return Storage::url($this->getFileKey(),$expire,$this->getOriginalFileName());
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

	public function updateInfo($title,$description,$protect,TagSet $tags,$con=null)
	{
		$this->row['title'] = $title;
		$this->row['description'] = $description;
		$this->row['protect'] = $protect? 1: 0;
		$this->update($con);
		TagDb::removeFromPackage($this,$con);
		TagDb::insertPackageTags($this,$tags,$con);
		$this->tags = $tags;
	}

	public function delete($con=null)
	{
		TagDb::removeFromPackage($this,$con);
		$this->getAttachedFiles()->delete($con);
		return parent::delete($con);
	}

	/**
	 * packageに紐付くguestpass一覧を取得する
	 */
	public function getGuestPasses()
	{
		if($this->guest_passes===null){
			$this->guest_passes = GuestPassDb::selectByPackageId($this->getId());
		}
		return $this->guest_passes;
	}

	public function getAttachedFiles()
	{
		if($this->attached_files===null){
			$this->attached_files = AttachedFileDb::selectByPackageId($this->getId());
		}
		return $this->attached_files;
	}

	public function isAndroidAppBundle()
	{
		return $this->getPlatform()===self::PF_ANDROID
			&& pathinfo($this->getOriginalFileName(),PATHINFO_EXTENSION)==='aab';
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

	public static function getPackageInfo($originalname,$filepath,$mime)
	{
		$platform = Package::PF_UNKNOWN;
		$ext = pathinfo($originalname,PATHINFO_EXTENSION);
		$is_zip = file_get_contents($filepath,false,null,0,4)==="PK\x03\x04";
		if($is_zip && $ext==='apk'){
			$platform = Package::PF_ANDROID;
			$mime = Package::MIME_ANDROID_APK;
		}
		if($is_zip && $ext=='aab'){
			$platform = Package::PF_ANDROID;
			$mime = Package::MIME_ANDROID_AAB;
		}
		if($is_zip && $ext==='ipa'){
			$platform = Package::PF_IOS;
			$mime = Package::MIME_IOS;
		}
		return array($platform,$ext,$mime);
	}

	public static function insertNewPackage($app_id,$platform,$ext,$title,$description,$identifier,$org_file_name,$file_size,TagSet $tags,$protect,$con)
	{
		$row = array(
			'app_id' => $app_id,
			'platform' => $platform,
			'file_name' => Random::string(16).".$ext",
			'title' => $title,
			'description' => $description,
			'identifier' => $identifier,
			'original_file_name' => $org_file_name,
			'file_size' => $file_size,
			'protect' => $protect? 1: 0,
			'created' => date('Y-m-d H:i:s'),
			);
		$pkg = new Package($row);
		$pkg->insert($con);
		$pkg->initTags($tags,$con);
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

	public static function selectNewestOneByAppId($app_id)
	{
		$query = 'WHERE app_id = :app_id ORDER BY id DESC LIMIT 1';
		$bind = array(':app_id' => $app_id);
		return static::selectOne($query,$bind);
	}

	public static function selectByAppIdPfTagsWithLimit($app_id, $pf_filter, array $tags, $offset, $count)
	{
		if (!$pf_filter && empty($tags)) {
			$query = 'WHERE app_id = :app_id';
			$query .= sprintf(' ORDER BY id DESC LIMIT %d, %d', $offset, $count);
			return static::selectSet($query, array(':app_id' => $app_id));
		}
		$sql = 'SELECT p.* FROM package AS p LEFT JOIN package_tag AS t ON p.id = t.package_id WHERE p.app_id = :app_id';
		$bind = array(':app_id' => $app_id);

		if ($pf_filter) {
			$sql .= ' AND p.platform = :platform';
			$bind[':platform'] = $pf_filter;
		}

		if (!empty($tags)) {
			$ph = static::makeInPlaceHolder($tags,$bind,'tag');
			$c = count($tags);
			$sql .= " AND t.tag_id in ($ph) GROUP BY p.id HAVING COUNT(p.id) = $c";
		} else {
			$sql .= " GROUP BY p.id";
		}

		$sql .= sprintf(' ORDER BY p.id DESC LIMIT %d, %d', $offset, $count);
		return new PackageSet(mfwDBIBase::getAll($sql, $bind));
	}

	/**
	 * 削除すべきパッケージを取得.
	 * @param[in] Application $app 対象アプリ
	 * @param[in] int $keep 削除ぜず保持する上限数
     * @param[in] int $limit 取得する上限数
	 */
	public function selectDeletablePackages(Application $app,$keep=1000,$limit=100,$con=null)
	{
		$sql = sprintf(
			'WHERE app_id=:app_id AND protect=0 AND id not in '
			. '(select * from (select id from package '
			.	'WHERE app_id=:app_id AND protect=0 ORDER BY id DESC LIMIT %d) t'
			. ') ORDER BY id LIMIT %d', $keep, $limit);
		return self::selectSet($sql,array(':app_id' => $app->getId()),$con);
	}
}
