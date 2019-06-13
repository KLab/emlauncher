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
				));
	}

	public function saveIcon($key,$data)
	{
		$type = 'image/png';
		$acl = 'public-read';
		$r = $this->client->putObject(
			array(
				'Bucket' => $this->config['bucket_name'],
				'Key' => $key,
				'ACL' => $acl,
				'ContentType' => $type,
				'Body' => Guzzle\Http\EntityBody::factory($data),
				));
		return $r;
	}

	public function saveFile($key,$filename,$mime)
	{
		$acl = 'private';
		$fp = fopen($filename,'rb');
		$r = $this->client->putObject(
			array(
				'Bucket' => $this->config['bucket_name'],
				'Key' => $key,
				'ACL' => $acl,
				'ContentType' => $mime,
				'Body' => $fp,
				));
		// Guzzleが中で勝手にfcloseしやがるのでここでfcloseしてはならない
		// fclose($fp)
		return $r;
	}

	public function rename($srckey,$dstkey)
	{
		$acl = 'private';
		$bucket = $this->config['bucket_name'];

		// copy
		$this->client->copyObject(
			array(
				'Bucket' => $bucket,
				'Key' => $dstkey,
				'ACL' => $acl,
				'CopySource' => "{$bucket}/{$srckey}",
				));
		// delete
		$this->client->deleteObject(
			array(
				'Bucket' => $bucket,
				'Key' => $srckey,
				));
	}

	public function delete($key)
	{
		$bucket = $this->config['bucket_name'];
		$this->client->deleteObject(
			array(
				'Bucket' => $bucket,
				'Key' => $key,
				));
	}

	public function url($key,$expires=null)
	{
		$bucket = $this->config['bucket_name'];
		if($expires===null){
			return "https://{$bucket}.s3.amazonaws.com/{$key}";
		}
		return $this->client->getObjectUrl($bucket,$key,$expires);
	}

}
