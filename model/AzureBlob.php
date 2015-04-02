<?php
require_once APP_ROOT . '/model/Config.php';
require_once APP_ROOT . '/vendor/autoload.php';

use WindowsAzure\Common\ServicesBuilder;
use WindowsAzure\Blob\Models\CreateContainerOptions;
use WindowsAzure\Blob\Models\PublicAccessType;
use WindowsAzure\Common\ServiceException;
class AzureBlob {
	protected static $singleton = null;
	protected $config;
	protected $client;
	protected $storageType;
	protected function __construct() {
		$this->config = Config::get ( 'azure' );
		$this->client = ServicesBuilder::getInstance ();
	}
	protected function singleton() {
		if (static::$singleton === null) {
			static::$singleton = new static ();
		}
		return static::$singleton;
	}
	public static function uploadData($key, $data, $type, $acl = 'private') {
		$myInstance = static::singleton ();
		$blobService = $myInstance->client->createBlobService ( $myInstance->config ['connectionString'] );
		return $blobService->createBlockBlob ( $myInstance->config ['container'], $key, ( string ) $data );
	}
	public static function uploadFile($key, $filename, $type, $acl = 'private') {
		$myInstance = static::singleton ();
		$fp = fopen ( $filename, 'rb' );
		$blobService = $myInstance->client->createBlobService ( $myInstance->config ['connectionString'] );
		return $blobService->createBlockBlob ( $myInstance->config ['container'], $key, $fp );
	}
	public static function rename($srckey, $dstkey, $acl = 'private') {
		$myInstance = static::singleton ();
		$blobService = $myInstance->client->createBlobService ( $myInstance->config ['connectionString'] );
		$content = $blobService->getBlob ( $myInstance->config ['container'], $srckey )->getContentStream ();
		$blobService->createBlockBlob ( $myInstance->config ['container'], $dstkey, $content );
		$blobService->deleteBlob ( $myInstance->config ['container'], $srckey );
	}
	public static function delete($key) {
		$myInstance = static::singleton ();
		$blobService = $myInstance->client->createBlobService ( $myInstance->config ['connectionString'] );
		$blobService->deleteBlob ( $myInstance->config ['container'], $key );
	}
	public static function url($key, $expires = null) {
		$myInstance = static::singleton ();
		$baseUrl = $myInstance->config ['url'];
		return "{$baseUrl}/{$key}";
	}
}

