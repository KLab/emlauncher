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
		$api_key = self::getRequestApiKey();
		$this->app = ApplicationDb::selectByApiKey($api_key);
		if(!$this->app){
			$error = 'Invalid api_key';
			if($api_key===null && !self::isApiKeyParamEnabled() && mfwRequest::has('api_key')){
				$error = 'api_key request parameter is disabled. Use the Authorization: Bearer header.';
			}
			return $this->jsonResponse(
				self::HTTP_400_BADREQUEST,
				array('error'=>$error));
		}

		apache_log('app_id',$this->app->getId());
		return null;
	}

	/**
	 * リクエストからAPI Keyを取り出す.
	 * `Authorization: Bearer {api_key}` ヘッダを優先し、
	 * なければリクエストパラメータ api_key (GET/POST) を使う.
	 * URLのクエリはアクセスログ等に残るため、ヘッダでの指定を推奨する.
	 * 設定 api_key_param_enabled が false の場合、パラメータは使わない.
	 * @return string|null API Key. 指定がなければnull.
	 */
	protected static function getRequestApiKey()
	{
		foreach(mfwRequest::allHeaders() as $name => $value){
			if(strcasecmp($name,'Authorization')===0
				&& preg_match('/^Bearer +(\S+) *$/i',$value,$m)){
				return $m[1];
			}
		}
		if(!self::isApiKeyParamEnabled()){
			return null;
		}
		return mfwRequest::param('api_key');
	}

	/**
	 * リクエストパラメータ api_key での認証を許可するか.
	 * 設定 api_key_param_enabled が未設定の場合は許可する (後方互換).
	 */
	protected static function isApiKeyParamEnabled()
	{
		return Config::get('api_key_param_enabled')!==false;
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

	protected static function makeAttachedFileLinkArray(AttachedFile $attached, string $url, DateTime $expire_at)
	{
		return array(
			'attached_file' => self::makeAttachedFileArray($attached),
			'download_url' => $url,
			'expire' => $expire_at->format('Y-m-d H:i:s'),
			);
	}
}
