<?php
require_once APP_ROOT.'/model/Config.php';

class LocalFile implements StorageImpl {

	protected $path;
	protected $urlprefix;

	public function __construct()
	{
		$config = Config::get('local_file');
		$this->path = rtrim($config['path'], '/');
		$this->urlprefix = rtrim($config['url_prefix'], '/');
	}

	private function genPath($key)
	{
		return implode('/', array($this->path, trim($key, '/')));
	}

	private function createDir($dir)
	{
		if(!file_exists($dir)){
			mkdir($dir, 0755, true);
		}
	}

	public function saveIcon($key, $data)
	{
		$dstfile = $this->genPath($key);
		$this->createDir(dirname($dstfile));
		return file_put_contents($dstfile, $data);
	}

	public function saveFile($key, $filename, $mime)
	{
		$dstfile = $this->genPath($key);
		$this->createDir(dirname($dstfile));
		copy($filename, $dstfile);
	}

	public function rename($srckey, $dstkey)
	{
		$srcfile = $this->genPath($srckey);
		$dstfile = $this->genPath($dstkey);
		$this->createDir(dirname($dstfile));
		rename($srcfile, $dstfile);
	}

	public function delete($key)
	{
		unlink($this->genPath($key));
	}

	public function url($key, $expire=null, $filename=null)
	{
		return implode('/', array($this->urlprefix, trim($key, '/')));
	}
}
