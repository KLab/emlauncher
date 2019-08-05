<?php
require_once APP_ROOT.'/model/Application.php';

class apiActions extends MainActions
{
	protected $app = null;

	public function initialize()
	{
		if(($err=parent::initialize())){
			return $err;
		}

		// API Keyによる認証
		$api_key = mfwRequest::param('api_key');
		$this->app = ApplicationDb::selectByApiKey($api_key);
		if(!$this->app){
			return $this->jsonResponse(
				self::HTTP_400_BADREQUEST,
				array('error'=>'Invalid api_key'));
		}

		apache_log('app_id',$this->app->getId());
		return null;
	}

	public function executeDefaultAction()
	{
		return $this->jsonResponse(
			self::HTTP_404_NOTFOUND,
			array('error'=>'404 Not Found'));
	}

	protected static function parseBool($param){
		if(!$param){
			return false;
		}
		if(in_array($param, array("False","false","F","f","No","no","Null","null"))){
			return false;
		}
		return true;
	}

	protected static function makePackageArray(Package $pkg)
	{
		$tags = array();
		foreach($pkg->getTags() as $t){
			$tags[] = $t->getName();
		}

		$attached_files = array();
		foreach($pkg->getAttachedFiles() as $afile){
			$attached_files[] = self::makeAttachedFileArray($afile);
		}

		return array(
			'package_url' => mfwRequest::makeUrl("/package?id={$pkg->getId()}"),
			'application_url' => mfwRequest::makeUrl("/app?id={$pkg->getAppId()}"),
			'id' => $pkg->getId(),
			'platform' => $pkg->getPlatform(),
			'title' => $pkg->getTitle(),
			'description' => $pkg->getDescription(),
			'identifier' => $pkg->getIdentifier(),
			'original_file_name' => $pkg->getOriginalFileName(),
			'file_size' => $pkg->getFileSize(),
			'protect' => $pkg->isProtected(),
			'created' => $pkg->getCreated(),
			'tags' => $tags,
			'install_count' => $pkg->getInstallCount(),
			'attached_files' => $attached_files,
			);
	}

	protected static function makeAttachedFileArray(AttachedFile $attached)
	{
		return array(
			'id' => $attached->getId(),
			'original_file_name' => $attached->getOriginalFileName(),
			'file_size' => $attached->getFileSize(),
			'created' => $attached->getCreated(),
			);
	}
}
