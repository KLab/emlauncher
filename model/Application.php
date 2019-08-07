<?php
require_once __DIR__.'/ApplicationOwner.php';
require_once __DIR__.'/Tag.php';
require_once __DIR__.'/InstallLog.php';
require_once __DIR__.'/Storage.php';
require_once __DIR__.'/Random.php';

/**
 * Row object for 'application' table.
 */
class Application extends mfwObject {
	const DB_CLASS = 'ApplicationDb';
	const SET_CLASS = 'ApplicationSet';

	protected $owners = null;
	protected $tags = null;
	protected $unused_tags = null;
	protected $install_users = null;

	public function getId(){
		return $this->value('id');
	}
	public function getTitle(){
		return $this->value('title');
	}
	public function getDescription(){
		return $this->value('description');
	}
	public function getRepository(){
		return $this->value('repository');
	}
	public function getIconUrl()
	{
		return Storage::url($this->value('icon_key'));
	}

	public function getLastUpload($format=null){
		$last_upload = $this->value('last_upload');
		if($format){
			$last_upload = date($format,strtotime($last_upload));
		}
		return $last_upload;
	}

	public function getLastCommented($format=null){
		$last_commented = $this->value('last_commented');
		if($format){
			$last_commented = date($format,strtotime($last_commented));
		}
		return $last_commented;
	}
	public function getDateToSort($format=null){
		$date_to_sort = $this->value('date_to_sort');
		if($format){
			$date_to_sort = date($format,strtotime($date_to_sort));
		}
		return $date_to_sort;
	}

	protected function calcDateToSort()
	{
		$this->row['date_to_sort'] = max(
			$this->getCreated(),
			$this->getLastUpload(),
			$this->getLastCommented());
	}

	public function updateLastUpload($date,$con=null)
	{
		$this->row['last_upload'] = $date;
		$this->calcDateToSort();

		$sql = 'UPDATE application SET last_upload = :last_upload, date_to_sort = :date_to_sort WHERE id = :id';
		$bind = array(
			':last_upload' => $this->getLastUpload(),
			':date_to_sort' => $this->getDateToSort(),
			':id' => $this->getId(),
			);
		mfwDBIBase::query($sql,$bind,$con);
	}

	public function updateLastCommented($date,$con=null)
	{
		$this->row['last_commented'] = $date;
		$this->calcDateToSort();

		$sql = 'UPDATE application SET last_commented = :last_commented, date_to_sort = :date_to_sort WHERE id = :id';
		$bind = array(
			':last_commented' => $this->getLastCommented(),
			':date_to_sort' => $this->getDateToSort(),
			':id' => $this->getId(),
			);
		mfwDBIBase::query($sql,$bind,$con);
	}

	public function getAPIKey()
	{
		return $this->value('api_key');
	}
	public function refreshApiKey($con=null)
	{
		$this->row['api_key'] = ApplicationDb::makeApiKey();
		$sql = 'UPDATE application SET api_key = :api_key WHERE id = :id';
		$bind = array(
			':id' => $this->getId(),
			':api_key' => $this->getApiKey(),
			);
		mfwDBIBase::query($sql,$bind,$con);
	}

	public function getCreated($format=null){
		$created = $this->value('created');
		if($format){
			$created = date($format,strtotime($created));
		}
		return $created;
	}
	public function getOwners()
	{
		if($this->owners===null){
			$this->owners = ApplicationOwnerDb::selectByAppId($this->getId());
		}
		return $this->owners;
	}
	public function isOwner(User $user)
	{
		$owners = $this->getOwners();
		$k = $owners->searchPK('owner_mail',$user->getMail());
		return $k!==null;
	}
	public function setOwners(array $owner_mails,$con=null)
	{
		$cur_mails = $this->getOwners()->getMailArray();

		$delete = array_diff($cur_mails,$owner_mails);
		$add = array_diff($owner_mails,$cur_mails);

		if(!empty($delete)){
			ApplicationOwnerDb::deleteOwner($this->getId(),$delete,$con);
		}
		if(!empty($add)){
			ApplicationOwnerDb::addOwner($this->getId(),$add,$con);
		}
		$this->owners = null;
	}

	public function getTags()
	{
		if($this->tags===null){
			$this->tags = TagDb::selectByAppId($this->getId());
		}
		return $this->tags;
	}

	public function getUnusedTags()
	{
		if($this->unused_tags===null){
			$this->unused_tags = TagDb::selectUnusedTagsByAppId($this->getId());
		}
		return $this->unused_tags;
	}

	public function getInstallUsers()
	{
		if($this->install_users===null){
			$this->install_users = InstallLog::getInstallUsers($this);
		}
		return $this->install_users;
	}

	public function getInstallUserCount()
	{
		$install_users = $this->getInstallUsers();
		return $install_users->count();
	}

	/**
	 * タグ名からTagSetを取得.
	 * 新しいtag_nameがあったら登録もする.
	 */
	public function getOrInsertTagsByName($tag_names,PDO $con=null)
	{
		if(empty($tag_names)){
			return new TagSet();
		}
		$this->tags = TagDb::selectByAppIdForUpdate($this->getId(),$con);
		$tags = new TagSet();
		// タグの数はたかが知れているので、愚直に一つずつ探す
		foreach($tag_names as $name){
			if(!$name){
				continue;
			}
			$pk = $this->tags->searchPK('name',$name);
			if($pk){
				$tags[] = $this->tags[$pk];
			}
			else{
				$tag = TagDb::insertNewTag($this->getId(),$name,$con);
				$tags[] = $tag;
				$this->tags[] = $tag;
			}
		}
		return $tags;
	}

