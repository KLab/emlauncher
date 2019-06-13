<?php
require_once APP_ROOT.'/model/Config.php';
require_once APP_ROOT.'/model/storage/S3.php';
require_once APP_ROOT.'/model/storage/LocalFile.php';

interface StorageImpl {
	public function saveIcon($key, $data);
	public function saveFile($key, $filename, $mime);
	public function rename($srckey, $dstkey);
	public function delete($key);
	public function url($key, $expires=null);
}

class Storage {

	protected static $singleton = null;

	private static function singleton()
	{
		if(static::$singleton===null){
			$class = Config::get('storage_class');
			static::$singleton = new $class();
		}
		return static::$singleton;
	}

	public static function saveIcon($name, $data)
	{
		$storage = static::singleton();
		return $storage->saveIcon($name, $data);
	}

	public static function saveFile($name, $fileName, $mime)
	{
		$storage = static::singleton();
		return $storage->saveFile($name, $fileName, $mime);
	}

	public static function rename($name, $newName)
	{
		$storage = static::singleton();
		return $storage->rename($name, $newName);
	}

	public static function delete($name)
	{
		$storage = static::singleton();
		return $storage->delete($name);
	}

	public static function url($name, $expire=null)
	{
		$storage = static::singleton();
		return $storage->url($name, $expire);
	}
}
