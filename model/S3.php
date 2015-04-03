<?php
require_once APP_ROOT . '/model/Config.php';
require_once APP_ROOT . '/vendor/autoload.php';

use WindowsAzure\Common\ServicesBuilder;
use WindowsAzure\Blob\Models\CreateContainerOptions;
use WindowsAzure\Blob\Models\PublicAccessType;
use WindowsAzure\Common\ServiceException;
class S3 {
	protected static $singleton = null;
	protected $config;
	protected $client;
	protected $storageType;
	protected function __construct() {
		$this->storageType = Config::has ( 'aws' ) ? 'aws' : 'azure';

		switch ($this->storageType) {
			case 'aws' :
				$this->config = Config::get ( 'aws' );
				$this->client = Aws\S3\S3Client::factory ( array (
						'key' => $this->config ['key'],
						'secret' => $this->config ['secret']
				) );
				break;

			case 'azure' :
				$this->config = Config::get ( 'azure' );
				$this->client = ServicesBuilder::getInstance ();

				break;

			default :
				throw new Exception ( 'Storage configuration is invalid.' );
				break;
		}
	}
	protected function singleton() {
		if (static::$singleton === null) {
			static::$singleton = new static ();
		}
		return static::$singleton;
	}
	public static function uploadData($key, $data, $type, $acl = 'private') {
		$myInstance = static::singleton ();

		switch ($myInstance->storageType) {
			case 'aws' :
				return $myInstance->client->putObject ( array (
						'Bucket' => $myInstance->config ['bucket_name'],
						'Key' => $key,
						'ACL' => $acl,
						'ContentType' => $type,
						'Body' => Guzzle\Http\EntityBody::factory ( $data )
				) );

			case 'azure' :
				$blobService = $myInstance->client->createBlobService ( $myInstance->config ['connectionString'] );
				return $blobService->createBlockBlob ( $myInstance->config ['container'], $key, ( string ) $data );
		}
	}
	public static function uploadFile($key, $filename, $type, $acl = 'private') {
		$myInstance = static::singleton ();
		switch ($myInstance->storageType) {
			case 'aws' :

				$fp = fopen ( $filename, 'rb' );
				$r = $myInstance->client->putObject ( array (
						'Bucket' => $myInstance->config ['bucket_name'],
						'Key' => $key,
						'ACL' => $acl,
						'ContentType' => $type,
						'Body' => $fp
				) );
				// Guzzleが中で勝手にfcloseしやがるのでここでfcloseしてはならない
				// fclose($fp)
				return $r;

			case 'azure' :
				$blobService = $myInstance->client->createBlobService ( $myInstance->config ['connectionString'] );
				return $blobService->createBlockBlob ( $myInstance->config ['container'], $key, file_get_contents ( $filename ) );
		}
	}
	public static function rename($srckey, $dstkey, $acl = 'private') {
		$myInstance = static::singleton ();
		switch ($myInstance->storageType) {
			case 'aws' :
				$bucket = $myInstance->config ['bucket_name'];

				// copy
				$myInstance->client->copyObject ( array (
						'Bucket' => $bucket,
						'Key' => $dstkey,
						'ACL' => $acl,
						'CopySource' => "{$bucket}/{$srckey}"
				) );
				// delete
				$myInstance->client->deleteObject ( array (
						'Bucket' => $bucket,
						'Key' => $srckey
				) );
				break;
			case 'azure' :
				$blobService = $myInstance->client->createBlobService ( $myInstance->config ['connectionString'] );
				$content = $blobService->getBlob ( $myInstance->config ['container'], $srckey )->getContentStream ();
				$blobService->createBlockBlob ( $myInstance->config ['container'], $dstkey, $content );
				$blobService->deleteBlob ( $myInstance->config ['container'], $srckey );
				break;
		}
	}
	public static function delete($key) {
		$myInstance = static::singleton ();
		switch ($myInstance->storageType) {
			case 'aws' :
				$bucket = $myInstance->config ['bucket_name'];
				$myInstance->client->deleteObject ( array (
						'Bucket' => $bucket,
						'Key' => $key
				) );
				break;

			case 'azure' :
				$blobService = $myInstance->client->createBlobService ( $myInstance->config ['connectionString'] );
				$blobService->deleteBlob ( $myInstance->config ['container'], $key );
				break;
		}
	}
	public static function url($key, $expires = null) {
		$myInstance = static::singleton ();
		switch ($myInstance->storageType) {
			case 'aws' :
				$bucket = $myInstance->config ['bucket_name'];
				if ($expires === null) {
					return "https://{$bucket}.s3.amazonaws.com/{$key}";
				}
				return $myInstance->client->getObjectUrl ( $bucket, $key, $expires );

			case 'azure' :
				$baseUrl = $myInstance->config ['url'];
				return "{$baseUrl}/{$key}";
		}
	}
}

