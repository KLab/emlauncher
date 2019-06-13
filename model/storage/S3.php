<?php
require_once APP_ROOT.'/model/Config.php';
require_once APP_ROOT.'/model/Storage.php';

class S3 implements StorageImpl {

	protected $config;
	protected $client;

	public function __construct()
	{
		$this->config = Config::get('aws');
		$this->client = Aws\S3\S3Client::factory(
			array(
				'key' => $this->config['key'],
				'secret' => $this->config['secret'],
				'base_url' => isset($this->config['base_url']) ? $this->config['base_url'] : NULL,
				));
	}

	public function saveIcon($key,$data)
	{
		$type = 'image/png';
		$acl = 'public-read';
		$pathstyle = isset($this->config['base_url']) ? true : false;
		$r = $this->client->putObject(
			array(
				'Bucket' => $this->config['bucket_name'],
				'Key' => $key,
				'ACL' => $acl,
				'ContentType' => $type,
				'Body' => Guzzle\Http\EntityBody::factory($data),
				'PathStyle' => $pathstyle,
				));
		return $r;
	}

	public function saveFile($key,$filename,$mime)
	{
		$acl = 'private';
		$fp = fopen($filename,'rb');
		$pathstyle = isset($this->config['base_url']) ? true : false;
		$r = $this->client->putObject(
			array(
				'Bucket' => $this->config['bucket_name'],
				'Key' => $key,
				'ACL' => $acl,
				'ContentType' => $mime,
				'Body' => $fp,
				'PathStyle' => $pathstyle,
				));
		// Guzzleが中で勝手にfcloseしやがるのでここでfcloseしてはならない
		// fclose($fp)
		return $r;
	}

	public function rename($srckey,$dstkey)
	{
		$acl = 'private';
		$bucket = $this->config['bucket_name'];
		$pathstyle = isset($this->config['base_url']) ? true : false;

		// copy
		$this->client->copyObject(
			array(
				'Bucket' => $bucket,
				'Key' => $dstkey,
				'ACL' => $acl,
				'CopySource' => "{$bucket}/{$srckey}",
				'PathStyle' => $pathstyle,
				));
		// delete
		$this->client->deleteObject(
			array(
				'Bucket' => $bucket,
				'Key' => $srckey,
				'PathStyle' => $pathstyle,
				));
	}

	public function delete($key)
	{
		$bucket = $this->config['bucket_name'];
		$pathstyle = isset($s3->config['base_url']) ? true : false;
		$this->client->deleteObject(
			array(
				'Bucket' => $bucket,
				'Key' => $key,
				'PathStyle' => $pathstyle,
				));
	}

	public function url($key,$expires=null)
	{
		$bucket = $this->config['bucket_name'];
		if($expires===null){
			if(isset($this->config['s3_external_url'])){
				$s3_external_url = rtrim($this->config['s3_external_url'], '/');
				error_log("url is {$s3_external_url}/{$bucket}/{$key}");
				return "{$s3_external_url}/{$bucket}/{$key}";
			}else{
				return "https://{$bucket}.s3.amazonaws.com/{$key}";
			}
		}
		if(isset($this->config['base_url'])){
			$obj_url = $this->client->getObjectUrl($bucket,$key,$expires,array('PathStyle' => true));
			if(isset($this->config['s3_external_url'])){
				$obj_url = str_replace(
					rtrim($this->config['base_url'], '/'),
					rtrim($this->config['s3_external_url'], '/'),
					$obj_url
				);
			}
			return $obj_url;
		}
		return $this->client->getObjectUrl($bucket,$key,$expires);
	}

}
