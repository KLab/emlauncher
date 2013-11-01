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

	const S3_DIR_NAME = 'app-icons/';

	public function getId(){
		return $this->value('id');
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

	const TEMP_ICON_DIR = 'tmp-icons/';
	const ICON_DIR = 'icons/';

	protected static function uploadIconTemporary($image)
	{
		$im = new Imagick();
		$im->readImageBlob($image);
		$im->scaleImage(144,144);
		$im->setFormat('png');
		$name = Random::string(8).'.png';

		S3::upload(
			static::TEMP_ICON_DIR.$name,$im,
			'image/png','private','+5 minits');

		return $name;
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
		$icon_name = static::uploadIconTemporary($image);
		$api_key = static::makeApiKey();

		// insert new application
		$row = array(
			'title' => $title,
			'api_key' => $api_key,
			'description' => $description,
			);
		$app = new Application($row);
		$app->insert();

		// set icon image url
		$icon_key = static::ICON_DIR."{$app->getId()}/$icon_name";
		S3::rename(static::TEMP_ICON_DIR.$icon_name,$icon_key,'public-read');
		$url = S3::url($icon_key);
		$table = static::TABLE_NAME;
		mfwDBIBase::query(
			"UPDATE $table SET icon_url = :url WHERE id= :id",
			array(':id'=>$app->getId(),':url'=>$url));

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

