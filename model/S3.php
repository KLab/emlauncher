<?php
require_once APP_ROOT.'/model/Config.php';

class S3 {

	protected static $singleton = null;
	protected $config;
	protected $client;

	protected function __construct()
	{
		$this->config = Config::get('aws');
		$this->client = Aws\S3\S3Client::factory(
			array(
				'key' => $this->config['key'],
				'secret' => $this->config['secret'],
				'base_url' => isset($this->config['base_url']) ? $this->config['base_url'] : NULL,
				));
	}
	protected function singleton()
	{
		if(static::$singleton===null){
			static::$singleton = new static();
		}
		return static::$singleton;
	}

	public static function uploadData($key,$data,$type,$acl='private')
	{
		$s3 = static::singleton();
		$pathstyle = isset($s3->config['base_url']) ? true : false;
		$r = $s3->client->putObject(
			array(
				'Bucket' => $s3->config['bucket_name'],
				'Key' => $key,
				'ACL' => $acl,
				'ContentType' => $type,
				'Body' => Guzzle\Http\EntityBody::factory($data),
				'PathStyle' => $pathstyle,
				));
		return $r;
	}

	public static function uploadFile($key,$filename,$type,$acl='private')
	{
		$s3 = static::singleton();
		$pathstyle = isset($s3->config['base_url']) ? true : false;
		$fp = fopen($filename,'rb');
		$r = $s3->client->putObject(
			array(
				'Bucket' => $s3->config['bucket_name'],
				'Key' => $key,
				'ACL' => $acl,
				'ContentType' => $type,
				'Body' => $fp,
				'PathStyle' => $pathstyle,
				));
		// Guzzleが中で勝手にfcloseしやがるのでここでfcloseしてはならない
		// fclose($fp)
		return $r;
	}

	public static function rename($srckey,$dstkey,$acl='private')
	{
		$s3 = static::singleton();
		$bucket = $s3->config['bucket_name'];
		$pathstyle = isset($s3->config['base_url']) ? true : false;

		// copy
		$s3->client->copyObject(
			array(
				'Bucket' => $bucket,
				'Key' => $dstkey,
				'ACL' => $acl,
				'CopySource' => "{$bucket}/{$srckey}",
				'PathStyle' => $pathstyle,
				));
		// delete
		$s3->client->deleteObject(
			array(
				'Bucket' => $bucket,
				'Key' => $srckey,
				'PathStyle' => $pathstyle,
				));
	}

	public static function delete($key)
	{
		$s3 = static::singleton();
		$bucket = $s3->config['bucket_name'];
		$pathstyle = isset($s3->config['base_url']) ? true : false;
		$s3->client->deleteObject(
			array(
				'Bucket' => $bucket,
				'Key' => $key,
				'PathStyle' => $pathstyle,
				));
	}

	public static function url($key,$expires=null)
	{
		$s3 = static::singleton();
		$bucket = $s3->config['bucket_name'];
		if($expires===null){
			if(isset($s3->config['s3_external_url'])){
				$s3_external_url = rtrim($s3->config['s3_external_url'], '/');
				error_log("url is {$s3_external_url}/{$bucket}/{$key}");
				return "{$s3_external_url}/{$bucket}/{$key}";
			}else{
				return "https://{$bucket}.s3.amazonaws.com/{$key}";
			}
		}
		if(isset($s3->config['base_url'])){
			$obj_url = $s3->client->getObjectUrl($bucket,$key,$expires,array('PathStyle' => true));
			if(isset($s3->config['s3_external_url'])){
				$obj_url = str_replace(
					rtrim($s3->config['base_url'], '/'),
					rtrim($s3->config['s3_external_url'], '/'),
					$obj_url
				);
			}
			return $obj_url;
		}
		return $s3->client->getObjectUrl($bucket,$key,$expires);
	}

}

