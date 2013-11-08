<?php
require_once __DIR__.'/ApplicationOwner.php';
require_once __DIR__.'/S3.php';
require_once __DIR__.'/Random.php';

/**
 * Row object for 'application' table.
 */
class Application extends mfwObject {
	const DB_CLASS = 'ApplicationDb';
	const SET_CLASS = 'ApplicationSet';

	protected $owners = null;

	public function getId(){
		return $this->value('id');
	}
	public function getTitle(){
		return $this->value('title');
	}
	public function getDescription(){
		return $this->value('description');
	}
	public function getIconUrl()
	{
		return S3::url($this->value('icon_key'));
	}
	public function getCreated()
	{
		return $this->value('created');
	}
	public function getOwners()
	{
		if($this->owners===null){
			$this->owners = ApplicationOwnerDb::selectByAppId($this->getId());
		}
		return $this->owners;
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

	protected static function uploadIcon($image,$app_id)
	{
		$im = new Imagick();
		$im->readImageBlob($image);
		$im->scaleImage(144,144);
		$im->setFormat('png');

		$key = static::ICON_DIR."$app_id/".Random::string(16).'.png';
		S3::upload($key,$im,'image/png','public-read');

		return $key;
	}

	protected static function makeApiKey()
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

	public static function insertNewApp($owner,$title,$image,$description)
	{
		// insert new application
		$row = array(
			'title' => $title,
			'api_key' => static::makeApiKey(),
			'description' => $description,
			'created' => date('Y-m-d H:i:s'),
			);
		$app = new Application($row);
		$app->insert();

		// upload icon to S3
		$icon_key = static::uploadIcon($image,$app->getId());

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

}

