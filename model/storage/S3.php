<?php
require_once APP_ROOT.'/libs/aws/aws-autoloader.php';
require_once APP_ROOT.'/model/Config.php';
require_once APP_ROOT.'/model/Storage.php';

class S3 implements StorageImpl {

	protected $bucket;
	protected $pathstyle;
	protected $base_url;
	protected $external_url;
	protected $client;

	public function __construct()
	{
		$config = Config::get('aws');
		$this->bucket = $config['bucket_name'];
		if(isset($config['base_url'])){
			$this->pathstyle = TRUE;
			$this->base_url = rtrim($config['base_url'],'/');
			$this->external_url = NULL;
			if(isset($config['external_url'])){
				$this->external_url = rtrim($config['external_url'], '/');
			}
		}
		else{
			$this->pathstyle = false;
			$this->base_url = NULL;
			$this->external_url = NULL;
		}

		$this->client = Aws\S3\S3Client::factory(
			array(
				'key' => $config['key'],
				'secret' => $config['secret'],
				'base_url' => $this->base_url,
				));
	}

	public function saveIcon($key,$data)
	{
		$type = 'image/png';
		$acl = 'public-read';
		$r = $this->client->putObject(
			array(
				'Bucket' => $this->bucket,
				'Key' => $key,
				'ACL' => $acl,
				'ContentType' => $type,
				'Body' => Guzzle\Http\EntityBody::factory($data),
				'PathStyle' => $this->pathstyle,
				));
		return $r;
	}

	public function saveFile($key,$filename,$mime)
	{
		$acl = 'private';
		$fp = fopen($filename,'rb');
		$r = $this->client->putObject(
			array(
				'Bucket' => $this->bucket,
				'Key' => $key,
				'ACL' => $acl,
				'ContentType' => $mime,
				'Body' => $fp,
				'PathStyle' => $this->pathstyle,
				));
		// Guzzleが中で勝手にfcloseしやがるのでここでfcloseしてはならない
		// fclose($fp)
		return $r;
	}

	public function rename($srckey,$dstkey)
	{
		$acl = 'private';

		// copy
		$this->client->copyObject(
			array(
				'Bucket' => $this->bucket,
				'Key' => $dstkey,
				'ACL' => $acl,
				'CopySource' => "{$this->bucket}/{$srckey}",
				'PathStyle' => $this->pathstyle,
				));
		// delete
		$this->client->deleteObject(
			array(
				'Bucket' => $this->bucket,
				'Key' => $srckey,
				'PathStyle' => $this->pathstyle,
				));
	}

	public function delete($key)
	{
		$this->client->deleteObject(
			array(
				'Bucket' => $this->bucket,
				'Key' => $key,
				'PathStyle' => $this->pathstyle,
				));
	}

	public function url($key,$expires=null)
	{
		$bucket = $this->bucket;
		if($expires===null){
			if($this->base_url===NULL){
				return "https://{$bucket}.s3.amazonaws.com/{$key}";
			}
			$base_url = $this->external_url ?: $this->base_url;
			return "{$base_url}/{$bucket}/{$key}";
		}

		$obj_url = $this->client->getObjectUrl($bucket,$key,$expires,array('PathStyle' => $this->pathstyle));
		if($this->external_url){
			$obj_url = str_replace($this->base_url, $this->external_url, $obj_url);
		}
		return $obj_url;
	}
}