	/**
	 * タグ名からTagSetを取得.
	 */
	public function getTagsByName($tag_names,PDO $con=null)
	{
		if(empty($tag_names)){
			return new TagSet();
		}
		$this->tags = TagDb::selectByAppId($this->getId(),$con);
		$tags = new TagSet();
		// タグの数はたかが知れているので、愚直に一つずつ探す
		foreach($tag_names as $name){
			if(!$name){
				continue;
			}
			$pk = $this->tags->searchPK('name',$name);
			if($pk){
				$tags[] = $this->tags[$pk];
			}
		}
		return $tags;
	}

	public function deleteTags($tag_names,PDO $con=null)
	{
		$tags = TagDb::selectByAppIdForUpdate($this->getId(),$con);
		$delete_ids = array();
		$this->tags = new TagSet();
		foreach($tags as $tag){
			if(in_array($tag->getName(),$tag_names)){
				$delete_ids[] = $tag->getId();
			}
			else{
				$this->tags[] = $tag;
			}
		}
		TagDb::deleteByIds($delete_ids,$con);
	}

	public function deleteTagsByIds($tag_ids,PDO $con=null)
	{
		$tags = TagDb::selectByAppIdForUpdate($this->getId(),$con);
		$delete_ids = array();
		$this->tags = new TagSet();
		foreach($tags as $tag){
			if(in_array($tag->getId(),$tag_ids)){
				$delete_ids[] = $tag->getId();
			}
			else{
				$this->tags[] = $tag;
			}
		}
		TagDb::deleteByIds($delete_ids,$con);
	}

	public function updateInfo($title,$image,$description,$repository,$con=null)
	{
		$this->row['title'] = $title;
		$this->row['description'] = $description;
		$this->row['repository'] = $repository;

		$old_icon_key = null;
		if($image){
			$old_icon_key = $this->value('icon_key');
			$this->row['icon_key'] = ApplicationDb::saveIcon($image,$this->getId());
		}
		$this->update($con);

		if($old_icon_key){
			try{
				Storage::delete($old_icon_key);
			}
			catch(Exception $e){
				error_log(__METHOD__.'('.__LINE__.'): '.get_class($e).":{$e->getMessage()}");
				// 画像削除は失敗しても気にしない
			}
		}
	}

}

/**
 * Set of Application objects.
 */
class ApplicationSet extends mfwObjectSet {
	public static function hypostatize(Array $row=array())
	{
		return new Application($row);
	}
	protected function unsetCache($id)
	{
		parent::unsetCache($id);
	}
}

/**
 * database accessor for 'application' table.
 */
class ApplicationDb extends mfwObjectDb {
	const TABLE_NAME = 'application';
	const SET_CLASS = 'ApplicationSet';

	const ICON_DIR = 'app-icons/';

	public static function saveIcon($image,$app_id)
	{
		$im = new Imagick();
		$im->readImageBlob($image);
		$im->scaleImage(144,144);
		$im->setFormat('png');

		$key = static::ICON_DIR."$app_id/".Random::string(16).'.png';
		Storage::saveIcon($key,$im);

		return $key;
	}

	public static function makeApiKey()
	{
		do{
			$api_key = Random::string();
		}while(static::selectByApiKey($api_key));
		return $api_key;
	}

	public static function selectByApiKey($key)
	{
		$query = 'WHERE api_key = ?';
		return static::selectOne($query,array($key));
	}

	public static function insertNewApp($owner,$title,$image,$description,$repository)
	{
		$now = date('Y-m-d H:i:s');
		// insert new application
		$row = array(
			'title' => $title,
			'api_key' => static::makeApiKey(),
			'description' => $description,
			'repository' => $repository,
			'date_to_sort' => $now,
			'created' => $now,
			);
		$app = new Application($row);
		$app->insert();

		// save icon to Storage
		$icon_key = static::saveIcon($image,$app->getId());

		$table = static::TABLE_NAME;
		mfwDBIBase::query(
			"UPDATE $table SET icon_key = :icon_key WHERE id= :id",
			array(':id'=>$app->getId(),':icon_key'=>$icon_key));

		// insert owner
		$row = array(
			'app_id' => $app->getId(),
			'owner_mail' => $owner->getMail(),
			);
		$owner = new ApplicationOwner($row);
		$owner->insert();

		return $app;
	}

	public static function selectAllByUpdateOrder()
	{
		$query = 'ORDER BY date_to_sort DESC';
		return static::selectSet($query);
	}

	public static function selectCount()
	{
		$table = static::TABLE_NAME;
		$sql = "SELECT count(*) FROM `$table`";
		return mfwDBIBase::getOne($sql);
	}

	public static function selectByUpdateOrderWithLimit($offset, $count)
	{
		$query = sprintf('ORDER BY date_to_sort DESC LIMIT %d, %d', $offset, $count);
		return static::selectSet($query);
	}

	public static function selectOwnApps($user)
	{
		$aos = ApplicationOwnerDb::selectByOwnerMail($user->getMail());
		if($aos->count()==0){
			return new ApplicationSet();
		}
		$ids = array();
		foreach($aos as $ao){
			$ids[] = $ao->getAppId();
		}
		$bind = array();
		$pf = static::makeInPlaceholder($ids,$bind);
		return static::selectSet("WHERE id IN ($pf) ORDER BY id DESC",$bind);
	}
}

