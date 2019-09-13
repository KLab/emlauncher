<?php
require_once APP_ROOT.'/model/Config.php';
require_once APP_ROOT.'/model/Storage.php';

use Aws\S3\S3Client;
use Aws\Credentials\Credentials;

class S3 implements StorageImpl {

	protected $bucket;
	protected $base_url;
	protected $external_url;
	protected $client;

	public function __construct()
	{
		$config = Config::get('aws');
		$this->bucket = $config['bucket_name'];
		if(isset($config['base_url'])){
			$this->base_url = rtrim($config['base_url'],'/');
			$this->external_url = NULL;
			if(isset($config['external_url'])){
				$this->external_url = rtrim($config['external_url'], '/');
			}
		}
		else{
			$this->base_url = NULL;
			$this->external_url = NULL;
		}

		$credentials = null;
		if($config['key']!==null){
			$credentials = new Credentials($config['key'],$config['secret']);
		}

		$this->client = new S3Client(
			array(
				'region' => $config['region'],
				'version' => '2006-03-01',
				'signature_version' => 'v4',
				'credentials' => $credentials,
				'endpoint' => $this->base_url,
				'use_path_style_endpoint' => ($this->base_url != NULL),
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
				'Body' => GuzzleHttp\Psr7\stream_for($data->getImageBlob()),
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
				));
		fclose($fp);
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
				));
		// delete
		$this->client->deleteObject(
			array(
				'Bucket' => $this->bucket,
				'Key' => $srckey,
				));
	}

	public function delete($key)
	{
		$this->client->deleteObject(
			array(
				'Bucket' => $this->bucket,
				'Key' => $key,
				));
	}

	public function url($key,$expires=null,$filename=null)
	{
		$bucket = $this->bucket;
		if($expires===null){
			return $this->client->getObjectUrl($this->bucket, $key);
		}

		$param = array('Bucket' => $this->bucket,'Key' => $key);
		if($filename!==null){
			$name = urlencode($filename);
			$param['ResponseContentDisposition'] = "attachment; filename*=UTF-8''{$name}";
		}
		$cmd = $this->client->getCommand('GetObject', $param);
		$obj_url = $this->client->createPresignedRequest($cmd, $expires)->getUri();
		if($this->external_url){
			$obj_url = str_replace($this->base_url, $this->external_url, $obj_url);
		}
		return $obj_url;
	}
}
